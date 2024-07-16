package proxy

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
)

type Client struct {
	SN       string               //唯一标识符
	Dial     *core.Dial           //连接配置
	Register *virtual.RegisterReq //注册配置
}

func (this *Client) DialTCP(op ...virtual.Option) error {
	//连接到服务端
	c, _, err := this.Dial.Dial()
	if err != nil {
		return err
	}
	defer c.Close()

	//虚拟设备管理,默认使用服务的代理配置代理
	v := virtual.New(c, op...)
	defer v.Close()

	go v.Run()

	//注册到服务
	if err := v.Register(this.Register); err != nil {
		return err
	}
	logs.Infof("[%s] 注册成功\n", this.Dial.Address)

	<-v.Done()
	return v.Err()
}
