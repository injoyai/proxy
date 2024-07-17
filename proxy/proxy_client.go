package proxy

import (
	"encoding/json"
	"github.com/injoyai/conv"
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
	c, k, err := this.Dial.Dial()
	if err != nil {
		return err
	}
	defer c.Close()

	//虚拟设备管理,默认使用服务的代理配置代理
	v := virtual.New(c)
	v.SetKey(k)
	v.SetOption(virtual.WithOpened(func(p virtual.Packet, d *core.Dial, key string) {
		logs.Infof("[%s -> :%s] 代理至 [%s -> %s]\n", p.GetKey(), this.Register.Listen.Port, v.Key(), d.Address)
	}))
	v.SetOption(op...)
	defer v.Close()

	go v.Run()

	//注册到服务
	resp, err := v.Register(this.Register)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(conv.Bytes(resp), &this.Register.Listen); err != nil {
		return err
	}
	logs.Infof("[%s] 注册至[%s]成功...\n", v.Key(), this.Dial.Address)

	<-v.Done()
	return v.Err()
}
