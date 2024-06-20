package main

import (
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/forward"
)

func main() {

	//初始化配置信息,优先获取flag,然后尝试从配置文件获取
	cfg.Init(
		cfg.WithFlag(
			&cfg.Flag{Name: "port", Usage: "监听端口"},
			&cfg.Flag{Name: "address", Usage: "代理地址"},
		),
		cfg.WithYaml("./cmd/forward/config/config.yaml"),
	)

	//设置日志只输出到控制台
	logs.SetWriter(logs.Stdout)

	//设置日志前缀为时间(不包括日期)
	logs.SetFormatterWithTime()

	//转发配置
	p := forward.Forward{
		Port:    cfg.GetInt("port"),
		Address: cfg.GetString("address"),
	}
	logs.PrintErr(p.ListenTCP())

}
