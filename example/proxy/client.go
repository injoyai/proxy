package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/tunnel"
	tun "github.com/injoyai/proxy/tunnel"
	"time"
)

func init() {
	core.DefaultLog.SetLevel(core.LevelInfo)
}

func main() {
	for {
		t := tun.Client{
			Dialer: &core.Dial{
				Address: "127.0.0.1:10007",
				Timeout: time.Second * 2,
			},
			Register: &tunnel.RegisterReq{
				Listen: &core.Listen{
					Type: "tcp",
					Port: "20001",
				},
				Username: "username",
				Password: "password",
			},
		}
		logs.Err(t.Run(
			tunnel.WithDialTCP("127.0.0.1:80"),
			tunnel.WithKey("ABC"),
		))
		<-time.After(time.Second * 5)
	}
}
