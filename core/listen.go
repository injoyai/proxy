package core

import (
	"context"
	"fmt"
	"net"

	"github.com/injoyai/logs"
)

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

func RunListen(network string, port int, onListen func(net.Listener), onConnect func(net.Listener, net.Conn) error) error {
	listener, err := net.Listen(network, fmt.Sprintf(":%d", port))
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

func GoListen(network string, port int, onConnect func(net.Listener, net.Conn) error) (net.Listener, error) {
	listener, err := net.Listen(network, fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			go onConnect(listener, c)
		}
	}()
	return listener, nil
}
