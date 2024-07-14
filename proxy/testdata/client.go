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
	logs.SetWriter(logs.Stdout)
}

func main() {
	for {
		t := proxy.Client{
			Dial: &core.Dial{
				Address: "127.0.0.1:7000",
				Timeout: time.Second * 2,
			},
			Register: virtual.RegisterReq{
				Port:     20001,
				Username: "username",
				Password: "password",
			},
		}
		logs.Err(t.DialTCP())
		<-time.After(time.Second * 5)
	}
}
