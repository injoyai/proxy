package proxy

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core/virtual"
	"io"
	"net"
	"time"
)

type Client struct {
	Address  string                                                     //连接服务地址
	Timeout  time.Duration                                              //连接诶服务超时时间
	Register virtual.RegisterReq                                        //注册配置
	OnOpen   func(p virtual.Packet) (io.ReadWriteCloser, string, error) //打开连接事件
}

func (this *Client) DialTCP(op ...virtual.Option) error {
	if this.Timeout <= 0 {
		this.Timeout = time.Second * 2
	}

	//连接到服务端
	c, err := net.DialTimeout("tcp", this.Address, this.Timeout)
	if err != nil {
		return err
	}
	defer c.Close()

	//虚拟设备管理,默认使用服务的代理配置代理
	v := virtual.New(c, virtual.WithOpen(this.OnOpen))
	v.SetOption(op...)
	defer v.Close()

	go v.Run()

	//注册到服务
	if err := v.Register(this.Register); err != nil {
		return err
	}
	logs.Trace("注册成功")

	<-v.Done()
	return v.Err()
}
