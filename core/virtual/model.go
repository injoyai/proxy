package virtual

import (
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/g"
	"io"
	"net"
	"time"
)

type RegisterReq struct {
	Port     int    `json:"port"`     //监听端口
	Username string `json:"username"` //用户名
	Password string `json:"password"` //密码
	Param    g.Map  `json:"param"`    //其他参数
}

func (this *RegisterReq) String() string {
	return fmt.Sprintf("监听: %d, 用户名: %s, 密码: %s", this.Port, this.Username, this.Password)
}

type Proxy struct {
	Type    string        `json:"type"`    //代理类型,例如TCP,UDP
	Address string        `json:"address"` //代理地址,默认地址,可被OnProxy覆盖
	Timeout time.Duration `json:"timeout"` //超时时间
	Param   g.Map         `json:"param"`   //其他配置
}

func (this *Proxy) Option() Option {
	return WithOpen(func(p Packet) (io.ReadWriteCloser, string, error) {
		if this == nil {
			//使用远程的代理配置进行代理
			m := conv.NewMap(p.GetData())
			proxy := &Proxy{
				Type:    m.GetString("type", "tcp"),
				Address: m.GetString("address"),
				Timeout: m.GetDuration("timeout"),
				Param:   m.GMap(),
			}
			return proxy.Dial()
		}
		//使用本地代理配置进行代理
		return this.Dial()
	})
}

// Dial 进行连接,不限于UDP,TCP,Websocket,Serial
func (this *Proxy) Dial() (io.ReadWriteCloser, string, error) {
	switch this.Type {
	case "tcp":
		c, err := net.DialTimeout(this.Type, this.Address, this.Timeout)
		if err != nil {
			return nil, "", err
		}
		return c, c.LocalAddr().String(), nil

	//case "udp":
	//case "websocker","ws":
	//case "serial":

	default:
		c, err := net.DialTimeout("tcp", this.Address, this.Timeout)
		if err != nil {
			return nil, "", err
		}
		return c, c.LocalAddr().String(), nil
	}
}
