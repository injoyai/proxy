package core

import (
	"github.com/injoyai/base/chans"
	"github.com/injoyai/base/safe"
	"io"
)

var _ io.ReadWriteCloser = (*Virtual)(nil)

type OptionVirtual func(v *Virtual)

func NewVirtual(w io.Writer, r *chans.IO, op ...OptionVirtual) *Virtual {
	i := &Virtual{
		writer: w,
		reader: r,
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

// Virtual 虚拟IO,依托于Tunnel
type Virtual struct {
	writer       io.Writer                         //虚拟(公共)写入通道
	reader       *chans.IO                         //虚拟读取通道
	*safe.Closer                                   //关闭通道
	OnWrite      func([]byte) ([]byte, error)      //写入事件
	OnClose      func(v *Virtual, err error) error //关闭事件
}

// ToRead 写入数据到buffer,数据会流转到Read函数
func (this *Virtual) ToRead(p []byte) error {
	if this.Closed() {
		return this.Err()
	}
	_, err := this.reader.Write(p)
	return err
}

func (this *Virtual) Read(p []byte) (n int, err error) {
	if this.Closed() {
		return 0, this.Err()
	}
	return this.reader.Read(p)
}

func (this *Virtual) Write(p []byte) (n int, err error) {
	if this.Closed() {
		return 0, this.Err()
	}
	if this.reader.Closed() {
		return 0, io.EOF
	}
	n = len(p) //取原始的长度
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
