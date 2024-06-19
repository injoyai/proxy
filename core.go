package main

import (
	"fmt"
	"net"
)

func ListenTCP(port int) error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}

		}

	}()

	return nil
}

func Handler(c net.Conn) {

}
