// Package core 提供隧道代理的核心功能
package core

import (
	"encoding/json"
	"io"
	"net"
	"time"

	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
)

// Dialer 连接拨号器接口,用于创建到目标地址的连接
type Dialer interface {
	// Dial 建立连接,返回 (连接对象, 唯一标识, 错误)
	Dial() (io.ReadWriteCloser, string, error)
}

// NewDialTCP 创建一个 TCP 类型的拨号器
// address 格式为 "host:port",例如 "192.168.1.100:8080"
// timeout 为可选参数,指定连接超时时间
func NewDialTCP(address string, timeout ...time.Duration) *Dial {
	return &Dial{
		Type:    "tcp",
		Address: address,
		Timeout: conv.Default(0, timeout...),
	}
}

// Dial 连接配置,描述如何建立一条到目标地址的连接
type Dial struct {
	Type    string         `json:"type,omitempty"`    // Type 连接类型,支持 tcp/udp/websocket/serial 等
	Address string         `json:"address"`           // Address 目标地址,格式如 "192.168.1.100:8080"
	Timeout time.Duration  `json:"timeout,omitempty"` // Timeout 连接超时时间
	Param   map[string]any `json:"param,omitempty"`   // Param 其他自定义参数
}

// Dial 根据配置建立连接
// 返回 (连接对象, 本地地址字符串, 错误)
func (this *Dial) Dial() (io.ReadWriteCloser, string, error) {
	switch this.Type {
	default:
		c, err := net.DialTimeout("tcp", this.Address, this.Timeout)
		if err != nil {
			return nil, "", err
		}
		return c, c.LocalAddr().String(), nil
	}
}

// DialRes 连接响应,服务端在成功建立连接后返回给客户端
type DialRes struct {
	Key   string `json:"key,omitempty"` // Key 虚拟IO的唯一标识
	*Dial        // Dial 连接配置信息
}

// Listen 监听器配置,描述服务端如何监听客户端连接

// RegisterReq 客户端注册请求,客户端连接服务端后发送此消息进行注册
type RegisterReq struct {
	Listen   *Listen        `json:"listen,omitempty"`   // Listen 客户端需要监听的端口配置,可选
	Key      string         `json:"key"`                // Key 客户端唯一标识
	Username string         `json:"username,omitempty"` // Username 用户名,用于认证
	Password string         `json:"password,omitempty"` // Password 密码,用于认证
	Param    map[string]any `json:"param,omitempty"`    // Param 其他自定义参数
}

// Extend 将 RegisterReq 转换为扩展版本 RegisterReqExtend
// 扩展版本包含更多运行时信息,如 OnProxy 回调
func (this *RegisterReq) Extend() *RegisterReqExtend {
	return &RegisterReqExtend{
		Listen:   this.Listen,
		Key:      this.Key,
		Username: this.Username,
		Password: this.Password,
		Param:    this.Param,
		Extend:   conv.NewExtend(this),
	}
}

// String 将注册请求序列化为 JSON 字符串,用于日志输出
func (this *RegisterReq) String() string {
	bs, err := json.Marshal(this)
	logs.PrintErr(err)
	return string(bs)
}

// GetVar 根据 key 获取对应的注册信息值
// 支持 key/username/password 以及 Param 中的自定义参数
func (this *RegisterReq) GetVar(key string) *conv.Var {
	switch key {
	case "key":
		return conv.New(this.Key)
	case "username":
		return conv.New(this.Username)
	case "password":
		return conv.New(this.Password)
	default:
		if this.Param != nil {
			return conv.New(this.Param[key])
		}
	}
	return conv.Nil()
}

// RegisterReqExtend 注册请求的扩展版本,包含运行时信息
// 在服务端处理注册时使用,可以设置 OnProxy 回调来控制代理行为
type RegisterReqExtend struct {
	Listen      *Listen                                           `json:"listen,omitempty"`   // Listen 客户端监听的端口配置
	Key         string                                            `json:"key,omitempty"`      // Key 客户端唯一标识
	Username    string                                            `json:"username,omitempty"` // Username 用户名
	Password    string                                            `json:"password,omitempty"` // Password 密码
	Param       map[string]any                                    `json:"param,omitempty"`    // Param 其他自定义参数
	conv.Extend `json:"-"`                                        // Extend 扩展属性容器
	OnProxy     func(r io.ReadWriteCloser) (*Dial, []byte, error) // OnProxy 代理回调,用于控制外部连接如何转发到隧道
}
