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

	"github.com/injoyai/base/maps/wait"
	"github.com/injoyai/base/safe"
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
	uuid "github.com/satori/go.uuid"
)

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

type Tunnel struct {
	*safe.Closer

	k          string             //唯一标识
	f          Frame              //帧协议
	r          io.ReadWriteCloser //实际连接
	ioMu       sync.RWMutex       //锁
	ioMap      map[string]*IO     //虚拟io
	wait       *wait.Entity       //异步等待机制
	running    atomic.Bool        //是否在运行
	registered atomic.Bool        //是否已经注册

	dial       func(d *Dial) (io.ReadWriteCloser, string, error)
	onRegister func(v *Tunnel, p Packet) (interface{}, error)
	onDialed   func(d *Dial, key string)
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

func (this *Tunnel) WritePacket(msgID string, t byte, i any) error {
	p := this.f.NewPacket(msgID, t, i)
	_, err := this.r.Write(p.Bytes())
	return err
}

func (this *Tunnel) Register(data any) (any, error) {
	if err := this.WritePacket(this.Key(), Request|Register|NeedAck, data); err != nil {
		return nil, err
	}
	return this.wait.Wait(this.Key())
}

func (this *Tunnel) Dial(msgID string, dial *Dial, closer io.Closer) (io.ReadWriteCloser, error) {
	if len(msgID) == 0 {
		msgID = uuid.NewV4().String()
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

func (this *Tunnel) DialBridge(msgID string, dial *Dial, userConn io.ReadWriteCloser) error {
	defer userConn.Close()
	i, err := this.Dial(msgID, dial, nil)
	if err != nil {
		return err
	}
	return Bridge(userConn, i)
}

func (this *Tunnel) GetIO(key string) *IO {
	this.ioMu.RLock()
	defer this.ioMu.RUnlock()
	return this.ioMap[key]
}

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

		//处理响应数据
		if !p.IsRequest() {
			if p.Success() {
				this.wait.Done(p.GetMsgID(), string(p.GetData()))
			} else {
				this.wait.Done(p.GetMsgID(), errors.New(string(p.GetData())))
			}
			continue
		}

		//处理隧道过来的请求数据
		data, err := this.dealMessage(p)

		//判断是否需要响应
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

/*



 */

func (this *Tunnel) dealMessage(p Packet) (any, error) {
	//对于没注册的非注册消息,返回错误
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
