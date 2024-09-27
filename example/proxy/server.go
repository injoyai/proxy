package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/tunnel"
	tun "github.com/injoyai/proxy/tunnel"
	"io"
)

func init() {
	core.DefaultLog.SetLevel(core.LevelInfo)
}

func main() {

	t := tun.Server{
		Listen: &core.Listen{Port: "7000"},
		OnProxy: func(r io.ReadWriteCloser) (*core.Dial, []byte, error) {
			return &core.Dial{Address: ":80"}, nil, nil
		},
		OnRegister: func(r io.ReadWriteCloser, v *tunnel.Tunnel, reg *tunnel.RegisterReq) error {
			logs.Debug("注册信息: ", reg)
			return nil
		},
	}
	logs.Err(t.Run())

}
