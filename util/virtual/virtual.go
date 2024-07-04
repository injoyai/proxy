package virtual

import (
	"errors"
	"fmt"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/conv"
	"io"
	"sync/atomic"
)

func New(r io.ReadWriteCloser) *Virtual {
	//按照固定协议去读取数据

}

type Virtual struct {
	*maps.Safe
}

func (this *Virtual) Publish(key string, p []byte) error {
	v, _ := this.Safe.GetOrSetByHandler(key, func() (interface{}, error) {
		return &IO{}, nil
	})
	_, err := v.(*IO).Write(p)
	return err
}

var _ io.ReadWriteCloser = (*IO)(nil)

type IO struct {
	Key     string                       //标识
	writer  io.Writer                    //公共写入通道
	reader  *Buffer                      //虚拟读取通道
	OnWrite func([]byte) ([]byte, error) //写入事件
	OnClose func(v *IO, err error) error //关闭事件
}

func (this *IO) ToBuffer(p []byte) error {
	_, err := this.reader.Write(p)
	return err
}

func (this *IO) Read(p []byte) (n int, err error) {
	return this.reader.Read(p)
}

func (this *IO) Write(p []byte) (n int, err error) {
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

func (this *IO) CloseWithErr(err error) error {
	if this.OnClose != nil {
		return this.OnClose(this, err)
	}
	return this.reader.Close()
}

func (this *IO) Close() error {
	return this.CloseWithErr(errors.New("主动关闭"))
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
}

func (this *Buffer) Write(p []byte) (n int, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	if atomic.LoadUint32(&this.closed) == 1 {
		return 0, errors.New("closed")
	}
	//当阻塞的时候,进行关闭操作,会panic
	this.C <- p
	return len(p), nil
}

func (this *Buffer) Read(p []byte) (n int, err error) {
	if atomic.LoadUint32(&this.closed) == 1 {
		return 0, errors.New("closed")
	}

	if len(this.cache) == 0 {
		bs, ok := <-this.C
		if !ok {
			atomic.StoreUint32(&this.closed, 1)
			return 0, io.EOF
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

func (this *Buffer) Close() error {
	if atomic.CompareAndSwapUint32(&this.closed, 0, 1) {
		close(this.C)
	}
	return nil
}

func (this *Buffer) Closed() bool {
	return atomic.LoadUint32(&this.closed) == 1
}
