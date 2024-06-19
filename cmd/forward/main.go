package main

import (
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/logs"
	"proxy/core"
)

func main() {

	//初始化配置信息,优先获取flag,然后尝试从配置文件获取
	cfg.Init(
		cfg.WithFlag(
			&cfg.Flag{Name: "listenPort", Default: "8080", Usage: "监听端口"},
			&cfg.Flag{Name: "proxyAddress", Default: "127.0.0.1:80", Usage: "代理地址"},
		),
		cfg.WithFile("./config/config.yaml"),
	)

	//设置日志只输出到控制台
	logs.SetWriter(logs.Stdout)

	//设置日志前缀为时间(不包括日期)
	logs.SetFormatterWithTime()

	//转发配置
	p := core.Forward{
		Port:    cfg.GetInt("listenPort"),
		Address: cfg.GetString("proxyAddress"),
	}
	logs.PrintErr(p.ListenTCP())

}
