package pool

import (
	"cmp"
	"context"
	"io"
	"net"
	"time"

	"github.com/injoyai/conv"
	"github.com/injoyai/proxy/core"
)

type Dialer interface {
	Dial(dial *core.Dial, onCloser func() error) (io.ReadWriteCloser, error)
	Done() <-chan struct{}
}

type Pool interface {
	Cap() int                     //池子容量
	Len() int                     //池子大小
	Get() Dialer                  //拿取一个连接
	Put(Dialer, ...time.Duration) //放回连接池,可以延时放回
	Run(...context.Context) error //运行服务
}

func New[T cmp.Ordered](addr T, size ...int) Pool {
	ch := make(chan Dialer, conv.Default(100, size...))
	return &pool{
		ch: ch,
		Listen: core.NewListenTCP(addr, core.WithConnected(func(listener net.Listener, conn net.Conn) {
			key := conn.RemoteAddr().String()
			tun := core.NewTunnel(conn, core.WithKey(key))
			go tun.Run()
			ch <- tun
		})),
	}
}

type pool struct {
	ch chan Dialer
	*core.Listen
}

func (this *pool) Cap() int {
	return cap(this.ch)
}

func (this *pool) Len() int {
	return len(this.ch)
}

func (this *pool) Get() Dialer {
	for dial := range this.ch {
		select {
		case <-dial.Done():
			continue
		default:
			return dial
		}
	}
	return nil
}

func (this *pool) Put(dial Dialer, after ...time.Duration) {
	t := conv.Default(0, after...)
	time.AfterFunc(t, func() {
		select {
		case <-dial.Done():
			return
		case this.ch <- dial:
		}
	})
}
