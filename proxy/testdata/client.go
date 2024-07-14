package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core/virtual"
	"github.com/injoyai/proxy/proxy"
	"io"
	"time"
)

func init() {
	//logs.SetFormatterWithTime()
	logs.SetWriter(logs.Stdout)
}

func main() {
	for {
		t := proxy.Client{
			Dial: virtual.Dial{
				Address: "127.0.0.1:7000",
				Timeout: time.Second * 2,
			},
			OnOpen: func(p virtual.Packet) (io.ReadWriteCloser, string, error) {
				proxy := virtual.Dial{
					Address: "192.168.10.24:10001",
					Timeout: time.Second * 2,
				}
				return proxy.Dial()
			},
			Register: virtual.RegisterReq{
				Port:     20001,
				Username: "username",
				Password: "password",
			},
		}
		logs.Err(t.RunTCP())
		<-time.After(time.Second * 5)
	}
}
