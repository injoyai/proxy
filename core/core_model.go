package core

import (
	"fmt"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/logs"
	"io"
	"net"
	"strings"
	"time"
)

type Dial struct {
	Type    string        `json:"type"`    //连接类型,TCP,UDP,Websocket,Serial...
	Address string        `json:"address"` //连接地址
	Timeout time.Duration `json:"timeout"` //超时时间
	Param   g.Map         `json:"param"`   //其他参数
}

func (this *Dial) Dial() (io.ReadWriteCloser, string, error) {
	//if this.Timeout==0{
	//	this.Timeout=10*time.Second
	//}
	logs.Debug(this.Address)
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

type Listen struct {
	Type    string `json:"type"`
	Port    string `json:"port"`
	Param   g.Map  `json:"param"`
	Handler func()
}

func (this *Listen) Listen(onListen func(net.Listener), onConnect func(net.Listener, net.Conn) error) error {
	var listener net.Listener
	var err error
	switch strings.ToLower(this.Type) {
	case "tcp":
		listener, err = net.Listen(this.Type, fmt.Sprintf(":%s", this.Port))
	default:
		listener, err = net.Listen("tcp", fmt.Sprintf(":%s", this.Port))
	}
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
