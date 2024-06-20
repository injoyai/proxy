package tunnel

import (
	"bytes"
	"errors"
)

const (
	Write    = 'w' //写入数据
	Open     = 'o' //建立连接,缩写和Close相同,固改成Open
	Close    = 'c' //关闭连接
	Info     = 'i' //信息
	Register = 'r' //注册
	Split    = '#' //分隔符
	Request  = 0   //请求
	Response = 1   //响应
)

type Packet struct {
	//10000001
	//第一位是方向,0是请求,1是响应
	//第二位是结果,0是成功,1是失败
	//第三位是是否需要响应,0是不需要,1是需要
	//第4~8位预留
	Code    byte   `json:"code,omitempty"`
	Type    byte   `json:"type"`
	Address string `json:""`
	Body    []byte
}

// IsRequest 是否是请求数据
func (this *Packet) IsRequest() bool {
	return this.Code&0x80 == 0
}

// IsResponse 是否是响应数据
func (this *Packet) IsResponse() bool {
	return this.Code&0x80 == 0x80
}

// Success 是否成功,当为响应时生效
func (this *Packet) Success() bool {
	return this.Code&0x40 == 0
}

// NeedResponse 是否需要响应,当为请求时生效
func (this *Packet) NeedResponse() bool {
	return this.Code&0x20 == 0x20
}

func (this *Packet) Bytes() []byte {
	bs := make([]byte, len(this.Body)+len(this.Address)+3)
	bs[0] = this.Code
	bs[1] = this.Type
	copy(bs[2:], []byte(this.Address))
	bs[len(this.Address)+2] = Split
	copy(bs[len(this.Address)+3:], this.Body)
	return bs
}

func Decode(bs []byte) (Packet, error) {
	list := bytes.SplitN(bs, []byte{Split}, 2)
	if len(list) == 1 || len(list[0]) < 2 {
		return Packet{}, errors.New("数据异常: " + string(bs))
	}
	return Packet{
		Code:    list[0][0],
		Type:    list[0][1],
		Address: string(list[0][2:]),
		Body:    list[1],
	}, nil
}
