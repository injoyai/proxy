package forward

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"net"
)

type Forward struct {
	Port    int
	Address string
}

func (this *Forward) ListenTCP() error {
	return core.RunListen("tcp", this.Port, nil, this.Handler)
}

func (this *Forward) Handler(l net.Listener, c net.Conn) error {
	logs.Infof("[%s] 转发至 [%s]\n", c.RemoteAddr().String(), this.Address)
	defer c.Close()

	newConn, err := net.Dial("tcp", this.Address)
	if err != nil {
		return err
	}
	defer newConn.Close()

	return core.Swap(c, newConn)
}
