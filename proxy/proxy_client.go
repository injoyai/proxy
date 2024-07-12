package proxy

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core/virtual"
	"net"
	"time"
)

type Client struct {
	Address  string        //服务地址
	Proxy    string        //代理地址
	Port     int           //监听端口
	Username string        //用户名
	Password string        //密码
	Timeout  time.Duration //超时时间
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

	//虚拟设备管理
	v := virtual.NewTCPDefault(c, this.Proxy)
	defer v.Close()

	go v.Run()

	if err := v.Register(virtual.RegisterReq{
		Port:     this.Port,
		Username: this.Username,
		Password: this.Password,
	}); err != nil {
		return err
	}
	logs.Trace("注册成功")

	<-v.Done()
	return v.Err()
}
