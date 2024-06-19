package core

import (
	"fmt"
	"github.com/injoyai/logs"
	"net"
)

type Forward struct {
	Port    int
	Address string
}

func (this *Forward) ListenTCP() error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", this.Port))
	if err != nil {
		return err
	}

	logs.Infof("[:%d] 监听成功...\n", this.Port)

	for {
		c, err := listener.Accept()
		if err != nil {
			return err
		}
		logs.Infof("[%s] 转发至 [%s]\n", c.RemoteAddr().String(), this.Address)
		go this.Handler(c)
	}

}

func (this *Forward) Handler(c net.Conn) error {
	defer c.Close()

	newConn, err := net.Dial("tcp", this.Address)
	if err != nil {
		return err
	}
	defer newConn.Close()

	return Copy(c, newConn)
}
