package core

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/injoyai/logs"
)

// NewListenTCP 创建一个 TCP 类型的监听器配置
// port 为监听端口号
func NewListenTCP(port int) *Listen {
	return &Listen{Port: strconv.Itoa(port)}
}

type Listen struct {
	Type  string         `json:"type,omitempty"`  // Type 监听类型,支持 tcp/udp/serial 等
	Port  string         `json:"port"`            // Port 监听端口,串口等类型可能使用字符串
	Param map[string]any `json:"param,omitempty"` // Param 其他自定义参数
}

func (this *Listen) Listener(ctx context.Context) (net.Listener, error) {
	var listener net.Listener
	var err error
	switch this.Type {
	case "tcp":
		listener, err = net.Listen("tcp", fmt.Sprintf(":%s", this.Port))
	default:
		listener, err = net.Listen("tcp", fmt.Sprintf(":%s", this.Port))
	}
	if err == nil {
		go func() {
			<-ctx.Done()
			listener.Close()
		}()
	}
	return listener, err
}

func (this *Listen) Listen(ctx context.Context, onListen func(net.Listener), onConnect func(net.Listener, net.Conn) error) error {
	listener, err := this.Listener(ctx)
	if err != nil {
		return err
	}
	if onListen != nil {
		onListen(listener)
	}
	for {
		c, err := listener.Accept()
		if err != nil {
			return err
		}
		go func(l net.Listener, c net.Conn) {
			defer c.Close()
			onConnect(l, c)
		}(listener, c)
	}
}

func (this *Listen) GoListen(ctx context.Context, onConnect func(net.Listener, net.Conn) error) (net.Listener, error) {
	listener, err := this.Listener(ctx)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			go func(l net.Listener, c net.Conn) {
				defer c.Close()
				onConnect(l, c)
			}(listener, c)
		}
	}()
	return listener, nil
}

func WithListenLog(l net.Listener) {
	logs.Infof("[%s] 监听成功...\n", l.Addr().String())
}
