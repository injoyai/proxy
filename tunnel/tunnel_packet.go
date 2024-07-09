package tunnel

//
//import (
//	"bytes"
//	"errors"
//	"fmt"
//	"github.com/injoyai/conv"
//	"strings"
//)
//
//const (
//	Write    = 'w' //写入数据
//	Open     = 'o' //建立连接,缩写和Close相同,固改成Open
//	Close    = 'c' //关闭连接
//	Register = 'r' //注册
//	Split    = '#' //分隔符
//
//	Request  = 0x00 //请求
//	Response = 0x80 //响应
//	Success  = 0x00 //成功
//	Fail     = 0x40 //失败
//	NeedAck  = 0x20 //需要响应
//)
//
//type Packet struct {
//	//10000001
//	//第一位是方向,0是请求,1是响应
//	//第二位是结果,0是成功,1是失败
//	//第三位是是否需要响应,0是不需要,1是需要
//	//第4~8位预留
//	Code byte   `json:"code,omitempty"` //消息状态码(控制码)
//	Type byte   `json:"type"`           //消息类型,读写开关
//	Key  string `json:"key"`            //消息标识,远程的地址
//	Data []byte `json:"data"`           //消息内容
//}
//
//func (this *Packet) Err() error {
//	return errors.New(string(this.Data))
//}
//
//func (this *Packet) String() string {
//	return fmt.Sprintf("[%s] 类型: %s, 结果: %s, 数据: %s",
//		this.Key,
//		func() string {
//			switch this.Type {
//			case Write:
//				return "写入数据"
//			case Open:
//				return "建立连接"
//			case Close:
//				return "关闭连接"
//			case Register:
//				return "注册代理"
//			default:
//				return "未知"
//			}
//		}(),
//		func() string {
//			ls := []string(nil)
//			ls = append(ls, conv.SelectString(this.IsRequest(), "请求", "响应"))
//			if this.IsRequest() {
//				ls = append(ls, conv.SelectString(this.NeedAck(), "需要确认", "无需确认"))
//			}
//			if this.IsResponse() {
//				ls = append(ls, conv.SelectString(this.Success(), "成功", "失败"))
//			}
//			return strings.Join(ls, "|")
//		}(),
//		this.Data,
//	)
//}
//
//func (this *Packet) Resp(code byte, data any) *Packet {
//	return &Packet{
//		Code: Response | code,
//		Type: this.Type,
//		Key:  this.Key,
//		Data: conv.Bytes(data),
//	}
//}
//
//// IsRequest 是否是请求数据
//func (this *Packet) IsRequest() bool {
//	return this.Code&0x80 == Request
//}
//
//// IsResponse 是否是响应数据
//func (this *Packet) IsResponse() bool {
//	return this.Code&0x80 == Response
//}
//
//// Success 是否成功,当为响应时生效
//func (this *Packet) Success() bool {
//	return this.Code&0x40 == Success
//}
//
//// NeedAck 是否需要确认,当为请求时生效
//func (this *Packet) NeedAck() bool {
//	return this.Code&0x20 == NeedAck
//}
//
//func (this *Packet) Bytes() []byte {
//	bs := make([]byte, len(this.Data)+len(this.Key)+3)
//	bs[0] = this.Code
//	bs[1] = this.Type
//	copy(bs[2:], []byte(this.Key))
//	bs[len(this.Key)+2] = Split
//	copy(bs[len(this.Key)+3:], this.Data)
//	return bs
//}
//
//func Decode(bs []byte) (*Packet, error) {
//	list := bytes.SplitN(bs, []byte{Split}, 2)
//	if len(list) == 1 || len(list[0]) < 2 {
//		return nil, errors.New("数据异常: " + string(bs))
//	}
//	return &Packet{
//		Code: list[0][0],
//		Type: list[0][1],
//		Key:  string(list[0][2:]),
//		Data: list[1],
//	}, nil
//}
