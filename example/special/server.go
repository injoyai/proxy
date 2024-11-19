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
			&cfg.Flag{Name: "username", Usage: "允许注册的用户"},
			&cfg.Flag{Name: "password", Usage: "允许注册的用户密码"},
			&cfg.Flag{Name: "address", Default: ":80", Usage: "内网穿透的地址"},
		),
	)
}

func main() {

	port := cfg.GetInt("port", 7001)
	username := cfg.GetString("username")
	password := cfg.GetString("password")
	address := cfg.GetString("address", ":80")

	logs.Debug("port:", port)
	logs.Debug("username:", username)
	logs.Debug("password:", password)
	logs.Debug("address:", address)

	s := special.New(
		special.WithPort(port),       //服务监听端口
		special.WithAddress(address), //内网穿透地址
		special.WithRegister(func(tun *core.Tunnel, register *core.RegisterReqExtend) error {
			if len(username) > 0 && register.Username != username {
				return fmt.Errorf("账号或密码错误")
			}
			if len(password) > 0 && register.Password != password {
				return fmt.Errorf("账号或密码错误")
			}
			logs.Debugf("[%s] 注册成功...\n", tun.Key())
			return nil
		}),
	)
	logs.Err(s.Run())

}
