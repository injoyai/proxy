package proxy

import (
	"bytes"
	"encoding/json"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"io"
	"net"
	"time"
)

type Server struct {
	Port       int                                                    //客户端连接的端口
	Timeout    time.Duration                                          //超时时间
	Proxy      string                                                 //代理地址
	OnRegister func(c net.Conn, r *virtual.RegisterReq) error         //注册事件
	OnProxy    func(c net.Conn, proxy string) ([]byte, string, error) //代理事件
}

func (this *Server) ListenTCP() error {
	return core.RunListen("tcp", this.Port, core.WithListenLog, this.Handler)
}

// Handler 对客户端进行注册验证操作
func (this *Server) Handler(tunListen net.Listener, c net.Conn) error {

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
				defer logs.Tracef("[%s] 关闭连接: %v\n", c.RemoteAddr().String(), err)

				//使用自定义代理
				if this.OnProxy != nil {
					prefix, proxy, err := this.OnProxy(c, this.Proxy)
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

				//使用默认代理
				return v.OpenAndSwap(this.Proxy, c)
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
