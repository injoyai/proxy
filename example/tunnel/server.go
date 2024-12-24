package main

import (
	"context"
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
		OnRegister: func(tun *core.Tunnel, reg *core.RegisterReqExtend) error {
			reg.OnProxy = func(r io.ReadWriteCloser) (*core.Dial, []byte, error) {
				return &core.Dial{Address: ":80"}, nil, nil
			}
			tun.SetKey(reg.GetString("key"))
			logs.Debug("注册信息: ", reg)
			return nil
		},
	}
	logs.Err(t.Run(context.Background()))

}
