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
	Dialer   core.Dialer          //连接配置
	Register *virtual.RegisterReq //注册配置
	virtual  *virtual.Virtual     //虚拟设备管理
}

func (this *Client) Close() error {
	if this.virtual != nil {
		return this.virtual.Close()
	}
	return nil
}

func (this *Client) Dial(op ...virtual.Option) error {

	//连接到服务端
	c, k, err := this.Dialer.Dial()
	if err != nil {
		return err
	}
	defer c.Close()

	//如果存在则关闭老的
	this.Close()

	//虚拟设备管理,默认使用服务的代理配置代理
	this.virtual = virtual.New(c)

	this.virtual.SetKey(k)
	this.virtual.SetOption(virtual.WithOpened(func(p virtual.Packet, d *core.Dial, key string) {
		logs.Infof("[%s -> :%s] 代理至 [%s -> %s]\n", p.GetKey(), this.Register.Listen.Port, this.virtual.Key(), d.Address)
	}))
	this.virtual.SetOption(op...)
	defer this.virtual.Close()

	go this.virtual.Run()

	//注册到服务
	resp, err := this.virtual.Register(this.Register)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(conv.Bytes(resp), &this.Register.Listen); err != nil {
		return err
	}
	logs.Infof("[%s] 注册至服务成功...\n", this.virtual.Key())

	<-this.virtual.Done()
	return this.virtual.Err()
}
