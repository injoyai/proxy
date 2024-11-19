package tunnel

import (
	"encoding/json"
	"github.com/injoyai/conv"
	"github.com/injoyai/proxy/core"
)

type Client struct {
	Dialer   core.Dialer       //连接配置
	Register *core.RegisterReq //注册配置
	tunnel   *core.Tunnel      //隧道实例
}

func (this *Client) Tunnel() *core.Tunnel {
	return this.tunnel
}

func (this *Client) Close() error {
	if this.tunnel != nil {
		return this.tunnel.Close()
	}
	return nil
}

func (this *Client) Run(op ...core.OptionTunnel) error {
	err := this.Dial(op...)
	if err != nil {
		return err
	}
	<-this.Tunnel().Done()
	return this.Tunnel().Err()
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
	this.tunnel = core.NewTunnel(c)
	this.tunnel.SetKey(k)
	this.tunnel.SetOption(core.WithDialed(func(d *core.Dial, key string) {
		if this.Register == nil || this.Register.Listen == nil || this.Register.Listen.Port == "" {
			core.DefaultLog.Infof("[桥接 -> 隧道[%s] -> 请求[%s]\n", this.tunnel.Key(), d.Address)
			return
		}
		core.DefaultLog.Infof("监听[:%s] -> 隧道[%s] -> 请求[%s]\n", this.Register.Listen.Port, this.tunnel.Key(), d.Address)
	}))
	this.tunnel.SetOption(op...)
	go this.tunnel.Run()

	//注册到服务
	resp, err := this.tunnel.Register(this.Register)
	if err != nil {
		//注册失败则关闭虚拟通道
		this.tunnel.CloseWithErr(err)
		return err
	}
	if err := json.Unmarshal(conv.Bytes(resp), &this.Register.Listen); err != nil {
		//可能返回空字符,则解析失败
		//return err
	}
	core.DefaultLog.Infof("[%s] 注册至服务成功...\n", k) // this.tunnel.Key())

	return nil
}
