package main

import (
	"context"
	"errors"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/tunnel"
)

var (
	Tunnel *tunnel.Server
)

func RunTunnel(port int) error {
	core.DefaultLog.SetLevel(core.LevelInfo)
	Tunnel = &tunnel.Server{
		Clients: maps.NewSafe(),
		Listen:  core.NewListenTCP(port),
		OnRegister: func(tun *core.Tunnel, reg *core.RegisterReqExtend) error {
			switch reg.Param["version"] {
			default:
				if reg.Password != "password" {
					return errors.New("账号或者密码错误")
				}
			}
			logs.Debugf("[%s] 新的客户端连接\n", tun.Key())
			return nil
		},
		OnClosed: func(key *core.Tunnel, err error) {
			logs.Debugf("[%s] 客户端断开连接: %v\n", key.Key(), err)
		},
	}
	return Tunnel.Run(context.Background())
}

type Info struct {
	SN      string `json:"sn"`
	Address string `json:"address"`
}
