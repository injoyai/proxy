package tunnel

import (
	"encoding/json"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"net"
	"time"
)

type Tunnel struct {
	Port         int                                    //客户端连接的端口
	OnRegister   func(c net.Conn, r *RegisterReq) error //注册事件
	Timeout      time.Duration                          //超时时间
	ProxyAddress string                                 //代理地址
}

func (this *Tunnel) ListenTCP() error {
	return core.RunListen("tcp", this.Port, core.WithListenLog, this.Handler)
}

// Handler 对客户端进行注册验证操作
func (this *Tunnel) Handler(tunListen net.Listener, c net.Conn) error {
	defer c.Close()

	tun := NewConn(c, this.Timeout)

	//读取注册数据
	c.SetDeadline(time.Now().Add(this.Timeout))
	p, err := tun.ReadPacket()
	if err != nil {
		return err
	}
	c.SetDeadline(time.Time{})

	//解析注册数据
	register := new(RegisterReq)
	if err := json.Unmarshal(p.Data, register); err != nil {
		return err
	}

	//注册事件
	if this.OnRegister != nil {
		if err := this.OnRegister(tun.c, register); err != nil {
			logs.Errf("[%s] 注册失败: %s\n", tun.Key(), err.Error())
			tun.WritePacket(p.Resp(Fail, err))
			return err
		}
	}

	{
		l, err := this.handlerListen(register.Port, tun)
		if err != nil {
			logs.Errf("[%s] 监听端口[:%d]失败: %s\n", tun.Key(), register.Port, err.Error())
			tun.WritePacket(p.Resp(Fail, err))
			return err
		}
		logs.Infof("[:%d] 开始监听...\n", register.Port)
		defer logs.Infof("[:%d] 关闭监听...\n", register.Port)
		defer l.Close()
	}

	//响应注册成功
	tun.WritePacket(p.Resp(Success, "注册成功"))
	return tun.runRead(this.ProxyAddress)
}

func (this *Tunnel) handlerListen(port int, tun *Conn) (net.Listener, error) {
	return core.GoListen("tcp", port, func(listener net.Listener, c net.Conn) (err error) {
		return tun.Swap(c, func(tun *Conn, c net.Conn, key string) error {
			logs.Tracef("[%s] 发起建立连接请求 \n", key)
			//发起建立连接请求
			if _, err := tun.WritePacket(&Packet{Type: Open, Key: key}); err != nil {
				return err
			}
			logs.Tracef("[%s] 等待建立连接结果 \n", key)
			//等待建立连接结果
			if _, err := tun.wait.Wait(key, this.Timeout); err != nil {
				return err
			}
			return nil
		})
	})
}
