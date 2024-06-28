package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/tunnel"
	"net"
	"time"
)

func init() {
	//logs.SetFormatterWithTime()
	logs.SetWriter(logs.Stdout)
}

func main() {

	t := tunnel.Tunnel{
		Port:    7000,
		Timeout: time.Second * 2,
		OnRegister: func(c net.Conn, r *tunnel.RegisterReq) error {
			logs.Debug("注册信息: ", r)
			return nil
		},
	}
	logs.Err(t.ListenTCP())

}
