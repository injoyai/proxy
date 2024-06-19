package core

import (
	"fmt"
	"io"
	"net"
)

type P2P struct {
	Port    int
	Address string
}

func (this *P2P) ListenTCP(port int) error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	for {
		c, err := listener.Accept()
		if err != nil {
			return err
		}
		go this.Handler(c)
	}

}

func (this *P2P) Handler(c net.Conn) error {
	defer c.Close()

	newConn, err := net.Dial("tcp", this.Address)
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
