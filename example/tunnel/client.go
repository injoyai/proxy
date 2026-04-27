package main

import (
	"time"

	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/tunnel"
)

func init() {
	logs.SetLevel(logs.LevelInfo)
}

func main() {
	key := "ABC"
	for {
		t := tunnel.Client{
			Dialer: &core.Dial{
				Address: "127.0.0.1:7000",
				Timeout: time.Second * 2,
			},
			Register: &core.RegisterReq{
				Listen: &core.Listen{
					Type:    "tcp",
					Address: ":20001",
				},
				Key:      key,
				Username: "username",
				Password: "password",
			},
		}
		logs.Err(t.Run(
			core.WithDialTCP("baidu.com:80"),
			core.WithKey(key),
		))
		<-time.After(time.Second * 5)
	}
}
