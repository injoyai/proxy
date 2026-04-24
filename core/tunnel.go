// Package core 提供隧道代理的核心功能
package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/injoyai/base/maps/wait"
	"github.com/injoyai/base/safe"
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
)

// NewTunnel 创建一个新的隧道实例
// r 是底层的物理连接,option 是可选的配置函数
// 隧道是虚拟通道的管理器,支持多条虚拟IO复用同一条物理连接
func NewTunnel(r io.ReadWriteCloser, option ...OptionTunnel) *Tunnel {
	v := &Tunnel{
		k:      fmt.Sprintf("%p", r),
		f:      DefaultFrame,
		r:      r,
		ioMap:  map[string]*IO{},
		wait:   wait.New(time.Second * 5),
		Closer: safe.NewCloser(),
		dial:   DefaultDial,
	}
	v.Closer.SetCloseFunc(func(err error) error {
		v.ioMu.Lock()
		for _, c := range v.ioMap {
			c.Close()
		}
		v.ioMu.Unlock()
		return v.r.Close()
	})
	for _, op := range option {
		op(v)
	}
	return v
}

// Tunnel 隧道结构体,是虚拟通道的核心管理器
// 一条隧道对应一条物理连接,可以承载多条虚拟IO
// 隧道负责:
//   - 管理虚拟IO的生命周期
//   - 解析和发送帧协议数据包
//   - 处理注册、连接建立等控制消息
//   - 异步等待请求响应
type Tunnel struct {
	*safe.Closer // Closer 安全关闭控制器

	k          string             // k 隧道唯一标识
	f          Frame              // f 帧协议实例
	r          io.ReadWriteCloser // r 底层物理连接
	ioMu       sync.RWMutex       // ioMu 保护 ioMap 的并发访问
	ioMap      map[string]*IO     // ioMap 虚拟IO映射,key为IO的唯一标识
	wait       *wait.Entity       // wait 异步等待机制,用于等待请求响应
	running    atomic.Bool        // running 隧道是否正在运行
	registered atomic.Bool        // registered 是否已完成注册

	dial       func(d *Dial) (io.ReadWriteCloser, string, error) // dial 拨号函数
	onRegister func(v *Tunnel, p Packet) (interface{}, error)    // onRegister 注册回调
	onDialed   func(d *Dial, key string)                         // onDialed 连接成功回调
}

// Key 获取隧道的唯一标识
func (this *Tunnel) Key() string {
	return this.k
}

// SetKey 设置隧道的唯一标识
func (this *Tunnel) SetKey(k string) {
	this.k = k
}

// SetOption 设置隧道选项
func (this *Tunnel) SetOption(op ...OptionTunnel) {
	for _, f := range op {
		f(this)
	}
}

// WritePacket 发送一个数据包到对端
// msgID 为消息唯一标识,t 为消息类型,i 为消息内容
func (this *Tunnel) WritePacket(msgID string, t byte, i any) error {
	p := this.f.NewPacket(msgID, t, i)
	_, err := this.r.Write(p.Bytes())
	return err
}

// Register 向对端发送注册请求并等待响应
// data 为注册信息,通常为 RegisterReq 结构体
// 返回对端的响应数据
func (this *Tunnel) Register(data any) (any, error) {
	if err := this.WritePacket(this.Key(), Request|Register|NeedAck, data); err != nil {
		return nil, err
	}
	return this.wait.Wait(this.Key())
}

// Dial 向对端发起建立连接的请求
// msgID 为消息唯一标识(为空则自动生成),dial 为目标连接配置,closer 为关闭回调
// 返回一个虚拟IO,可以通过此IO与目标地址进行数据交互
func (this *Tunnel) Dial(msgID string, dial *Dial, closer io.Closer) (io.ReadWriteCloser, error) {
	if len(msgID) == 0 {
		msgID = uuid.New().String()
	}
	if err := this.WritePacket(msgID, Open|Request|NeedAck, dial); err != nil {
		return nil, err
	}
	val, err := this.wait.Wait(msgID)
	if err != nil {
		return nil, err
	}
	res := new(DialRes)
	if err := json.Unmarshal([]byte(val.(string)), res); err != nil {
		return nil, err
	}
	*dial = *res.Dial
	return this.CreateIO(res.Key, closer), nil
}

// DialBridge 建立连接并进行数据桥接
// 相当于 Dial 后调用 Bridge 进行双向数据转发
func (this *Tunnel) DialBridge(msgID string, dial *Dial, userConn io.ReadWriteCloser) error {
	defer userConn.Close()
	i, err := this.Dial(msgID, dial, nil)
	if err != nil {
		return err
	}
	return Bridge(userConn, i)
}

// GetIO 根据 key 获取虚拟IO实例
func (this *Tunnel) GetIO(key string) *IO {
	this.ioMu.RLock()
	defer this.ioMu.RUnlock()
	return this.ioMap[key]
}

// CreateIO 创建一个新的虚拟IO并注册到隧道中
// key 为IO的唯一标识,closer 为关闭时触发的回调
func (this *Tunnel) CreateIO(key string, closer io.Closer) *IO {
	v := NewIO(this.r, func(v *IO) {
		v.OnWrite = func(bs []byte) ([]byte, error) {
			p := this.f.NewPacket(key, Write, bs)
			logs.Debug(p)
			return p.Bytes(), nil
		}
		v.OnClose = func(v *IO, err error) error {
			this.WritePacket(key, Close, err)
			this.ioMu.Lock()
			delete(this.ioMap, key)
			this.ioMu.Unlock()
			if closer != nil {
				closer.Close()
			}
			return nil
		}
	})
	this.ioMu.Lock()
	this.ioMap[key] = v
	this.ioMu.Unlock()
	return v
}

// Run 启动隧道主循环,开始处理数据包
// 该方法会阻塞直到连接关闭或发生错误
// 只能被调用一次,重复调用会等待第一次调用结束
func (this *Tunnel) Run() (err error) {

	if !this.running.CompareAndSwap(false, true) {
		<-this.Done()
		return this.Err()
	}

	defer func() {
		this.CloseWithErr(err)
		this.running.Store(false)
	}()
	buf := bufio.NewReader(this.r)

	for {
		p, err := this.f.ReadPacket(buf)
		if err != nil {
			return err
		}

		// 处理响应数据
		if !p.IsRequest() {
			if p.Success() {
				this.wait.Done(p.GetMsgID(), string(p.GetData()))
			} else {
				this.wait.Done(p.GetMsgID(), errors.New(string(p.GetData())))
			}
			continue
		}

		// 处理隧道过来的请求数据
		data, err := this.dealMessage(p)

		// 判断是否需要响应
		if p.NeedAck() {
			if err != nil {
				err = this.WritePacket(p.GetMsgID(), p.GetType()|Response|Fail, err)
				logs.PrintErr(err)
			} else {
				err = this.WritePacket(p.GetMsgID(), p.GetType()|Response|Success, data)
				logs.PrintErr(err)
			}
		}
	}
}

// dealMessage 处理请求类型的消息
func (this *Tunnel) dealMessage(p Packet) (any, error) {
	// 对于没注册的非注册消息,返回错误
	if !this.registered.Load() && p.GetType() != Register {
		return nil, ErrNotRegister
	}

	switch p.GetType() {

	case Register:
		if this.onRegister != nil {
			res, err := this.onRegister(this, p)
			if err == nil {
				this.registered.Store(true)
			}
			return res, err
		}

	case Read:
		i := this.GetIO(p.GetMsgID())
		if i != nil {
			bs := make([]byte, conv.Uint32(p.GetData()))
			n, err := i.Read(bs)
			if err != nil {
				return nil, err
			}
			return bs[:n], nil
		}
		return nil, ErrRemoteClose

	case Write:
		i := this.GetIO(p.GetMsgID())
		if i != nil {
			err := i.ToRead(p.GetData())
			return conv.Bytes(uint32(len(p.GetData()))), err
		}
		return nil, ErrRemoteClose

	case Close:
		i := this.GetIO(p.GetMsgID())
		if i != nil {
			errMsg := string(p.GetData())
			if len(errMsg) == 0 {
				errMsg = io.EOF.Error()
			}
			i.CloseWithErr(errors.New(errMsg))
		}

	case Open:
		return this.dealOpen(p)

	}

	return nil, nil
}

// dealOpen 处理 Open 类型的请求,建立到目标地址的连接
func (this *Tunnel) dealOpen(p Packet) (any, error) {
	if this.dial == nil {
		return nil, ErrDialInvalid
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
	c, key, err := this.dial(d)
	if err != nil {
		return nil, err
	}
	if this.onDialed != nil {
		this.onDialed(d, key)
	}
	i := this.CreateIO(key, c)
	go Bridge(i, c)
	return DialRes{Key: key, Dial: d}, nil
}
