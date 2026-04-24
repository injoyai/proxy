package core

import (
	"io"

	"github.com/injoyai/base/chans"
	"github.com/injoyai/base/safe"
)

var _ io.ReadWriteCloser = (*IO)(nil)

type IOOption func(v *IO)

func NewIO(w io.Writer, op ...IOOption) *IO {
	i := &IO{
		writer: w,
		reader: chans.NewIO(20),
		Closer: safe.NewCloser(),
	}
	for _, v := range op {
		v(i)
	}
	i.SetCloseFunc(func(error) error {
		if i.OnClose != nil {
			return i.OnClose(i, i.Err())
		}
		return i.reader.Close()
	})
	return i
}

// IO 虚拟IO,依托于Tunnel
type IO struct {
	writer       io.Writer                    //虚拟(公共)写入通道
	reader       *chans.IO                    //虚拟读取通道
	*safe.Closer                              //关闭通道
	OnWrite      func([]byte) ([]byte, error) //写入事件
	OnClose      func(v *IO, err error) error //关闭事件
}

// ToRead 写入数据到buffer,数据会流转到Read函数
func (this *IO) ToRead(p []byte) error {
	if this.Closed() {
		return this.Err()
	}
	_, err := this.reader.Write(p)
	return err
}

func (this *IO) Read(p []byte) (n int, err error) {
	if this.Closed() {
		return 0, this.Err()
	}
	return this.reader.Read(p)
}

func (this *IO) Write(p []byte) (n int, err error) {
	if this.Closed() {
		return 0, this.Err()
	}
	if this.reader.Closed() {
		return 0, io.EOF
	}
	//取原始的长度,外部调用者不用关系内部细节
	n = len(p)
	if this.OnWrite != nil {
		p, err = this.OnWrite(p)
		if err != nil {
			return 0, err
		}
		if p == nil {
			return
		}
	}
	_, err = this.writer.Write(p)
	return
}
