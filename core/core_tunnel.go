package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/base/maps/wait"
	"github.com/injoyai/base/safe"
	"github.com/injoyai/conv"
	uuid "github.com/satori/go.uuid"
	"io"
	"sync/atomic"
	"time"
)

type OptionTunnel func(v *Tunnel)

func WithKey(k string) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.k = k
	}
}

func WithFrame(f Frame) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.f = f
	}
}

func WithWait(timeout time.Duration) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.Wait = wait.New(timeout)
	}
}

// WithRegister 当客户端进行注册,处理校验客户端的信息
func WithRegister(f func(v *Tunnel, p Packet) (interface{}, error)) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.OnRegister = f
	}
}

// WithDialed 请求建立连接成功事件
func WithDialed(f func(d *Dial, key string)) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.OnDialed = f
	}
}

func WithDial(f func(d *Dial) (io.ReadWriteCloser, string, error)) func(v *Tunnel) {
	if f == nil {
		return WithDialRemote()
	}
	return func(v *Tunnel) { v.OnDial = f }
}

func WithDialTCP(address string, timeout ...time.Duration) func(v *Tunnel) {
	return WithDialCustom(&Dial{
		Type:    "tcp",
		Address: address,
		Timeout: conv.DefaultDuration(0, timeout...),
	})
}

// WithDialCustom 使用自定义代理配置进行代理,忽略服务端的配置
func WithDialCustom(proxy *Dial) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.OnDial = func(d *Dial) (io.ReadWriteCloser, string, error) {
			*d = *proxy
			return proxy.Dial()
		}
	}
}

// WithDialRemote 使用服务端的代理配置进行代理
func WithDialRemote() func(v *Tunnel) {
	return WithDial(func(d *Dial) (io.ReadWriteCloser, string, error) {
		return d.Dial()
	})
}

// WithBufferSize 设置复制的buffer大小
func WithBufferSize(size uint) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.copyBufferSize = size
	}
}

// WithRegistered 可以设置跳过注册
func WithRegistered(b ...bool) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.Registered = len(b) > 0 && b[0]
	}
}

func NewTunnel(r io.ReadWriteCloser, option ...OptionTunnel) *Tunnel {
	v := &Tunnel{
		k:              fmt.Sprintf("%p", r),
		f:              DefaultFrame,
		r:              r,
		copyBufferSize: 1024 * 32, //这里太小会有bug,界面会卡在那里,可能是丢失数据的情况
		Tag:            maps.NewSafe(),
		Virtual:        maps.NewSafe(),
		Wait:           wait.New(time.Second * 5),
		Closer:         safe.NewCloser(),
	}
	v.Closer.SetCloseFunc(func(err error) error {
		v.Virtual.Range(func(key, value interface{}) bool {
			value.(*Virtual).Close()
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

// Tunnel 隧道,虚拟设备管理,收到的数据自动转发到对应的IO
type Tunnel struct {
	k              string             //唯一标识
	f              Frame              //传输协议
	r              io.ReadWriteCloser //实际IO,
	running        uint32             //运行状态,内部字段,不能修改
	copyBufferSize uint               //复制的buffer大小

	*safe.Closer              //安全关闭
	Tag          *maps.Safe   //记录一些信息
	Virtual      *maps.Safe   //虚拟IO管理
	Wait         *wait.Entity //异步等待机制
	Registered   bool         //是否已经注册,未注册的需要先注册

	OnRegister func(v *Tunnel, p Packet) (interface{}, error)    //注册事件,能校验注册信息
	OnDial     func(d *Dial) (io.ReadWriteCloser, string, error) //新建连接事件,指定连接
	OnDialed   func(d *Dial, key string)                         //连接成功事件
	OnRequest  func(p []byte) ([]byte, error)                    //copy的请求数据
	OnResponse func(p []byte) ([]byte, error)                    //copy的响应数据
}

func (this *Tunnel) Key() string {
	return this.k
}

func (this *Tunnel) SetKey(k string) {
	this.k = k
}

func (this *Tunnel) SetOption(op ...OptionTunnel) {
	for _, f := range op {
		f(this)
	}
}

// WritePacket 发送数据包到虚拟IO,mid(消息ID),t(消息类型),i(消息内容)
func (this *Tunnel) WritePacket(mid string, t byte, i interface{}) error {
	p := this.f.NewPacket(mid, t, i)
	DefaultLog.Write(p)
	_, err := this.r.Write(p.Bytes())
	return err
}

// Register 进行注册操作,等待注册结果
func (this *Tunnel) Register(data interface{}) (interface{}, error) {
	if err := this.WritePacket(this.Key(), Request|Register|NeedAck, data); err != nil {
		return nil, err
	}
	return this.Wait.Wait(this.Key())
}

// Dial 建立代理连接(另一端想请求的信息)，
// @mid 是消息唯一标识，用来确定消息响应，为空的话自动生成
// @dial 代理的链接信息,会赋值为实际的链接信息
// @closer 就是虚拟IO关闭事件，可以为nil
func (this *Tunnel) Dial(mid string, dial *Dial, closer io.Closer) (io.ReadWriteCloser, error) {
	if len(mid) == 0 {
		mid = uuid.NewV4().String()
	}
	if err := this.WritePacket(mid, Open|Request|NeedAck, dial); err != nil {
		return nil, err
	}
	val, err := this.Wait.Wait(mid)
	if err != nil {
		return nil, err
	}
	res := new(DialRes)
	if err := json.Unmarshal([]byte(val.(string)), res); err != nil {
		return nil, err
	}
	*dial = *res.Dial
	//NewIO已缓存IO
	return this.NewVirtual(res.Key, closer), nil
}

// DialAndSwap 建立代理连接(另一端想请求的信息),然后进行数据交互
// @mid 是消息唯一标识，用来确定消息响应，为空的话自动生成
// @dial 代理的链接信息,会赋值为实际的链接信息
// @c 就是IO,用来交互数据
func (this *Tunnel) DialAndSwap(mid string, dial *Dial, c io.ReadWriteCloser) error {
	defer c.Close()
	i, err := this.Dial(mid, dial, nil)
	if err != nil {
		return err
	}
	defer i.Close()
	go io.Copy(c, i)
	_, err = io.Copy(i, c)
	return err
}

//// WriteTo 写入到指定虚拟IO,不过这个key可不好获取，除非字节通过NewVirtual生成
//func (this *Tunnel) WriteTo(key string, p []byte) error {
//	i := this.GetVirtual(key)
//	if i != nil {
//		_, err := i.Write(p)
//		return err
//	}
//	return errors.New("use closed io")
//}

// GetVirtual 获取虚拟IO实例
func (this *Tunnel) GetVirtual(key string) *Virtual {
	v := this.Virtual.MustGet(key)
	if v != nil {
		return v.(*Virtual)
	}
	return nil
}

// NewVirtual 新建个虚拟IO,closer就是关闭虚拟IO的事件
// Writer就是隧道，Reader是新建的，数据是通过Virtual.ToBuffer加入
func (this *Tunnel) NewVirtual(key string, closer io.Closer) *Virtual {
	v := NewVirtual(this.r, chans.NewIO(20),
		func(v *Virtual) {
			v.OnWrite = func(bs []byte) ([]byte, error) {
				p := this.f.NewPacket(key, Write, bs)
				DefaultLog.Write(p)
				return p.Bytes(), nil
			}
			v.OnClose = func(v *Virtual, err error) error {
				//发送至隧道,通知隧道另一端
				this.WritePacket(key, Close, err)
				//从缓存中移除
				this.Virtual.Del(key)
				//关闭客户端
				if closer != nil {
					closer.Close()
				}
				return nil
			}
		},
	)
	this.Virtual.Set(key, v)
	return v
}

// Run 开始监听数据，并通过设置的协议进行解析
func (this *Tunnel) Run() (err error) {

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
		DefaultLog.Read(p)

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
						this.Wait.Done(p.GetMsgID(), string(p.GetData()))
					} else {
						this.Wait.Done(p.GetMsgID(), errors.New(string(p.GetData())))
					}

				}

			case Read:

				if p.IsRequest() {
					i := this.GetVirtual(p.GetMsgID())
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
					i := this.GetVirtual(p.GetMsgID())
					if i != nil {

					}

				}

			case Write:

				if p.IsRequest() {
					i := this.GetVirtual(p.GetMsgID())
					if i != nil {
						err = i.ToRead(p.GetData())
						return conv.Bytes(uint32(len(p.GetData()))), err
					} else {
						//当A还没意识到B已关闭,发送数据的话,会收到远程意外关闭连接的错误
						return nil, errors.New("远程意外关闭连接")
					}

				} else {
					if p.Success() {
						this.Wait.Done(p.GetMsgID()+".read", conv.Uint32(p.GetData()))
					} else {
						this.Wait.Done(p.GetMsgID()+".write", errors.New(string(p.GetData())))
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
					d := &Dial{
						Type:    m.GetString("type", "tcp"),
						Address: m.GetString("address"),
						Timeout: m.GetDuration("timeout"),
						Param:   gm,
					}
					c, key, err := this.OnDial(d)
					if err != nil {
						return nil, err
					}
					if this.OnDialed != nil {
						this.OnDialed(d, key)
					}
					//1. 如果p.Get重复怎么处理,请求的是临时key,一般用uuid
					//2. 是否需要读写分离,这样错误能对应上,或者被动read,下发read
					//	2.1 能分离错误信息
					//	2.2 避免阻塞,现在Chan的容量有限,读的数据没处理的话,或越来越多
					//  2.3 性能会下降,多了一次IO

					//新建虚拟IO
					i := this.NewVirtual(key, c)
					go func() {
						defer func() {
							DefaultLog.Tracef("[%s] 关闭连接,%v\n", key, i.Err())
							c.Close()
							i.Close()
						}()

						go func() {
							err = CopyBufferWith(c, i, make([]byte, this.copyBufferSize), this.OnResponse)
							DefaultLog.PrintErr(err)
							i.CloseWithErr(err)
						}()
						err = CopyBufferWith(i, c, make([]byte, this.copyBufferSize), this.OnResponse)
						DefaultLog.PrintErr(err)
						i.CloseWithErr(err)
					}()

					//响应成功,并返回唯一标识
					return DialRes{Key: key, Dial: d}, nil

				} else {
					//响应
					if p.Success() {
						this.Wait.Done(p.GetMsgID(), string(p.GetData()))
					} else {
						this.Wait.Done(p.GetMsgID(), errors.New(string(p.GetData())))
					}

				}

			case Close:

				//当IO收到关闭信息试时候
				if p.IsRequest() {
					i := this.GetVirtual(p.GetMsgID())
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
				err = this.WritePacket(p.GetMsgID(), p.GetType()|Response|Fail, err)
				DefaultLog.PrintErr(err)
			} else {
				err = this.WritePacket(p.GetMsgID(), p.GetType()|Response|Success, data)
				DefaultLog.PrintErr(err)
			}
		}

	}
}
