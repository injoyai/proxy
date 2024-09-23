package main

import (
	"errors"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"io"
)

var (
	Tunnel *tunnel.Server
)

func RunTunnel(port int) error {
	core.DefaultLog.SetLevel(core.LevelInfo)
	Tunnel = &tunnel.Server{
		Clients: maps.NewSafe(),
		Listen:  core.NewListenTCP(port),
		OnRegister: func(r io.ReadWriteCloser, key *virtual.Virtual, reg *virtual.RegisterReq) error {
			switch reg.Param["version"] {
			default:
				if reg.Password != "password" {
					return errors.New("账号或者密码错误")
				}
			}
			logs.Debugf("[%s] 新的客户端连接\n", key.Key())
			return nil
		},
		OnClosed: func(key *virtual.Virtual, err error) {
			logs.Debugf("[%s] 客户端断开连接: %v\n", key.Key(), err)
		},
	}
	return Tunnel.Run()
}

type Info struct {
	SN      string `json:"sn"`
	Address string `json:"address"`
}
