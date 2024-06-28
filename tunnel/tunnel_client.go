package tunnel

import (
	"errors"
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"net"
	"time"
)

type Client struct {
	Address      string        //服务地址
	Port         int           //监听端口
	ProxyAddress string        //代理地址
	Username     string        //用户名
	Password     string        //密码
	Timeout      time.Duration //超时时间
}

func (this *Client) Dial() error {
	if this.Timeout <= 0 {
		this.Timeout = time.Second * 2
	}

	c, err := net.DialTimeout("tcp", this.Address, this.Timeout)
	if err != nil {
		return err
	}
	defer c.Close()

	tun := NewConn(c, this.Timeout)

	//进行注册认证
	if err := this.Register(tun); err != nil {
		logs.Trace("注册到服务错误: ", err.Error())
		return err
	}
	logs.Infof("[%s] 注册到服务成功...\n", tun.Key())

	return tun.runRead(this.ProxyAddress)

}

func (this *Client) Register(tun *Conn) error {

	_, err := tun.WritePacket(&Packet{
		Key:  tun.c.LocalAddr().String(),
		Code: Request | NeedAck,
		Type: Register,
		Data: conv.Bytes(RegisterReq{
			Port:     this.Port,
			Username: this.Username,
			Password: this.Password,
		}),
	})
	if err != nil {
		return err
	}

	_ = tun.c.SetReadDeadline(time.Now().Add(this.Timeout))
	p, err := tun.ReadPacket()
	if err != nil {
		logs.Trace(err)
		return err
	}
	_ = tun.c.SetReadDeadline(time.Time{})

	if p.IsResponse() && p.Success() {
		return nil
	}

	return errors.New(string(p.Data))
}

func (this *Client) handlerOpen(c net.Conn) {
	_, _ = core.Write(c, &Packet{
		Type: Register,
		Data: conv.Bytes(RegisterReq{
			Port:     this.Port,
			Username: this.Username,
			Password: this.Password,
		}),
	})
}

type RegisterReq struct {
	Port     int    //监听端口
	Username string //用户名
	Password string //密码
}
