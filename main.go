package main

import (
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/logs"
	"proxy/core"
)

func main() {

	logs.SetWriter(logs.Stdout)
	logs.SetFormatterWithTime()

	cfg.Init(
		cfg.WithFlag(
			&cfg.Flag{Name: "listenPort", Default: "8080", Usage: "监听端口"},
			&cfg.Flag{Name: "proxyAddress", Default: "127.0.0.1:80", Usage: "代理地址"},
		),
		cfg.WithFile("./config/config.yaml"),
	)

	p := core.P2P{
		ListenPort:   cfg.GetInt("listenPort"),
		ProxyAddress: cfg.GetString("proxyAddress"),
	}
	logs.PrintErr(p.ListenTCP())

}
