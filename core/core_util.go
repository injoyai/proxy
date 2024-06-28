package core

import (
	"fmt"
	"github.com/injoyai/logs"
	"io"
	"net"
)

// Swap 交换数据
func Swap(c1, c2 io.ReadWriter) error {
	go io.Copy(c1, c2)
	_, err := io.Copy(c2, c1)
	return err
}

// SwapClose 交换数据并关闭
func SwapClose(c1, c2 io.ReadWriteCloser) error {
	defer c1.Close()
	defer c2.Close()
	return Swap(c1, c2)
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

type CoverWriter struct {
	io.Writer
	Handler func(p []byte) ([]byte, error)
}

func (this *CoverWriter) Write(bs []byte) (n int, err error) {
	if this.Handler != nil {
		bs, err = this.Handler(bs)
		if err != nil {
			return 0, err
		}
	}
	return this.Writer.Write(bs)
}

type CoverReader struct {
	io.Reader
	Handler func(p []byte) ([]byte, error)
}

func (this *CoverReader) Read(bs []byte) (n int, err error) {
	n, err = this.Reader.Read(bs)
	if err != nil {
		return 0, err
	}
	if this.Handler != nil {
		bs, err = this.Handler(bs)
		if err != nil {
			return 0, err
		}
	}
	return
}
