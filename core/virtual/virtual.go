package virtual

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/base/maps/wait"
	"github.com/injoyai/base/safe"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"io"
	"time"
)

type Option func(v *Virtual)

func New(r io.ReadWriteCloser, option ...Option) *Virtual {
	v := &Virtual{
		k:      fmt.Sprintf("%p", r),
		f:      DefaultFrame,
		r:      r,
		IO:     maps.NewSafe(),
		Wait:   wait.New(time.Second * 5),
		Closer: safe.NewCloser(),
	}
	v.Closer.SetCloseFunc(func(err error) error {
		v.IO.Range(func(key, value interface{}) bool {
			value.(*IO).Close()
			return true
		})
		return v.r.Close()
	})
	//默认使用远程的代理配置
	WithOpenRemote()(v)
	//自定义选项
	for _, op := range option {
		op(v)
	}
	return v
}

func WithKey(k string) func(v *Virtual) {
	return func(v *Virtual) {
		v.k = k
	}
}

func WithFrame(f Frame) func(v *Virtual) {
	return func(v *Virtual) {
		v.f = f
	}
}

func WithWait(timeout time.Duration) func(v *Virtual) {
	return func(v *Virtual) {
		v.Wait = wait.New(timeout)
	}
}

// WithRegister 当客户端进行注册,处理校验客户端的信息
func WithRegister(f func(v *Virtual, p Packet) (interface{}, error)) func(v *Virtual) {
	return func(v *Virtual) {
		v.OnRegister = f
	}
}

// WithOpened 请求建立连接成功事件
func WithOpened(f func(p Packet, d *core.Dial, key string)) func(v *Virtual) {
	return func(v *Virtual) {
		v.OnOpened = f
	}
}

func WithOpen(f func(p Packet, d *core.Dial) (io.ReadWriteCloser, string, error)) func(v *Virtual) {
	if f == nil {
		return WithOpenRemote()
	}
	return func(v *Virtual) { v.open = f }
}

func WithOpenTCP(address string, timeout ...time.Duration) func(v *Virtual) {
	return WithOpenCustom(&core.Dial{
		Type:    "tcp",
		Address: address,
		Timeout: conv.DefaultDuration(0, timeout...),
	})
}

// WithOpenCustom 使用自定义代理配置进行代理,忽略服务端的配置
func WithOpenCustom(proxy *core.Dial) func(v *Virtual) {
	return func(v *Virtual) {
		v.open = func(p Packet, d *core.Dial) (io.ReadWriteCloser, string, error) {
			return proxy.Dial()
		}
	}
}

// WithOpenRemote 使用服务端的代理配置进行代理
func WithOpenRemote() func(v *Virtual) {
	return WithOpen(func(p Packet, d *core.Dial) (io.ReadWriteCloser, string, error) {
		return d.Dial()
	})
}

// Virtual 虚拟设备管理,收到的数据自动转发到对应的IO
type Virtual struct {
	k    string
	f    Frame
	r    io.ReadWriteCloser
	IO   *maps.Safe
	Wait *wait.Entity
	*safe.Closer

	open       func(p Packet, d *core.Dial) (io.ReadWriteCloser, string, error)
	OnRegister func(v *Virtual, p Packet) (interface{}, error)
	OnOpened   func(p Packet, d *core.Dial, key string)
}

func (this *Virtual) Key() string {
	return this.k
}

func (this *Virtual) SetKey(k string) {
	this.k = k
}

func (this *Virtual) SetOption(op ...Option) {
	for _, f := range op {
		f(this)
	}
}

// WritePacket 发送数据包到虚拟IO
func (this *Virtual) WritePacket(k string, t byte, i interface{}) error {
	p := this.NewPacket(k, t, i)
	_, err := this.r.Write(p.Bytes())
	return err
}

func (this *Virtual) NewPacket(k string, t byte, i interface{}) Packet {
	p := this.f.NewPacket(k, t, i)
	logs.Write(p)
	return p
}

func (this *Virtual) Register(data interface{}) (interface{}, error) {
	if err := this.WritePacket(this.Key(), Request|Register|NeedAck, data); err != nil {
		return nil, err
	}
	return this.Wait.Wait(this.Key())
}

func (this *Virtual) Open(k string, p *core.Dial, closer io.Closer) (io.ReadWriteCloser, error) {
	if len(k) == 0 {
		k = g.UUID()
	}
	if err := this.WritePacket(k, Open|Request|NeedAck, p); err != nil {
		return nil, err
	}
	val, err := this.Wait.Wait(k)
	if err != nil {
		return nil, err
	}
	//NewIO已缓存IO
	return this.NewIO(val.(string), closer), nil
}

func (this *Virtual) OpenAndSwap(k string, p *core.Dial, c io.ReadWriteCloser) error {
	defer c.Close()
	i, err := this.Open(k, p, c)
	if err != nil {
		return err
	}
	defer i.Close()
	go io.Copy(c, i)
	_, err = io.Copy(i, c)
	return err
}

//func (this *Virtual)DialWithTimeout(address string, timeout time.Duration) (io.ReadWriteCloser, error)

func (this *Virtual) WriteTo(key string, p []byte) error {
	i := this.GetIO(key)
	if i != nil {
		_, err := i.Write(p)
		return err
	}
	return errors.New("use closed io")
}

func (this *Virtual) GetIO(key string) *IO {
	v := this.IO.MustGet(key)
	if v != nil {
		return v.(*IO)
	}
	return nil
}

func (this *Virtual) NewIO(key string, closer io.Closer) *IO {
	i := NewIO(key, this.r, NewBuffer(20),
		func(bs []byte) ([]byte, error) {
			p := this.NewPacket(key, Write, bs)
			return p.Bytes(), nil
		}, func(v *IO, err error) error {
			//发送至隧道,通知隧道另一端
			this.WritePacket(key, Close, err)
			//从缓存中移除
			this.IO.Del(key)
			//关闭客户端
			closer.Close()
			return nil
		},
	)
	this.IO.Set(key, i)
	return i
}

func (this *Virtual) Run() (err error) {
	defer func() { this.CloseWithErr(err) }()
	buf := bufio.NewReader(this.r)
	for {
		//按照协议去读取数据
		p, err := this.f.ReadPacket(buf)
		if err != nil {
			return err
		}
		logs.Read(p)

		//处理代理数据
		data, err := func() (interface{}, error) {

			switch p.GetType() {

			case Register:

				if p.IsRequest() {
					if this.OnRegister != nil {
						return this.OnRegister(this, p)
					}

				} else {
					if p.Success() {
						this.Wait.Done(p.GetKey(), string(p.GetData()))
					} else {
						this.Wait.Done(p.GetKey(), errors.New(string(p.GetData())))
					}

				}

			case Read:

				if p.IsRequest() {
					i := this.GetIO(p.GetKey())
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
					i := this.GetIO(p.GetKey())
					if i != nil {

					}

				}

			case Write:

				if p.IsRequest() {
					i := this.GetIO(p.GetKey())
					if i != nil {
						err = i.ToBuffer(p.GetData())
						return conv.Bytes(uint32(len(p.GetData()))), err
					} else {
						//当A还没意识到B已关闭,发送数据的话,会收到远程意外关闭连接的错误
						return nil, errors.New("远程意外关闭连接")
					}

				} else {
					if p.Success() {
						this.Wait.Done(p.GetKey()+".read", conv.Uint32(p.GetData()))
					} else {
						this.Wait.Done(p.GetKey()+".write", errors.New(string(p.GetData())))
					}

				}

			case Open:

				if p.IsRequest() {
					if this.open == nil {
						return nil, errors.New("open is nil")
					}
					m := conv.NewMap(p.GetData())
					gm := m.GMap()
					delete(gm, "type")
					delete(gm, "address")
					delete(gm, "timeout")
					d := &core.Dial{
						Type:    m.GetString("type", "tcp"),
						Address: m.GetString("address"),
						Timeout: m.GetDuration("timeout"),
						Param:   gm,
					}
					c, key, err := this.open(p, d)
					if err != nil {
						return nil, err
					}
					if this.OnOpened != nil {
						this.OnOpened(p, d, key)
					}
					//1. 如果p.Get重复怎么处理,请求的是临时key,一般用uuid
					//2. 是否需要读写分离,这样错误能对应上,或者被动read,下发read
					//	2.1 能分离错误信息
					//	2.2 避免阻塞,现在Chan的容量有限,读的数据没处理的话,或越来越多
					//  2.3 性能会下降,多了一次IO

					//新建虚拟IO
					i := this.NewIO(key, c)
					go func() {
						defer func() {
							logs.Tracef("[%s] 关闭连接,%v\n", key, i.Err())
							c.Close()
							i.Close()
						}()

						go func() {
							_, err := io.Copy(c, i)
							logs.PrintErr(err)
							i.CloseWithErr(err)
						}()
						_, err := io.Copy(i, c)
						logs.PrintErr(err)
						i.CloseWithErr(err)
					}()

					//响应成功,并返回唯一标识
					return key, nil

				} else {
					//响应
					if p.Success() {
						this.Wait.Done(p.GetKey(), string(p.GetData()))
					} else {
						this.Wait.Done(p.GetKey(), errors.New(string(p.GetData())))
					}

				}

			case Close:

				//当IO收到关闭信息试时候
				if p.IsRequest() {
					i := this.GetIO(p.GetKey())
					if i != nil {
						errMsg := string(p.GetData())
						if len(errMsg) == 0 {
							errMsg = io.EOF.Error()
						}
						i.CloseWithErr(errors.New(errMsg))
					}
				}

			}

			return nil, nil
		}()

		if p.IsRequest() && p.NeedAck() {
			if err != nil {
				err = this.WritePacket(p.GetKey(), p.GetType()|Response|Fail, err)
				logs.PrintErr(err)
			} else {
				err = this.WritePacket(p.GetKey(), p.GetType()|Response|Success, data)
				logs.PrintErr(err)
			}
		}

	}
}
