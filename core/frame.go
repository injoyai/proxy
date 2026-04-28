package core

import "io"

type (
	Type uint8
	Tag  uint8
)

// 消息类型常量,定义隧道中传输的各种操作类型
const (
	Register Type = 0x00 // Register 注册消息,客户端向服务端注册身份
	Open     Type = 0x01 // Open 打开连接,请求建立一条新的虚拟通道
	Close    Type = 0x02 // Close 关闭连接,通知对端关闭某条虚拟通道
	Read     Type = 0x03 // Read 读取数据,从虚拟IO中读取数据
	Write    Type = 0x04 // Write 写入数据,向虚拟IO中写入数据
)

// 控制码常量,用于标识消息的方向和状态
const (
	Request  Tag = 0x00 // Request 请求包,由发起方发送
	Response Tag = 0x80 // Response 响应包,由接收方回复
	Success  Tag = 0x00 // Success 成功状态,仅用于响应包
	Fail     Tag = 0x40 // Fail 失败状态,仅用于响应包
	NeedAck  Tag = 0x20 // NeedAck 需要确认,请求包设置此标志表示需要对方回复
)

type Tags [3]Tag

// IsRequest 判断是否为请求包
func (this Tags) IsRequest() bool {
	return this[0]&Response == 0
}

// Success 判断响应是否成功,通过检查Fail位是否为0
func (this Tags) Success() bool {
	return this[1]&Fail == 0x00
}

// NeedAck 判断是否需要确认,仅对请求包有效
func (this Tags) NeedAck() bool {
	return this[2]&NeedAck == NeedAck
}

type Frame interface {
	NewPacket(msgID string, _type Type, tags Tags, data any) []byte
	ReadPacket(r io.Reader) (msgID string, _type Type, tags Tags, data []byte, err error)
}
