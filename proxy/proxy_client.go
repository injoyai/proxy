package proxy

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core/virtual"
	"io"
)

type Client struct {
	Dial     virtual.Dial                                               //连接配置
	Register virtual.RegisterReq                                        //注册配置
	OnOpen   func(p virtual.Packet) (io.ReadWriteCloser, string, error) //打开连接事件
}

func (this *Client) RunTCP(op ...virtual.Option) error {
	//连接到服务端
	c, _, err := this.Dial.Dial()
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
