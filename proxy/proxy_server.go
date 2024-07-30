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
	Clients    *maps.Safe                                                                       //客户端
	Listen     *core.Listen                                                                     //监听配置
	OnRegister func(r io.ReadWriteCloser, key *virtual.Virtual, reg *virtual.RegisterReq) error //注册事件
	OnProxy    func(r io.ReadWriteCloser) (*core.Dial, []byte, error)                           //代理事件
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

	v := virtual.New(tun, virtual.WithKey(tun.RemoteAddr().String()))
	v.SetOption(virtual.WithRegister(func(v *virtual.Virtual, p virtual.Packet) (interface{}, error) {
		//解析注册数据
		register := new(virtual.RegisterReq)
		err := json.Unmarshal(p.GetData(), register)
		if err != nil {
			return nil, err
		}

		v.SetKey(p.GetKey())
		//注册事件
		if this.OnRegister != nil {
			if err := this.OnRegister(tun, v, register); err != nil {
				return nil, err
			}
		}
		//如果存在老的连接的话,会被覆盖,变成野连接,能收到数据,不能发数据,还是说关闭老连接?
		this.Clients.Set(v.Key(), v)

		//判断客户端是否需要监听端口
		//客户端可以选择不监听端口,而由服务端进行安排
		if register.Listen == nil || register.Listen.Port == "" {
			return register.Listen, nil
		}

		{ //监听端口
			listener, err = register.Listen.GoListen(func(listener net.Listener, c net.Conn) error {
				cKey := c.RemoteAddr().String()
				defer logs.Tracef("[%s] 关闭连接: %v\n", cKey, err)

				//使用自定义(服务端)代理
				if this.OnProxy != nil {
					proxy, prefix, err := this.OnProxy(c)
					if err != nil {
						return err
					}
					if proxy != nil {
						logs.Infof("[%s -> :%s] 代理至 [%s -> %s]\n", cKey, register.Listen.Port, v.Key(), proxy.Address)
						return v.OpenAndSwap(c.RemoteAddr().String(), proxy, struct {
							io.Reader
							io.WriteCloser
						}{
							Reader:      io.MultiReader(bytes.NewReader(prefix), c),
							WriteCloser: c,
						})
					}
				}

				logs.Infof("[%s] 代理至 [%s]\n", cKey, v.Key())
				//使用默认(客户端)代理
				return v.OpenAndSwap(c.RemoteAddr().String(), &core.Dial{}, c)
			})
			if err != nil {
				logs.Errf("[%s] 监听端口[:%s]失败: %s\n", p.GetKey(), register.Listen.Port, err.Error())
				return nil, err
			}
			logs.Infof("[%s] 请求监听[:%s]成功...\n", v.Key(), register.Listen.Port)
		}
		return register.Listen, nil
	}))

	defer func() {
		this.Clients.Del(v.Key())
		tun.Close()
		v.Close()
		if listener != nil {
			listener.Close()
			logs.Infof("[%s] 关闭监听...\n", listener.Addr().String())
		}
	}()

	return v.Run()
}
