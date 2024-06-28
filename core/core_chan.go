package core

import (
	"errors"
	"fmt"
	"github.com/injoyai/conv"
	"io"
	"sync/atomic"
)

type ChanByte chan byte

func (this ChanByte) Write(p []byte) (n int, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	for i := range p {
		this <- p[i]
	}
	return len(p), nil
}

func (this ChanByte) Read(p []byte) (n int, err error) {
	var ok bool
	for i := range p {
		if i == 0 {
			p[i], ok = <-this
			if !ok {
				//这个类型的目的就是为了控制EOF,
				//返回错误的话就不能达到目标效果
				//固这里返回EOF,下同
				return 0, io.EOF
			}
			n++
		}
		if i > 0 {
			select {
			case b, ok := <-this:
				if !ok {
					return 0, io.EOF
				}
				p[i] = b
				n++
			default:
				break
			}
		}
	}
	return
}

func (this ChanByte) Close() error {
	close(this)
	return nil
}

//========================================

func NewChanIO(cap ...uint) *ChanIO {
	return &ChanIO{
		C: make(chan []byte, conv.DefaultUint(0, cap...)),
	}
}

// ChanIO 通道类型IO,关闭后会返回
type ChanIO struct {
	C      chan []byte
	cache  []byte
	closed uint32
}

func (this *ChanIO) Write(p []byte) (n int, err error) {
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

func (this *ChanIO) Read(p []byte) (n int, err error) {
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

func (this *ChanIO) Close() error {
	if atomic.CompareAndSwapUint32(&this.closed, 0, 1) {
		close(this.C)
	}
	return nil
}

func (this *ChanIO) Closed() bool {
	return atomic.LoadUint32(&this.closed) == 1
}

func CopyChan[T ~uint8 | ~[]byte | ~string](w, r chan T) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	for {
		p, ok := <-r
		if !ok {
			return io.EOF
		}
		w <- p
	}
}
