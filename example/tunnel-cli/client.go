package main

import (
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/tunnel"
	"time"
)

func init() {
	core.DefaultLog.SetLevel(core.LevelInfo)
	cfg.Init(
		cfg.WithYaml("./config/config.yaml"),
		cfg.WithFlag(
			&cfg.Flag{Name: "address", Default: ":7001", Usage: "服务地址"},
			&cfg.Flag{Name: "timeout", Default: "2s", Usage: "超时时间"},
			&cfg.Flag{Name: "username", Default: "username", Usage: "注册的用户"},
			&cfg.Flag{Name: "password", Default: "password", Usage: "注册的用户密码"},
			&cfg.Flag{Name: "forward", Default: "", Usage: "本地转发地址,可选"},
			&cfg.Flag{Name: "key", Default: "", Usage: "服务端显示的标识,可选"},
		),
		cfg.WithEnv(),
	)
}

func main() {
	address := cfg.GetString("address")
	timeout := cfg.GetDuration("timeout", time.Second*2)
	username := cfg.GetString("username", "username")
	password := cfg.GetString("password", "password")
	forward := cfg.GetString("forward")
	key := cfg.GetString("key")

	for {
		t := tunnel.Client{
			Dialer: &core.Dial{
				Address: address,
				Timeout: timeout,
			},
			Register: &core.RegisterReq{
				Key:      key,
				Username: username,
				Password: password,
			},
		}
		if len(forward) > 0 {
			logs.Err(t.Run(core.WithDialTCP(forward)))
		} else {
			logs.Err(t.Run())
		}
		<-time.After(time.Second * 5)
	}
}
