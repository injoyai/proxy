package core

import (
	"bytes"
	"fmt"
	"io"

	"github.com/injoyai/conv"
)

var DefaultFrame Frame = &frameV1{}

// 帧协议常量
const (
	Prefix    = 0x89 // FramePrefix1 帧头第一个字节
	Delimiter = '#'  // FrameDelimiter 帧内字段分隔符
)

type frameV1 struct{}

func (this *frameV1) NewPacket(msgID string, _type Type, tags Tags, data any) []byte {
	return packetV1{
		MsgID: msgID,
		Data:  conv.Bytes(data),
		Code:  uint8(_type) | uint8(tags[0]|tags[1]|tags[2]),
	}.Bytes()
}

func (this *frameV1) ReadPacket(r io.Reader) (msgID string, _type Type, tags Tags, data []byte, err error) {
	data, err = this.read(r)
	if err != nil {
		return
	}
	var p *packetV1
	p, err = this.decode(data)
	if err != nil {
		return
	}
	tags = Tags{
		Tag(p.Code & 0x80),
		Tag(p.Code & 0x40),
		Tag(p.Code & 0x20),
	}

	return p.MsgID, Type(p.Code & 0x0F), tags, p.Data, nil
}

// Decode 将字节数组解码为数据包
func (this *frameV1) decode(bs []byte) (*packetV1, error) {
	if len(bs) < 2 {
		return nil, fmt.Errorf("基础长度错误,预期至少2字节,得到%d", len(bs))
	}
	list := bytes.SplitN(bs, []byte{Delimiter}, 2)
	if len(list) != 2 {
		return nil, fmt.Errorf("数据分割异常: %v", bs)
	}
	if len(list[1]) == 0 {
		return nil, fmt.Errorf("数据类型异常: %v", bs)
	}
	return &packetV1{
		MsgID: string(list[0]),
		Code:  list[1][0],
		Data:  list[1][1:],
	}, nil
}

// Read 从Reader中读取一帧完整数据
// 帧格式: [0x89][0x89][长度4字节][数据域]
// 通过循环查找帧头实现自动分包
func (this *frameV1) read(r io.Reader) ([]byte, error) {

	for {

		// 校验帧头标识 0x8989
		bufPrefix := make([]byte, 2)
		n, err := r.Read(bufPrefix)
		if err != nil {
			return nil, err
		}
		if n != 2 || bufPrefix[0] != Prefix || bufPrefix[1] != Prefix {
			continue
		}

		// 读取数据域长度(4字节)
		bufLength := make([]byte, 4)
		n, err = r.Read(bufLength)
		if err != nil {
			return nil, err
		}
		if n != 4 {
			continue
		}
		length := conv.Int64(bufLength)

		// 读取数据域
		bufData := make([]byte, length)
		n, err = io.ReadAtLeast(r, bufData, int(length))
		if err != nil {
			return nil, err
		}
		if int64(n) != length {
			continue
		}

		return bufData, nil
	}
}

type packetV1 struct {
	MsgID string // MsgID 消息唯一标识,用于关联请求和响应
	Data  []byte // Data 负载数据
	Code  uint8  // Code 控制码,包含消息类型、方向、状态等信息
}

// Bytes 将数据包序列化为字节数组
// 格式: [0x89][0x89][长度4字节][MsgID][#][Code][Data]
func (this packetV1) Bytes() []byte {
	lenData := len(this.Data)
	lenKey := len(this.MsgID)
	bs := make([]byte, len(this.Data)+lenKey+8)
	copy(bs[0:2], []byte{Prefix, Prefix})
	copy(bs[2:6], conv.Bytes(uint32(lenKey+2+lenData)))
	copy(bs[6:len(this.MsgID)+6], this.MsgID)
	bs[lenKey+6] = Delimiter
	bs[lenKey+7] = uint8(this.Code)
	copy(bs[lenKey+8:], this.Data)
	return bs
}
