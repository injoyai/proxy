package main

import (
	"io"

	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/tunnel"
)

func init() {
	//logs.SetLevel(logs.LevelInfo)
}

func main() {

	t := tunnel.Server{
		Listen: core.NewListenTCP(7000),
		OnRegister: func(tun *core.Tunnel, reg *core.RegisterReqExtend) error {
			reg.OnProxy = func(r io.ReadWriteCloser) (*core.Dial, []byte, error) {
				return &core.Dial{Address: ":80"}, nil, nil
			}
			tun.SetKey(reg.GetString("key"))
			logs.Debugf("注册信息: %v\n", *reg)
			return nil
		},
	}
	logs.Err(t.Run())

}
