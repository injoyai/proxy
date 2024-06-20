package core

import (
	"fmt"
	"github.com/injoyai/logs"
	"io"
	"net"
)

func Swap(c1, c2 io.ReadWriter) error {
	go io.Copy(c1, c2)
	_, err := io.Copy(c2, c1)
	return err
}

func Listen(network string, port int, handler func(net.Listener, net.Conn) error) error {
	listener, err := net.Listen(network, fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	logs.Infof("[:%d] 监听成功...\n", port)
	for {
		c, err := listener.Accept()
		if err != nil {
			return err
		}
		go func(l net.Listener, c net.Conn) {
			defer c.Close()
			err = handler(l, c)
			if err != nil {
				logs.Error(err)
			}
		}(listener, c)
	}
}

func CopyFunc(w io.Writer, r io.Reader, f func(bs []byte) ([]byte, error)) (rErr error, wErr error) {
	size := 32 * 1024
	buf := make([]byte, size)
	for {
		n, err := r.Read(buf)
		if err != nil {
			return err, nil
		}
		bs, err := f(buf[:n])
		if err != nil {
			return err, nil
		}
		if len(bs) == 0 {
			continue
		}
		_, err = w.Write(bs)
		if err != nil {
			return nil, err
		}
	}
}
