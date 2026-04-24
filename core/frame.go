// Package core 提供隧道代理的核心功能,包括帧协议、虚拟IO、隧道管理等
package core

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/injoyai/conv"
)

// 消息类型常量,定义隧道中传输的各种操作类型
const (
	Register = 0x00 // Register 注册消息,客户端向服务端注册身份
	Open     = 0x01 // Open 打开连接,请求建立一条新的虚拟通道
	Close    = 0x02 // Close 关闭连接,通知对端关闭某条虚拟通道
	Read     = 0x03 // Read 读取数据,从虚拟IO中读取数据
	Write    = 0x04 // Write 写入数据,向虚拟IO中写入数据
)

// 控制码常量,用于标识消息的方向和状态
const (
	Request  = 0x00 // Request 请求包,由发起方发送
	Response = 0x80 // Response 响应包,由接收方回复
	Success  = 0x00 // Success 成功状态,仅用于响应包
	Fail     = 0x40 // Fail 失败状态,仅用于响应包
	NeedAck  = 0x20 // NeedAck 需要确认,请求包设置此标志表示需要对方回复
)

// 帧协议常量
const (
	FramePrefix1   = 0x89 // FramePrefix1 帧头第一个字节
	FramePrefix2   = 0x89 // FramePrefix2 帧头第二个字节
	FrameDelimiter = '#'  // FrameDelimiter 帧内字段分隔符
)

// DefaultFrame 默认帧协议实例,用于全局复用
var DefaultFrame = &frame{}

// Frame 帧协议接口,定义数据包的编解码和读写操作
type Frame interface {
	// NewPacket 创建一个新的数据包
	NewPacket(k string, t byte, i interface{}) Packet
	// ReadPacket 从Reader中读取并解析一个完整的数据包
	ReadPacket(r io.Reader) (Packet, error)
}

// Packet 数据包接口,定义数据包的访问方法
type Packet interface {
	Bytes() []byte    // Bytes 将数据包序列化为字节数组
	GetMsgID() string // GetMsgID 获取消息唯一标识
	GetData() []byte  // GetData 获取数据包负载数据
	GetType() byte    // GetType 获取消息类型(Register/Open/Close/Read/Write)
	IsRequest() bool  // IsRequest 判断是否为请求包
	Success() bool    // Success 判断响应是否成功
	NeedAck() bool    // NeedAck 判断是否需要对方确认
}

// packet 数据包的具体实现,内部结构
type packet struct {
	MsgID string // MsgID 消息唯一标识,用于关联请求和响应
	Data  []byte // Data 负载数据
	Code  byte   // Code 控制码,包含消息类型、方向、状态等信息
}

// String 实现Stringer接口,用于日志输出
func (this *packet) String() string {
	return fmt.Sprintf("[%s] 类型: %s, 控制码: %s, 数据: %s",
		this.MsgID,
		func() string {
			switch this.GetType() {
			case Register:
				return "注册"
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
			ls = append(ls, conv.Select(this.IsRequest(), "请求", "响应"))
			if this.IsRequest() {
				ls = append(ls, conv.Select(this.NeedAck(), "需要确认", "无需确认"))
			} else {
				ls = append(ls, conv.Select(this.Success(), "成功", "失败"))
			}
			return strings.Join(ls, "|")
		}(),
		this.Data,
	)
}

// IsRequest 判断是否为请求包,通过检查Response位是否为0
func (this *packet) IsRequest() bool {
	return this.Code&Response == 0x00
}

// Success 判断响应是否成功,通过检查Fail位是否为0
func (this *packet) Success() bool {
	return this.Code&Fail == 0x00
}

// NeedAck 判断是否需要确认,仅对请求包有效
func (this *packet) NeedAck() bool {
	return this.Code&NeedAck == NeedAck
}

// GetType 获取消息类型,取低4位
func (this *packet) GetType() byte {
	return this.Code & 0x0F
}

// Bytes 将数据包序列化为字节数组
// 格式: [0x89][0x89][长度4字节][MsgID][#][Code][Data]
func (this *packet) Bytes() []byte {
	lenData := len(this.Data)
	lenKey := len(this.MsgID)
	bs := make([]byte, len(this.Data)+lenKey+8)
	copy(bs[0:2], []byte{FramePrefix1, FramePrefix2})
	copy(bs[2:6], conv.Bytes(uint32(lenKey+2+lenData)))
	copy(bs[6:len(this.MsgID)+6], this.MsgID)
	bs[lenKey+6] = FrameDelimiter
	bs[lenKey+7] = this.Code
	copy(bs[lenKey+8:], this.Data)
	return bs
}

// GetMsgID 获取消息唯一标识
func (this *packet) GetMsgID() string {
	return this.MsgID
}

// GetData 获取负载数据
func (this *packet) GetData() []byte {
	return this.Data
}

// frame 帧协议的具体实现
type frame struct{}

// NewPacket 创建一个新的数据包
func (this *frame) NewPacket(mid string, t byte, d interface{}) Packet {
	return &packet{
		MsgID: mid,
		Data:  conv.Bytes(d),
		Code:  t,
	}
}

// WritePacket 将数据包写入Writer
func (this *frame) WritePacket(w io.Writer, p Packet) error {
	_, err := w.Write(p.Bytes())
	return err
}

// ReadPacket 从Reader中读取并解析一个完整的数据包
func (this *frame) ReadPacket(r io.Reader) (Packet, error) {
	bs, err := this.Read(r)
	if err != nil {
		return nil, err
	}
	p, err := this.Decode(bs)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Decode 将字节数组解码为数据包
func (this *frame) Decode(bs []byte) (Packet, error) {
	if len(bs) < 2 {
		return nil, fmt.Errorf("基础长度错误,预期至少2字节,得到%d", len(bs))
	}
	list := bytes.SplitN(bs, []byte{FrameDelimiter}, 2)
	if len(list) != 2 {
		return nil, fmt.Errorf("数据分割异常: %v", bs)
	}
	if len(list[1]) == 0 {
		return nil, fmt.Errorf("数据类型异常: %v", bs)
	}
	return &packet{
		MsgID: string(list[0]),
		Code:  list[1][0],
		Data:  list[1][1:],
	}, nil
}

// Read 从Reader中读取一帧完整数据
// 帧格式: [0x89][0x89][长度4字节][数据域]
// 通过循环查找帧头实现自动分包
func (this *frame) Read(r io.Reader) ([]byte, error) {

	for {

		// 校验帧头标识 0x8989
		bufPrefix := make([]byte, 2)
		n, err := r.Read(bufPrefix)
		if err != nil {
			return nil, err
		}
		if n != 2 || bufPrefix[0] != 0x89 || bufPrefix[1] != 0x89 {
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
