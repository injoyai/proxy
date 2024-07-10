package virtual

import (
	"fmt"
	"github.com/injoyai/base/safe"
	"github.com/injoyai/conv"
	"io"
	"sync/atomic"
)

var _ io.ReadWriteCloser = (*IO)(nil)

type OptionIO func(v *IO)

func NewIO(key string, w io.Writer, r *Buffer, onWrite func([]byte) ([]byte, error), onClose func(v *IO, err error) error) *IO {
	i := &IO{
		Key:     key,
		writer:  w,
		reader:  r,
		Closer:  safe.NewCloser(),
		OnWrite: onWrite,
		OnClose: onClose,
	}
	i.SetCloseFunc(func() error {
		if i.OnClose != nil {
			return i.OnClose(i, i.Err())
		}
		return i.reader.Close()
	})
	return i
}

type IO struct {
	Key          string                       //标识
	writer       io.Writer                    //公共写入通道
	reader       *Buffer                      //虚拟读取通道
	*safe.Closer                              //关闭通道
	OnWrite      func([]byte) ([]byte, error) //写入事件
	OnClose      func(v *IO, err error) error //关闭事件
}

// ToBuffer 写入数据到buffer,数据会流转到Read函数
func (this *IO) ToBuffer(p []byte) error {
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
		return 0, this.reader.err
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

//================

func NewBuffer(cap ...uint) *Buffer {
	return &Buffer{
		C: make(chan []byte, conv.DefaultUint(0, cap...)),
	}
}

// Buffer 通道类型IO,关闭后会返回
type Buffer struct {
	C      chan []byte
	cache  []byte
	closed uint32
	err    error
}

func (this *Buffer) Write(p []byte) (n int, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	if atomic.LoadUint32(&this.closed) == 1 {
		return 0, this.err
	}
	//当阻塞的时候,进行关闭操作,会panic
	this.C <- p
	return len(p), nil
}

func (this *Buffer) Read(p []byte) (n int, err error) {
	if atomic.LoadUint32(&this.closed) == 1 {
		return 0, this.err
	}

	if len(this.cache) == 0 {
		bs, ok := <-this.C
		if !ok {
			if atomic.CompareAndSwapUint32(&this.closed, 0, 1) {
				if this.err == nil {
					//手动关闭的,才会没有错误信息,则返回EOF
					this.err = io.EOF
				}
			}
			return 0, this.err
		}
		this.cache = bs
	}

	//从缓存(上次剩余的字节)复制数据到p
	n = copy(p, this.cache)
	if n < len(this.cache) {
		this.cache = this.cache[n:]
		return
	}

	this.cache = nil
	return
}

func (this *Buffer) CloseWithErr(err error) error {
	if err == nil {
		return nil
	}
	if atomic.CompareAndSwapUint32(&this.closed, 0, 1) {
		this.err = err
		close(this.C)
	}
	return nil
}

func (this *Buffer) Close() error {
	return this.CloseWithErr(io.EOF)
}

func (this *Buffer) Closed() bool {
	return atomic.LoadUint32(&this.closed) == 1
}
