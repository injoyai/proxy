package core

import (
	"encoding/json"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
)

type DialFunc func() (io.ReadWriteCloser, string, error)

func (this DialFunc) Dial() (io.ReadWriteCloser, string, error) { return this() }

type Dialer interface {
	Dial() (io.ReadWriteCloser, string, error)
}

func NewDialTCP(address string, timeout ...time.Duration) *Dial {
	return &Dial{
		Type:    "tcp",
		Address: address,
		Timeout: conv.Default(0, timeout...),
	}
}

type Dial struct {
	Type    string         `json:"type,omitempty"`    //连接类型,TCP,UDP,Websocket,Serial...
	Address string         `json:"address"`           //连接地址
	Timeout time.Duration  `json:"timeout,omitempty"` //超时时间
	Param   map[string]any `json:"param,omitempty"`   //其他参数
}

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

type DialRes struct {
	Key string `json:"key,omitempty"`
	*Dial
}

func NewListenTCP(port int) *Listen {
	return &Listen{Port: strconv.Itoa(port)}
}

type Listen struct {
	Type  string         `json:"type,omitempty"`  //类型,TCP,UDP,Serial等
	Port  string         `json:"port"`            //例如串口是字符的,固使用字符类型
	Param map[string]any `json:"param,omitempty"` //其他参数
}

type RegisterReq struct {
	Listen   *Listen        `json:"listen,omitempty"`   //监听信息
	Key      string         `json:"key"`                //唯一标识
	Username string         `json:"username,omitempty"` //用户名
	Password string         `json:"password,omitempty"` //密码
	Param    map[string]any `json:"param,omitempty"`    //其他参数
}

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

func (this *RegisterReq) String() string {
	bs, err := json.Marshal(this)
	logs.PrintErr(err)
	return string(bs)
}

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

type RegisterReqExtend struct {
	Listen      *Listen        `json:"listen,omitempty"`   //监听信息
	Key         string         `json:"key,omitempty"`      //唯一标识
	Username    string         `json:"username,omitempty"` //用户名
	Password    string         `json:"password,omitempty"` //密码
	Param       map[string]any `json:"param,omitempty"`    //其他参数
	conv.Extend `json:"-"`
	OnProxy     func(r io.ReadWriteCloser) (*Dial, []byte, error)
}
