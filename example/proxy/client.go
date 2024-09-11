package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"github.com/injoyai/proxy/proxy"
	"time"
)

func init() {
	core.DefaultLog.SetLevel(core.LevelInfo)
}

func main() {
	for {
		t := proxy.Client{
			Dialer: &core.Dial{
				Address: "127.0.0.1:10007",
				Timeout: time.Second * 2,
			},
			Register: &virtual.RegisterReq{
				Listen: &core.Listen{
					Type: "tcp",
					Port: "20001",
				},
				Username: "username",
				Password: "password",
			},
		}
		logs.Err(t.Run(
			virtual.WithOpenTCP("127.0.0.1:80"),
			virtual.WithKey("ABC"),
		))
		<-time.After(time.Second * 5)
	}
}
