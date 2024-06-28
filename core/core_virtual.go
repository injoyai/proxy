package core

import (
	"errors"
	"io"
)

var _ io.ReadWriteCloser = (*Virtual)(nil)

// Virtual 虚拟IO
// 从一个IO中虚拟出多个虚拟IO
type Virtual struct {
	Key     string                            //标识
	writer  io.Writer                         //公共写入通道
	reader  *ChanIO                           //虚拟读取通道
	OnWrite func([]byte) ([]byte, error)      //写入事件
	OnClose func(v *Virtual, err error) error //关闭事件
}

func (this *Virtual) ToBuffer(p []byte) error {
	_, err := this.reader.Write(p)
	return err
}

func (this *Virtual) Read(p []byte) (n int, err error) {
	return this.reader.Read(p)
}

func (this *Virtual) Write(p []byte) (n int, err error) {
	if this.reader.Closed() {
		return 0, errors.New("closed")
	}
	if this.OnWrite != nil {
		p, err = this.OnWrite(p)
		if err != nil {
			return 0, err
		}
	}
	return this.writer.Write(p)
}

func (this *Virtual) CloseWithErr(err error) error {
	if this.OnClose != nil {
		return this.OnClose(this, err)
	}
	return this.reader.Close()
}

func (this *Virtual) Close() error {
	return this.CloseWithErr(errors.New("主动关闭"))
}

func NewVirtual(key string, tun io.Writer, buf *ChanIO, onWrite func([]byte) ([]byte, error), onClose func(v *Virtual, err error) error) *Virtual {
	return &Virtual{
		Key:     key,
		writer:  tun,
		reader:  buf,
		OnWrite: onWrite,
		OnClose: onClose,
	}
}
