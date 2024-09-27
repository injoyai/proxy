package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/tunnel"
	"io"
)

func init() {
	core.DefaultLog.SetLevel(core.LevelInfo)
}

func main() {

	t := tunnel.Server{
		Listen: &core.Listen{Port: "7000"},
		OnProxy: func(r io.ReadWriteCloser) (*core.Dial, []byte, error) {
			return &core.Dial{Address: ":80"}, nil, nil
		},
		OnRegister: func(r io.ReadWriteCloser, v *core.Tunnel, reg *core.RegisterReq) error {
			logs.Debug("注册信息: ", reg)
			return nil
		},
	}
	logs.Err(t.Run())

}
