package proxy

import (
	"bytes"
	"encoding/json"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"io"
	"net"
)

type Server struct {
	Clients    *maps.Safe                                     //客户端
	Listen     *core.Listen                                   //监听配置
	OnRegister func(c net.Conn, r *virtual.RegisterReq) error //注册事件
	OnProxy    func(c net.Conn) (*core.Dial, []byte, error)   //代理事件
}

func (this *Server) Run() error {
	if this.Clients == nil {
		this.Clients = maps.NewSafe()
	}
	return this.Listen.Listen(core.WithListenLog, this.Handler)
}

// Handler 对客户端进行注册验证操作
func (this *Server) Handler(tunListen net.Listener, tun net.Conn) error {

	var listener net.Listener

	v := virtual.New(tun, virtual.WithRegister(func(v *virtual.Virtual, p virtual.Packet) error {
		//解析注册数据
		register := new(virtual.RegisterReq)
		err := json.Unmarshal(p.GetData(), register)
		if err != nil {
			return err
		}
		//注册事件
		if this.OnRegister != nil {
			if err = this.OnRegister(tun, register); err != nil {
				return err
			}
		}

		//判断客户端是否需要监听端口
		//客户端可以选择不监听端口,而由服务端进行安排
		if register.Listen == nil || register.Listen.Port == "" {
			return nil
		}

		{ //监听端口
			listener, err = register.Listen.GoListen(func(listener net.Listener, c net.Conn) error {
				logs.Tracef("[%s] 新的连接\n", c.RemoteAddr().String())
				defer logs.Tracef("[%s] 关闭连接: %v\n", c.RemoteAddr().String(), err)

				//使用自定义(服务端)代理
				if this.OnProxy != nil {
					proxy, prefix, err := this.OnProxy(c)
					if err != nil {
						return err
					}
					return v.OpenAndSwap(proxy, struct {
						io.Reader
						io.WriteCloser
					}{
						Reader:      io.MultiReader(bytes.NewReader(prefix), c),
						WriteCloser: c,
					})
				}

				//使用默认(客户端)代理
				return v.OpenAndSwap(&core.Dial{}, c)
			})
			if err != nil {
				logs.Errf("[%s] 监听端口[:%s]失败: %s\n", p.GetKey(), register.Listen.Port, err.Error())
				return err
			}
			logs.Infof("[:%s] 开始监听...\n", register.Listen.Port)
		}
		return nil
	}))
	//this.Clients.Set(v, v)

	defer func() {
		tun.Close()
		v.Close()
		if listener != nil {
			listener.Close()
			logs.Infof("[%s] 关闭监听...\n", listener.Addr().String())
		}
	}()

	return v.Run()
}
