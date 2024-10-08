package core

import (
	"fmt"
	"io"
	"net"
)

func CopyBufferWith(w io.Writer, r io.Reader, buf []byte, f func(p []byte) ([]byte, error)) error {
	//if false {
	//	_, err := io.Copy(w, r)
	//	return err
	//}

	if len(buf) == 0 {
		//未声明或者cap为0的情况,重新声明
		//todo 这里设置太小会有bug,还不清楚原因,会卡在那里,后续排查
		buf = make([]byte, 1024*32)
	}
	if f == nil {
		f = func(p []byte) ([]byte, error) {
			return p, nil
		}
	}
	for {
		n, err := r.Read(buf)
		if err != nil {
			return err
		}
		bs, err := f(buf[:n])
		if err != nil {
			return err
		}
		if _, err = w.Write(bs); err != nil {
			return err
		}
	}
}

// Swap 交换数据
func Swap(c1, c2 io.ReadWriteCloser) error {
	defer c1.Close()
	defer c2.Close()
	go io.Copy(c1, c2)
	_, err := io.Copy(c2, c1)
	return err
}

func WithListenLog(l net.Listener) {
	DefaultLog.Infof("[%s] 监听成功...\n", l.Addr().String())
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
