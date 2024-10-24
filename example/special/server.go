package main

import (
	"fmt"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/special"
)

func init() {
	core.DefaultLog.SetLevel(core.LevelInfo)
	cfg.Init(
		cfg.WithYaml("./config/config.yaml"),
		cfg.WithFlag(
			&cfg.Flag{Name: "port", Default: 7001, Usage: "服务端口"},
			&cfg.Flag{Name: "username", Default: "username", Usage: "允许注册的用户"},
			&cfg.Flag{Name: "password", Default: "password", Usage: "允许注册的用户密码"},
			&cfg.Flag{Name: "address", Default: ":80", Usage: "内网穿透的地址"},
		),
		cfg.WithEnv(),
	)
}

func main() {

	port := cfg.GetInt("port", 7001)
	username := cfg.GetString("username", "username")
	password := cfg.GetString("password", "password")
	address := cfg.GetString("address", ":80")

	s := special.New(
		special.WithPort(port),
		special.WithAddress(address),
		special.WithRegister(func(tun *core.Tunnel, register *core.RegisterReqExtend) error {
			if register.Username != username {
				return fmt.Errorf("没有权限")
			}
			if register.Password != password {
				return fmt.Errorf("没有权限")
			}
			return nil
		}),
	)
	logs.Err(s.Run())

}
