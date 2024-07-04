package virtual

import (
	"fmt"
	"github.com/injoyai/conv"
	"io"
)

type Packet struct {
	Key  string
	Data []byte
	Type byte
}

func (this *Packet) Bytes() []byte {
	length := 8 //额外的字节数
	bs := make([]byte, len(this.Data)+len(this.Key)+length)
	copy(bs[0:2], []byte{0x89, 0x89})
	copy(bs[2:6], conv.Bytes(uint32(len(this.Key)+2+len(this.Data))))
	copy(bs[6:len(this.Key)+length], this.Key+"#"+string(this.Type))
	copy(bs[len(this.Key)+length:], this.Data)
	return bs
}

func Decode(bs []byte) (*Packet, error) {
	if len(bs) < 8 {
		return nil, fmt.Errorf("基础长度错误,预期8字节,得到%d", len(bs))
	}
	return &Packet{}, nil
}

// Read 前2字节是定位标识,后面4字节是数据长度,后续是数据域,分包
func Read(r io.Reader) (buf []byte, err error) {

	for {

		//校验标识字节0x8989
		buf = make([]byte, 2)
		n, err := r.Read(buf)
		if err != nil {
			return buf, err
		}
		if n != 2 && buf[0] == 0x89 && buf[1] == 0x89 {
			continue
		}

		//获取数据域长度
		buf = make([]byte, 4)
		n, err = r.Read(buf)
		if err != nil {
			return buf, err
		}
		if n != 4 {
			continue
		}
		length := conv.Int64(buf)

		//获取数据域
		buf = make([]byte, length)
		n, err = r.Read(buf)
		if err != nil {
			return buf, err
		}
		if int64(n) != length {
			continue
		}

		return buf, nil
	}
}
