package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/tunnel"
	"time"
)

func init() {
	core.DefaultLog.SetLevel(core.LevelInfo)
}

func main() {
	for {
		t := tunnel.Client{
			Dialer: &core.Dial{
				Address: "127.0.0.1:10007",
				Timeout: time.Second * 2,
			},
			Register: &core.RegisterReq{
				Listen: &core.Listen{
					Type: "tcp",
					Port: "20001",
				},
				Username: "username",
				Password: "password",
			},
		}
		logs.Err(t.Run(
			core.WithDialTCP("127.0.0.1:80"),
			core.WithKey("ABC"),
		))
		<-time.After(time.Second * 5)
	}
}
