package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core/virtual"
	"github.com/injoyai/proxy/proxy"
	"net"
	"time"
)

func init() {
	//logs.SetFormatterWithTime()
	logs.SetWriter(logs.Stdout)
}

func main() {

	t := proxy.Server{
		Port:    7000,
		Timeout: time.Second * 2,
		Proxy:   "192.168.10.24:10001",
		OnRegister: func(c net.Conn, r *virtual.RegisterReq) error {
			logs.Debug("注册信息: ", r)
			return nil
		},
	}
	logs.Err(t.ListenTCP())

}
