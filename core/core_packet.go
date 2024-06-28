package core

import (
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
	"io"
)

type Packet struct {
	Data []byte
}

func (this *Packet) Bytes() []byte {
	bs := make([]byte, len(this.Data)+6)
	copy(bs[0:2], []byte{0x89, 0x89})
	copy(bs[2:6], conv.Bytes(uint32(len(this.Data))))
	copy(bs[6:], this.Data)
	return bs
}

func Write(c io.Writer, any interface{ Bytes() []byte }) (int, error) {
	logs.Write(any)
	return c.Write((&Packet{
		Data: any.Bytes(),
	}).Bytes())
}

// Read 前2字节是定位标识,后面4字节是数据长度,后续是数据域
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
