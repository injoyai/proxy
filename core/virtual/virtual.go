package virtual

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/base/maps/wait"
	"github.com/injoyai/base/safe"
	"github.com/injoyai/conv"
	"github.com/injoyai/proxy/core"
	uuid "github.com/satori/go.uuid"
	"io"
	"sync/atomic"
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
	WithDialRemote()(v)
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

// WithDialed 请求建立连接成功事件
func WithDialed(f func(p Packet, d *core.Dial, key string)) func(v *Virtual) {
	return func(v *Virtual) {
		v.OnDialed = f
	}
}

func WithDial(f func(p Packet, d *core.Dial) (io.ReadWriteCloser, string, error)) func(v *Virtual) {
	if f == nil {
		return WithDialRemote()
	}
	return func(v *Virtual) { v.OnDial = f }
}

func WithDialTCP(address string, timeout ...time.Duration) func(v *Virtual) {
	return WithDialCustom(&core.Dial{
		Type:    "tcp",
		Address: address,
		Timeout: conv.DefaultDuration(0, timeout...),
	})
}

// WithDialCustom 使用自定义代理配置进行代理,忽略服务端的配置
func WithDialCustom(proxy *core.Dial) func(v *Virtual) {
	return func(v *Virtual) {
		v.OnDial = func(p Packet, d *core.Dial) (io.ReadWriteCloser, string, error) {
			*d = *proxy
			return proxy.Dial()
		}
	}
}

// WithDialRemote 使用服务端的代理配置进行代理
func WithDialRemote() func(v *Virtual) {
	return WithDial(func(p Packet, d *core.Dial) (io.ReadWriteCloser, string, error) {
		return d.Dial()
	})
}

// WithBufferSize 设置复制的buffer大小
func WithBufferSize(size uint) func(v *Virtual) {
	return func(v *Virtual) {
		v.copyBufferSize = size
	}
}

// WithRegistered 可以设置跳过注册
func WithRegistered(b ...bool) func(v *Virtual) {
	return func(v *Virtual) {
		v.Registered = len(b) > 0 && b[0]
	}
}

// Virtual 虚拟设备管理,收到的数据自动转发到对应的IO
type Virtual struct {
	k              string             //唯一标识
	f              Frame              //传输协议
	r              io.ReadWriteCloser //实际IO,
	running        uint32             //运行状态,内部字段,不能修改
	copyBufferSize uint               //复制的buffer大小

	*safe.Closer              //安全关闭
	IO           *maps.Safe   //虚拟IO管理
	Wait         *wait.Entity //异步等待机制
	Registered   bool         //是否已经注册,未注册的需要先注册

	OnRegister func(v *Virtual, p Packet) (interface{}, error)                  //注册事件,能校验注册信息
	OnDial     func(p Packet, d *core.Dial) (io.ReadWriteCloser, string, error) //新建连接事件,指定连接
	OnDialed   func(p Packet, d *core.Dial, key string)                         //连接成功事件
	OnRequest  func(p []byte) ([]byte, error)                                   //copy的请求数据
	OnResponse func(p []byte) ([]byte, error)                                   //copy的响应数据
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

// WritePacket 发送数据包到虚拟IO,k(消息ID),t(消息类型),i(消息内容)
func (this *Virtual) WritePacket(k string, t byte, i interface{}) error {
	p := this.NewPacket(k, t, i)
	_, err := this.r.Write(p.Bytes())
	return err
}

// NewPacket 新建个协议数据 k(消息ID),t(消息类型),i(消息内容)
func (this *Virtual) NewPacket(k string, t byte, i interface{}) Packet {
	p := this.f.NewPacket(k, t, i)
	core.DefaultLog.Write(p)
	return p
}

// Register 进行注册操作,等待注册结果
func (this *Virtual) Register(data interface{}) (interface{}, error) {
	if err := this.WritePacket(this.Key(), Request|Register|NeedAck, data); err != nil {
		return nil, err
	}
	return this.Wait.Wait(this.Key())
}

// Dial 建立代理连接(另一端想请求的信息)
func (this *Virtual) Dial(k string, p *core.Dial, closer io.Closer) (io.ReadWriteCloser, error) {
	if len(k) == 0 {
		k = uuid.NewV4().String()
	}
	if closer == nil {
		closer = io.NopCloser(nil)
	}
	if err := this.WritePacket(k, Open|Request|NeedAck, p); err != nil {
		return nil, err
	}
	val, err := this.Wait.Wait(k)
	if err != nil {
		return nil, err
	}
	res := new(core.DialRes)
	if err := json.Unmarshal([]byte(val.(string)), res); err != nil {
		return nil, err
	}
	*p = *res.Dial
	//NewIO已缓存IO
	return this.NewIO(res.Key, closer), nil
}

// DialAndSwap 建立代理连接(另一端想请求的信息),然后进行数据交互
func (this *Virtual) DialAndSwap(k string, p *core.Dial, c io.ReadWriteCloser) error {
	defer c.Close()
	i, err := this.Dial(k, p, c)
	if err != nil {
		return err
	}
	defer i.Close()
	go io.Copy(c, i)
	_, err = io.Copy(i, c)
	return err
}

// WriteTo 写入到指定虚拟IO
func (this *Virtual) WriteTo(key string, p []byte) error {
	i := this.GetIO(key)
	if i != nil {
		_, err := i.Write(p)
		return err
	}
	return errors.New("use closed io")
}

// GetIO 获取虚拟IO实例
func (this *Virtual) GetIO(key string) *IO {
	v := this.IO.MustGet(key)
	if v != nil {
		return v.(*IO)
	}
	return nil
}

// NewIO 新建个虚拟IO
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

	if !atomic.CompareAndSwapUint32(&this.running, 0, 1) {
		<-this.Done()
		return this.Err()
	}

	defer func() {
		this.CloseWithErr(err)
		atomic.StoreUint32(&this.running, 0)
	}()
	buf := bufio.NewReader(this.r)
	for {
		//按照协议去读取数据
		p, err := this.f.ReadPacket(buf)
		if err != nil {
			return err
		}
		core.DefaultLog.Read(p)

		//处理代理数据
		data, err := func() (interface{}, error) {

			if !this.Registered && p.GetType() != Register && p.IsRequest() {
				//想跳过注册进行数据交互的操作,全部请求数据返回错误
				return nil, errors.New("未注册")
			}

			switch p.GetType() {

			case Register:

				if p.IsRequest() {
					if this.OnRegister != nil {
						res, err := this.OnRegister(this, p)
						if err == nil {
							//客户端请求注册成功
							this.Registered = true
						}
						return res, err
					}

				} else {
					if p.Success() {
						//注册到服务端成功
						this.Registered = true
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
					if this.OnDial == nil {
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
					c, key, err := this.OnDial(p, d)
					if err != nil {
						return nil, err
					}
					if this.OnDialed != nil {
						this.OnDialed(p, d, key)
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
							core.DefaultLog.Tracef("[%s] 关闭连接,%v\n", key, i.Err())
							c.Close()
							i.Close()
						}()

						go func() {
							//_, err := io.Copy(c, i)
							err = core.CopyBufferWith(i, c, make([]byte, this.copyBufferSize), func(p []byte) ([]byte, error) {
								if this.OnRequest != nil {
									return this.OnRequest(p)
								}
								return p, nil
							})
							core.DefaultLog.PrintErr(err)
							i.CloseWithErr(err)
						}()
						//_, err := io.Copy(i, c)
						err = core.CopyBufferWith(c, i, make([]byte, this.copyBufferSize), func(p []byte) ([]byte, error) {
							if this.OnResponse != nil {
								return this.OnResponse(p)
							}
							return p, nil
						})
						core.DefaultLog.PrintErr(err)
						i.CloseWithErr(err)
					}()

					//响应成功,并返回唯一标识
					return core.DialRes{Key: key, Dial: d}, nil

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
				core.DefaultLog.PrintErr(err)
			} else {
				err = this.WritePacket(p.GetKey(), p.GetType()|Response|Success, data)
				core.DefaultLog.PrintErr(err)
			}
		}

	}
}
