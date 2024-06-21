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

	Request  = 0x00 //请求
	Response = 0x80 //响应
	Success  = 0x00 //成功
	Fail     = 0x40 //失败
	NeedAck  = 0x20 //需要响应
)

type Packet struct {
	//10000001
	//第一位是方向,0是请求,1是响应
	//第二位是结果,0是成功,1是失败
	//第三位是是否需要响应,0是不需要,1是需要
	//第4~8位预留
	Code byte   `json:"code,omitempty"` //消息状态码(控制码)
	Type byte   `json:"type"`           //消息类型,读写开关
	Key  string `json:"key"`            //消息标识,远程的地址
	Data []byte `json:"data"`           //消息内容
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

// NeedAck 是否需要确认,当为请求时生效
func (this *Packet) NeedAck() bool {
	return this.Code&0x20 == 0x20
}

func (this *Packet) Bytes() []byte {
	bs := make([]byte, len(this.Data)+len(this.Key)+3)
	bs[0] = this.Code
	bs[1] = this.Type
	copy(bs[2:], []byte(this.Key))
	bs[len(this.Key)+2] = Split
	copy(bs[len(this.Key)+3:], this.Data)
	return bs
}

func Decode(bs []byte) (Packet, error) {
	list := bytes.SplitN(bs, []byte{Split}, 2)
	if len(list) == 1 || len(list[0]) < 2 {
		return Packet{}, errors.New("数据异常: " + string(bs))
	}
	return Packet{
		Code: list[0][0],
		Type: list[0][1],
		Key:  string(list[0][2:]),
		Data: list[1],
	}, nil
}
