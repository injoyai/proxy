package core

import (
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/g"
	"io"
	"net"
	"strings"
	"time"
)

func NewDialTCP(address string, timeout ...time.Duration) *Dial {
	return &Dial{
		Type:    "tcp",
		Address: address,
		Timeout: conv.DefaultDuration(0, timeout...),
	}
}

type Dial struct {
	Type    string        `json:"type,omitempty"`    //连接类型,TCP,UDP,Websocket,Serial...
	Address string        `json:"address"`           //连接地址
	Timeout time.Duration `json:"timeout,omitempty"` //超时时间
	Param   g.Map         `json:"param,omitempty"`   //其他参数
}

func (this *Dial) Dial() (io.ReadWriteCloser, string, error) {
	switch strings.ToLower(this.Type) {
	case "tcp":
		c, err := net.DialTimeout(this.Type, this.Address, this.Timeout)
		if err != nil {
			return nil, "", err
		}
		return c, c.LocalAddr().String(), nil

	//case "udp":
	//case "websocker","ws":
	//case "serial":

	default:
		c, err := net.DialTimeout("tcp", this.Address, this.Timeout)
		if err != nil {
			return nil, "", err
		}
		return c, c.LocalAddr().String(), nil
	}
}

func NewListenTCP(port string) *Listen {
	return &Listen{Port: port}
}

type Listen struct {
	Type  string `json:"type,omitempty"`  //类型,TCP,UDP,Serial等
	Port  string `json:"port"`            //例如串口是字符的,固使用字符类型
	Param g.Map  `json:"param,omitempty"` //其他参数
}

func (this *Listen) Listener() (net.Listener, error) {
	var listener net.Listener
	var err error
	switch strings.ToLower(this.Type) {
	case "tcp":
		listener, err = net.Listen(this.Type, fmt.Sprintf(":%s", this.Port))
	default:
		listener, err = net.Listen("tcp", fmt.Sprintf(":%s", this.Port))
	}
	if err != nil {
		return nil, err
	}
	return listener, nil
}

func (this *Listen) Listen(onListen func(net.Listener), onConnect func(net.Listener, net.Conn) error) error {
	listener, err := this.Listener()
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

func (this *Listen) GoListen(onConnect func(net.Listener, net.Conn) error) (net.Listener, error) {
	listener, err := this.Listener()
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
