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
			Address:  "127.0.0.1:7000",
			Port:     20001,
			Username: "username",
			Password: "password",
		}
		logs.Err(t.Dial())
		<-time.After(time.Second * 5)
	}
}
