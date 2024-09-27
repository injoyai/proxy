package tunnel

import (
	"encoding/json"
	"github.com/injoyai/conv"
	"github.com/injoyai/proxy/core"
)

type Client struct {
	Dialer   core.Dialer       //连接配置
	Register *core.RegisterReq //注册配置
	virtual  *core.Tunnel      //虚拟设备管理
}

func (this *Client) Virtual() *core.Tunnel {
	return this.virtual
}

func (this *Client) Close() error {
	if this.virtual != nil {
		return this.virtual.Close()
	}
	return nil
}

func (this *Client) Run(op ...core.OptionTunnel) error {
	err := this.Dial(op...)
	if err != nil {
		return err
	}
	<-this.Virtual().Done()
	return this.Virtual().Err()
}

func (this *Client) Dial(op ...core.OptionTunnel) error {

	//连接到服务端
	c, k, err := this.Dialer.Dial()
	if err != nil {
		return err
	}

	//如果存在则关闭老的
	this.Close()

	//虚拟设备管理,默认使用服务的代理配置代理
	this.virtual = core.NewTunnel(c)
	this.virtual.SetKey(k)
	this.virtual.SetOption(core.WithDialed(func(p core.Packet, d *core.Dial, key string) {
		if this.Register == nil || this.Register.Listen == nil || this.Register.Listen.Port == "" {
			core.DefaultLog.Infof("[%s] 代理至 [%s -> %s]\n", p.GetKey(), this.virtual.Key(), d.Address)
			return
		}
		core.DefaultLog.Infof("[%s -> :%s] 代理至 [%s -> %s]\n", p.GetKey(), this.Register.Listen.Port, this.virtual.Key(), d.Address)
	}))
	this.virtual.SetOption(op...)
	go this.virtual.Run()

	//注册到服务
	resp, err := this.virtual.Register(this.Register)
	if err != nil {
		//注册失败则关闭虚拟通道
		this.virtual.CloseWithErr(err)
		return err
	}
	if err := json.Unmarshal(conv.Bytes(resp), &this.Register.Listen); err != nil {
		//可能返回空字符,则解析失败
		//return err
	}
	core.DefaultLog.Infof("[%s] 注册至服务成功...\n", this.virtual.Key())

	return nil
}
