package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/tunnel"
	"net"
)

func init() {
	//logs.SetFormatterWithTime()
	logs.SetWriter(logs.Stdout)
}

func main() {

	t := tunnel.Tunnel{
		Port: 7000,
		OnRegister: func(c net.Conn, r *tunnel.RegisterReq) error {
			logs.Debug("注册信息: ", r)
			return nil
		},
	}
	logs.Err(t.ListenTCP())

}
