package virtual

import (
	"bytes"
	"fmt"
	"github.com/injoyai/conv"
	"io"
	"strings"
)

const (
	Read   = 0x00 //读取,能有效抑制源源不断地数据堆积在内存,当然性能会有所下降,多了一次IO
	Write  = 0x01 //写入
	Open   = 0x02 //打开
	Close  = 0x03 //关闭
	Listen = 0x04

	Request  = 0x00 //请求
	Response = 0x80 //响应
	Success  = 0x00 //成功
	Fail     = 0x40 //失败
	NeedAck  = 0x20 //需要响应
)

var DefaultFrame = &frame{}

type Frame interface {
	NewPacket(k string, t byte, i interface{}) Packet
	ReadPacket(r io.Reader) (Packet, error)
}

type Packet interface {
	Bytes() []byte
	GetKey() string
	GetData() []byte
	GetType() byte
	IsRequest() bool
	Success() bool
	NeedAck() bool
}

/*
Split = '#'  //分隔符
Request  = 0x00
Response = 0x80
Success  = 0x00 //成功
Fail     = 0x40 //失败
NeedAck  = 0x20 //需要响应
*/
type packet struct {
	Key  string
	Data []byte
	Code byte
}

func (this *packet) String() string {
	return fmt.Sprintf("[%s] 类型: %s, 结果: %s, 数据: %s",
		this.Key,
		func() string {
			switch this.GetType() {
			case Write:
				return "写入数据"
			case Open:
				return "建立连接"
			case Close:
				return "关闭连接"
			default:
				return "未知"
			}
		}(),
		func() string {
			ls := []string(nil)
			ls = append(ls, conv.SelectString(this.IsRequest(), "请求", "响应"))
			if this.IsRequest() {
				ls = append(ls, conv.SelectString(this.NeedAck(), "需要确认", "无需确认"))
			} else {
				ls = append(ls, conv.SelectString(this.Success(), "成功", "失败"))
			}
			return strings.Join(ls, "|")
		}(),
		this.Data,
	)
}

func (this *packet) IsRequest() bool {
	return this.Code&0x80 == 0x00
}

func (this *packet) Success() bool {
	return this.Code&0x40 == 0x00
}

// NeedAck 是否需要确认,当为请求时生效
func (this *packet) NeedAck() bool {
	return this.Code&0x20 == 0x20
}

func (this *packet) GetType() byte {
	return this.Code & 0x03
}

func (this *packet) Bytes() []byte {
	lenData := len(this.Data) //额外的字节数
	lenKey := len(this.Key)
	bs := make([]byte, len(this.Data)+lenKey+8)
	copy(bs[0:2], []byte{0x89, 0x89})
	copy(bs[2:6], conv.Bytes(uint32(lenKey+2+lenData)))
	copy(bs[6:len(this.Key)+6], this.Key)
	bs[lenKey+6] = '#'
	bs[lenKey+7] = this.Code
	copy(bs[lenKey+8:], this.Data)
	return bs
}

func (this *packet) GetKey() string {
	return this.Key
}

func (this *packet) GetData() []byte {
	return this.Data
}

type frame struct{}

func (this *frame) NewPacket(k string, t byte, d interface{}) Packet {
	return &packet{
		Key:  k,
		Data: conv.Bytes(d),
		Code: t,
	}
}

func (this *frame) WritePacket(w io.Writer, p Packet) error {
	_, err := w.Write(p.Bytes())
	return err
}

func (this *frame) ReadPacket(r io.Reader) (Packet, error) {
	bs, err := this.Read(r)
	if err != nil {
		return nil, err
	}
	return this.Decode(bs)
}

func (this *frame) Decode(bs []byte) (Packet, error) {
	if len(bs) < 2 {
		return nil, fmt.Errorf("基础长度错误,预期至少2字节,得到%d", len(bs))
	}
	list := bytes.SplitN(bs, []byte{'#'}, 2)
	if len(list) != 2 {
		return nil, fmt.Errorf("数据分割异常: %v", bs)
	}
	if len(list[1]) == 0 {
		return nil, fmt.Errorf("数据类型异常: %v", bs)
	}
	return &packet{
		Key:  string(list[0]),
		Code: list[1][0],
		Data: list[1][1:],
	}, nil
}

// Read 前2字节是定位标识,后面4字节是数据长度,后续是数据域,分包
func (this *frame) Read(r io.Reader) (buf []byte, err error) {

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
