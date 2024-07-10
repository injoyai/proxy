package tunnel

import (
	"encoding/json"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"net"
	"time"
)

type Tunnel struct {
	Port         int                                            //客户端连接的端口
	OnRegister   func(c net.Conn, r *virtual.RegisterReq) error //注册事件
	Timeout      time.Duration                                  //超时时间
	ProxyAddress string                                         //代理地址
}

func (this *Tunnel) ListenTCP() error {
	return core.RunListen("tcp", this.Port, core.WithListenLog, this.Handler)
}

// Handler 对客户端进行注册验证操作
func (this *Tunnel) Handler(tunListen net.Listener, c net.Conn) error {

	var listener net.Listener

	v := virtual.New(c, virtual.WithRegister(func(v *virtual.Virtual, p virtual.Packet) error {
		//解析注册数据
		register := new(virtual.RegisterReq)
		err := json.Unmarshal(p.GetData(), register)
		if err != nil {
			return err
		}
		//注册事件
		if this.OnRegister != nil {
			if err := this.OnRegister(c, register); err != nil {
				return err
			}
		}
		{ //监听
			listener, err = core.GoListen("tcp", register.Port, func(listener net.Listener, c net.Conn) (err error) {
				logs.Tracef("[%s] 新的连接\n", c.RemoteAddr().String())
				defer logs.Tracef("[%s] 关闭连接\n", c.RemoteAddr().String())
				return v.OpenAndSwap(this.ProxyAddress, c)
			})
			if err != nil {
				logs.Errf("[%s] 监听端口[:%d]失败: %s\n", p.GetKey(), register.Port, err.Error())
				return err
			}

			logs.Infof("[:%d] 开始监听...\n", register.Port)

		}
		return nil
	}))

	defer func() {
		c.Close()
		v.Close()
		if listener != nil {
			listener.Close()
			logs.Infof("[%s] 关闭监听...\n", listener.Addr().String())
		}
	}()

	return v.Run()
}

func (this *Tunnel) handlerListen(port int, v *virtual.Virtual) (net.Listener, error) {
	return core.GoListen("tcp", port, func(listener net.Listener, c net.Conn) (err error) {
		defer c.Close()
		return v.OpenAndSwap("", c)
	})
}
