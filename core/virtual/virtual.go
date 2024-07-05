package virtual

import (
	"bufio"
	"errors"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/base/maps/wait"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/logs"
	"io"
	"net"
	"time"
)

func NewTCPDefault(r io.ReadWriteCloser) *Virtual {
	return New(r, DefaultFrame, func(p Packet) (io.ReadWriteCloser, string, error) {
		c, err := net.Dial("tcp", string(p.GetData()))
		if err != nil {
			return nil, "", err
		}
		return c, c.LocalAddr().String(), nil
	})
}

func New(r io.ReadWriteCloser, f Frame, open func(p Packet) (io.ReadWriteCloser, string, error)) *Virtual {

	v := &Virtual{
		f:    f,
		r:    r,
		Safe: maps.NewSafe(),
		wait: wait.New(time.Second * 5),
	}

	go func() {
		buf := bufio.NewReader(r)
		for {
			//按照协议去读取数据
			p, err := f.ReadPacket(buf)
			if err != nil {
				return
			}

			data, err := func() (interface{}, error) {

				switch p.GetType() {

				case Read:

					if p.IsRequest() {
						i := v.GetIO(p.GetKey())
						if i != nil {
							bs := make([]byte, conv.Uint32(p.GetData()))
							n, err := i.Read(bs)
							if err != nil {
								return nil, err
							}
							return bs[:n], nil
						} else {
							//当A还没意识到B已关闭,发送数据的话,会收到远程意外关闭连接的错误
							return nil, errors.New("远程意外关闭连接")
						}

					} else {
						i := v.GetIO(p.GetKey())
						if i != nil {

						}

					}

				case Write:

					if p.IsRequest() {
						logs.Debug(p)
						i := v.GetIO(p.GetKey())
						if i != nil {
							err = i.ToBuffer(p.GetData())
							//n, err := i.Write(p.GetData())
							return conv.Bytes(uint32(len(p.GetData()))), err
						} else {
							//当A还没意识到B已关闭,发送数据的话,会收到远程意外关闭连接的错误
							return nil, errors.New("远程意外关闭连接")
						}

					} else {
						if p.Success() {
							v.wait.Done(p.GetKey()+".read", conv.Uint32(p.GetData()))
						} else {
							v.wait.Done(p.GetKey()+".write", errors.New(string(p.GetData())))
						}

					}

				case Open:

					if p.IsRequest() {
						c, key, err := open(p)
						if err != nil {
							return nil, err
						}
						//1. 如果p.Get重复怎么处理,请求的是临时key,一般用uuid
						//2. 是否需要读写分离,这样错误能对应上,或者被动read,下发read
						//	2.1 能分离错误信息
						//	2.2 避免阻塞,现在Chan的容量有限,读的数据没处理的话,或越来越多
						//  2.3 性能会下降,多了一次IO

						//新建虚拟IO
						i := v.NewIO(key)
						go func() {
							defer c.Close()
							go func() {
								_, err := io.Copy(c, i)
								logs.Err(err)
								i.CloseWithErr(err)
							}()
							_, err := io.Copy(i, c)
							logs.Err(err)
							i.CloseWithErr(err)
						}()

						//响应成功,并返回唯一标识
						return key, nil

					} else {
						logs.Debug(p)
						//响应
						if p.Success() {
							v.wait.Done(p.GetKey(), string(p.GetData()))
						} else {
							v.wait.Done(p.GetKey(), errors.New(string(p.GetData())))
						}

					}

				case Close:

					//当IO收到关闭信息试时候
					i := v.GetIO(p.GetKey())
					if i != nil {
						errMsg := string(p.GetData())
						if len(errMsg) == 0 {
							errMsg = io.EOF.Error()
						}
						i.CloseWithErr(errors.New(errMsg))
					}

				}

				return nil, nil
			}()

			if p.IsRequest() && p.NeedAck() {
				if err != nil {
					v.WritePacket(p.GetKey(), p.GetType()|Response|Fail, err)
				} else if p.NeedAck() {
					v.WritePacket(p.GetKey(), p.GetType()|Response|Success, data)
				}
			}

		}
	}()

	return v
}

type Virtual struct {
	f Frame
	r io.ReadWriteCloser
	*maps.Safe
	wait *wait.Entity
}

func (this *Virtual) WritePacket(k string, t byte, i interface{}) error {
	_, err := this.r.Write(this.f.NewPacket(k, t, i).Bytes())
	return err
}

func (this *Virtual) NewPacket(k string, t byte, i interface{}) Packet {
	return this.f.NewPacket(k, t, i)
}

func (this *Virtual) Dial(address string) (io.ReadWriteCloser, error) {
	key := g.UUID() //临时key
	if err := this.WritePacket(key, Open|Request|NeedAck, address); err != nil {
		return nil, err
	}
	val, err := this.wait.Wait(key)
	if err != nil {
		return nil, err
	}
	return this.NewIO(val.(string)), nil
}

//func (this *Virtual)DialWithTimeout(address string, timeout time.Duration) (io.ReadWriteCloser, error)

func (this *Virtual) Publish(key string, p []byte) error {
	i := this.GetIO(key)
	if i != nil {
		_, err := i.Write(p)
		return err
	}
	return errors.New("use closed io")
}

func (this *Virtual) GetIO(key string) *IO {
	v := this.Safe.MustGet(key)
	if v != nil {
		return v.(*IO)
	}
	return nil
}

func (this *Virtual) NewIO(key string) *IO {
	i := NewIO(key, this.r, NewBuffer(20),
		func(bs []byte) ([]byte, error) {
			p := this.NewPacket(key, Write, bs)
			return p.Bytes(), nil
		}, func(v *IO, err error) error {
			//go里面,IO的一方如果正常关闭,那么另一方读写的时候会收到io.EOF的错误
			p := this.NewPacket(key, Close, err)
			this.r.Write(p.Bytes())
			this.Safe.Del(key)
			return nil
		},
	)
	this.Safe.Set(key, i)
	return i
}
