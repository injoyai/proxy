package core

import (
	"fmt"
	"github.com/injoyai/logs"
	"io"
	"net"
)

type Forward struct {
	ListenPort   int
	ProxyAddress string
}

func (this *Forward) ListenTCP() error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", this.ListenPort))
	if err != nil {
		return err
	}

	logs.Infof("[:%d] 监听成功...\n", this.ListenPort)

	for {
		c, err := listener.Accept()
		if err != nil {
			return err
		}
		logs.Infof("[%s] 代理至 [%s]\n", c.RemoteAddr().String(), this.ProxyAddress)
		go this.Handler(c)
	}

}

func (this *Forward) Handler(c net.Conn) error {
	defer c.Close()

	newConn, err := net.Dial("tcp", this.ProxyAddress)
	if err != nil {
		return err
	}
	defer newConn.Close()

	return Copy(c, newConn)
}

func Copy(c1, c2 io.ReadWriter) error {
	go io.Copy(c1, c2)
	_, err := io.Copy(c2, c1)
	return err
}
