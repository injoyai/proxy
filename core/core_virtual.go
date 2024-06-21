package core

import (
	"bytes"
	"errors"
	"io"
)

// Virtual 虚拟设备
type Virtual struct {
	Key     string                       //标识
	Tun     io.ReadWriter                //tun连接
	Buffer  *bytes.Buffer                //缓存
	closed  bool                         //判断是否被关闭
	OnWrite func([]byte) ([]byte, error) //写入事件
	OnClose func(v *Virtual) error       //关闭事件
}

func (this *Virtual) Read(p []byte) (n int, err error) {
	if this.Buffer.Len() == 0 && this.closed {
		return 0, io.EOF
	}
	return this.Buffer.Read(p)
}

func (this *Virtual) Write(p []byte) (n int, err error) {
	if this.closed {
		return 0, errors.New("closed")
	}
	return this.Tun.Write(p)
}

func (this *Virtual) Close() error {
	if this.closed {
		return nil
	}
	this.closed = true
	return nil
}
