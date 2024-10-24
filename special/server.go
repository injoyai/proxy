package special

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"io"
	"net"
)

type Option func(s *Server)

func WithAddress(address string) Option {
	return func(s *Server) {
		s.Address = address
	}
}

func WithPort(port int) Option {
	return func(s *Server) {
		s.Port = port
	}
}

func WithRegister(onRegister func(tun *core.Tunnel, register *core.RegisterReqExtend) error) Option {
	return func(s *Server) {
		s.OnRegister = onRegister
	}
}

func WithTunnel(op ...core.OptionTunnel) Option {
	return func(s *Server) {
		s.TunnelOption = append(s.TunnelOption, op...)
	}
}

func New(op ...Option) *Server {
	s := &Server{
		Port:       7001,
		OnRegister: nil,
	}
	for _, v := range op {
		v(s)
	}
	return s
}

/*
Server
只有1个隧道连接
隧道和代理共用一个端口
根据首次连接的前几个字节来进行判断是否是隧道连接
*/
type Server struct {
	tunnel   *core.Tunnel
	listener net.Listener

	Port         int                                                            //服务监听的端口
	Address      string                                                         //客户端转发的地址
	OnRegister   func(tun *core.Tunnel, register *core.RegisterReqExtend) error //注册事件
	TunnelOption []core.OptionTunnel                                            //隧道选项
}

func (this *Server) Run() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", this.Port))
	if err != nil {
		return err
	}
	core.DefaultLog.Infof("监听端口[:%d]成功...\n", this.Port)

	this.listener = listener
	for {
		c, err := this.listener.Accept()
		if err != nil {
			return err
		}
		go func(c net.Conn) {
			if err := this.handler(c); err != nil && err != io.EOF {
				logs.Err(err)
			}
		}(c)
	}
}

func (this *Server) handler(c net.Conn) error {
	defer c.Close()

	//读取前2字节是否是默认协议的帧头
	prefix := make([]byte, 2)
	n, err := io.ReadAtLeast(c, prefix, 2)
	if err != nil {
		return err
	}

	conn := struct {
		io.Reader
		io.WriteCloser
	}{
		io.MultiReader(bytes.NewReader(prefix), c),
		c,
	}

	//说明是隧道连接
	if n == 2 && prefix[0] == 0x89 && prefix[1] == 0x89 {
		//关闭老的链接
		if this.tunnel != nil && !this.tunnel.Closed() {
			this.tunnel.Close()
		}
		//建立新的连接实例
		this.tunnel = core.NewTunnel(
			conn,
			core.WithKey(c.RemoteAddr().String()),
			core.WithRegister(func(tun *core.Tunnel, p core.Packet) (interface{}, error) {
				register := new(core.RegisterReq)
				err := json.Unmarshal(p.GetData(), register)
				if err != nil {
					return nil, err
				}
				//注册事件
				if this.OnRegister != nil {
					if err := this.OnRegister(tun, register.Extend()); err != nil {
						return nil, err
					}
				}
				return nil, nil
			}),
		)
		this.tunnel.SetOption(this.TunnelOption...)
		return this.tunnel.Run()
	}

	if this.tunnel == nil || this.tunnel.Closed() {
		return nil
	}

	core.DefaultLog.Infof("监听[:%d] -> 隧道[%s] -> 请求[%s]\n", this.Port, this.tunnel.Key(), this.Address)

	//普通代理连接
	return this.tunnel.DialAndSwap(c.RemoteAddr().String(), core.NewDialTCP(this.Address), conn)

}
