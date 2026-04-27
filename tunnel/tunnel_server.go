package tunnel

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"

	"github.com/injoyai/base/maps"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
)

type Server struct {
	Clients     *maps.Safe                                                //客户端
	Listen      *core.Listen                                              //监听配置
	OnRegister  func(tun *core.Tunnel, reg *core.RegisterReqExtend) error //注册事件
	OnConnected func(conn io.ReadWriteCloser, tun *core.Tunnel)           //连接事件
	OnClosed    func(key *core.Tunnel, err error)                         //关闭事件
}

func (this *Server) GetTunnel(key string) *core.Tunnel {
	x, ok := this.Clients.Get(key)
	if ok {
		return x.(*core.Tunnel)
	}
	return nil
}

func (this *Server) Run(ctx ...context.Context) error {
	if this.Clients == nil {
		this.Clients = maps.NewSafe()
	}
	this.Listen.OnConnected(this.Handler)
	return this.Listen.ListenAndRun(ctx...)
}

// Handler 对客户端进行注册验证操作
func (this *Server) Handler(_ net.Listener, tunConn net.Conn) {

	var listener *core.Listen

	tun := core.NewTunnel(tunConn, core.WithKey(tunConn.RemoteAddr().String()))
	tun.SetOption(core.WithRegister(func(tun *core.Tunnel, data []byte) (interface{}, error) {
		//解析注册数据
		register := new(core.RegisterReq)
		err := json.Unmarshal(data, register)
		if err != nil {
			return nil, err
		}
		registerExtend := register.Extend()

		//注册事件
		if this.OnRegister != nil {
			if err := this.OnRegister(tun, registerExtend); err != nil {
				return nil, err
			}
		}
		//如果存在老的连接的话,会被覆盖,变成野连接,能收到数据,不能发数据,还是说关闭老连接?
		this.Clients.Set(tun.Key(), tun)

		//判断客户端是否需要监听端口
		//客户端可以选择不监听端口,而由服务端进行安排
		if register.Listen == nil || register.Listen.Address == "" {
			return register.Listen, nil
		}

		//监听端口
		err = register.Listen.Listen()
		if err != nil {
			logs.Errf("[%s] 监听[:%s]失败: %s\n", tun.Key(), register.Listen.Address, err.Error())
			return nil, err
		}
		listener = register.Listen

		//监听端口
		register.Listen.OnConnected(func(listener net.Listener, c net.Conn) {

			cKey := c.RemoteAddr().String()

			var err error
			defer func() {
				logs.Tracef("[%s] 关闭连接: %v\n", cKey, err)
			}()
			defer c.Close()

			proxy := &core.Dial{}
			prefix := []byte(nil)

			//使用自定义(服务端)代理
			if registerExtend.OnProxy != nil {
				proxy, prefix, err = registerExtend.OnProxy(c)
				if err != nil {
					return
				}
			}
			if proxy == nil {
				proxy = &core.Dial{}
			}

			//新建个虚拟IO
			var virtual io.ReadWriteCloser
			virtual, err = tun.Dial(cKey, proxy, c)
			if err != nil {
				return
			}
			defer virtual.Close()

			logs.Infof("监听[%s] -> 隧道[%s] -> 请求[%s]\n", register.Listen.Address, tun.Key(), proxy.Address)

			err = core.Bridge(virtual, struct {
				io.Reader
				io.WriteCloser
			}{
				Reader:      io.MultiReader(bytes.NewReader(prefix), c),
				WriteCloser: c,
			})

		})

		go register.Listen.Run()
		logs.Infof("[%s] 监听[%s]成功...\n", tun.Key(), register.Listen.Address)

		return register.Listen, nil
	}))

	if this.OnConnected != nil {
		this.OnConnected(tunConn, tun)
	}

	err := tun.Run()
	logs.Err(err)

	{
		this.Clients.Del(tun.Key())
		tunConn.Close()
		tun.Close()
		if this.OnClosed != nil {
			this.OnClosed(tun, err)
		}
		if listener != nil {
			listener.Close()
			logs.Infof("[%s] 关闭监听...\n", listener.Key())
		}
	}

}
