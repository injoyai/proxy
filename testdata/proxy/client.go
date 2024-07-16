package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"github.com/injoyai/proxy/proxy"
	"time"
)

func init() {
	//logs.SetFormatterWithTime()
	//logs.SetLevel(logs.LevelInfo)
	logs.SetWriter(logs.Stdout)
}

func main() {
	for {
		t := proxy.Client{
			Dial: &core.Dial{
				Address: "127.0.0.1:7000",
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
		logs.Err(t.DialTCP(virtual.WithOpenTCP("192.168.10.24:10001")))
		<-time.After(time.Second * 5)
	}
}
