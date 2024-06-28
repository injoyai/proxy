package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/tunnel"
	"time"
)

func init() {
	//logs.SetFormatterWithTime()
	logs.SetWriter(logs.Stdout)
}

func main() {
	for {
		t := tunnel.Client{
			Address:      "127.0.0.1:7000",
			Port:         20001,
			ProxyAddress: "127.0.0.1:10086", //"192.168.10.24:10001",
			Username:     "username",
			Password:     "password",
			Timeout:      time.Second * 2,
		}
		logs.Err(t.Dial())
		<-time.After(time.Second * 5)
	}
}
