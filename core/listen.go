package core

import (
	"cmp"
	"context"
	"net"

	"github.com/injoyai/logs"
)

type ListenOption func(*Listen)

func WithListened(f func(net.Listener)) ListenOption {
	return func(l *Listen) {
		l.OnListened(f)
	}
}

func WithListenLog() ListenOption {
	return func(listen *Listen) {
		listen.OnListened(func(listener net.Listener) {
			logs.Infof("[%s] 开始监听...\n", listener.Addr().String())
		})
		listen.OnListenErr(func(listener net.Listener, err error) {
			logs.Errorf("[%s] 结束监听: %v\n", listener.Addr().String(), err)
		})
	}
}

func WithListenErr(f func(net.Listener, error)) ListenOption {
	return func(l *Listen) {
		l.OnListenErr(f)
	}
}

func WithConnected(f func(net.Listener, net.Conn)) ListenOption {
	return func(l *Listen) {
		l.OnConnected(f)
	}
}

// NewListenTCP 创建一个 TCP 类型的监听器配置
// addr 为监听地址/端口
func NewListenTCP[T cmp.Ordered](addr T, op ...ListenOption) *Listen {
	return NewListen(TCP, addr, op...)
}

// NewListen 创建一个任意类型的监听器配置
// addr 为监听地址/端口
func NewListen[T cmp.Ordered](_type string, addr T, op ...ListenOption) *Listen {
	l := &Listen{
		Type:    _type,
		Address: Address(addr),
	}
	WithListenLog()(l)
	l.SetOption(op...)
	return l
}

type Listen struct {
	Type        string         `json:"type,omitempty"`  // Type 监听类型,支持 tcp/udp/serial 等
	Address     string         `json:"address"`         // Address 监听地址
	Param       map[string]any `json:"param,omitempty"` // Param 其他自定义参数
	onListened  func(net.Listener)
	onListenErr func(net.Listener, error)
	onConnected func(net.Listener, net.Conn)
	listener    net.Listener
}

func (this *Listen) Key() string {
	if this.listener == nil {
		return ""
	}
	return this.listener.Addr().String()
}

func (this *Listen) Close() error {
	if this.listener == nil {
		return nil
	}
	return this.listener.Close()
}

func (this *Listen) SetOption(op ...ListenOption) {
	for _, v := range op {
		v(this)
	}
}

func (this *Listen) OnListened(f func(net.Listener)) {
	this.onListened = f
}

func (this *Listen) OnListenErr(f func(net.Listener, error)) {
	this.onListenErr = f
}

func (this *Listen) OnConnected(f func(net.Listener, net.Conn)) {
	this.onConnected = f
}

func (this *Listen) Listen() error {
	var err error
	switch this.Type {
	case TCP:
		this.listener, err = net.Listen(TCP, this.Address)
	default:
		this.listener, err = net.Listen(TCP, this.Address)
	}
	return err
}

func (this *Listen) Run(ctx ...context.Context) (err error) {
	defer func() {
		if this.onListenErr != nil {
			this.onListenErr(this.listener, err)
		}
	}()
	if this.onListened != nil {
		this.onListened(this.listener)
	}
	if len(ctx) > 0 && ctx[0] != nil {
		go func() {
			<-ctx[0].Done()
			this.Close()
		}()
	}
	for {
		c, err := this.listener.Accept()
		if err != nil {
			return err
		}
		go func(l net.Listener, c net.Conn) {
			defer c.Close()
			if this.onConnected != nil {
				this.onConnected(l, c)
			}
		}(this.listener, c)
	}
}

func (this *Listen) ListenAndRun(ctx ...context.Context) error {
	err := this.Listen()
	if err != nil {
		return err
	}
	return this.Run(ctx...)
}
