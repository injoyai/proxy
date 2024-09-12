package main

import (
	"errors"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"github.com/injoyai/proxy/proxy"
	"io"
)

var (
	Tunnel *proxy.Server
)

func RunTunnel(port int) error {
	//core.DefaultLog.SetLevel(core.LevelInfo)
	Tunnel = &proxy.Server{
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
	}
	return Tunnel.Run()
}

type Info struct {
	SN      string `json:"sn"`
	Address string `json:"address"`
}
