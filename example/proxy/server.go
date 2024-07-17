package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"github.com/injoyai/proxy/proxy"
	"io"
)

func init() {
	//logs.SetFormatterWithTime()
	logs.SetLevel(logs.LevelInfo)
	logs.SetWriter(logs.Stdout)
}

func main() {

	t := proxy.Server{
		Listen: &core.Listen{Port: "7000"},
		OnProxy: func(r io.ReadWriteCloser) (*core.Dial, []byte, error) {
			return &core.Dial{Address: ":80"}, nil, nil
		},
		OnRegister: func(r io.ReadWriteCloser, v *virtual.Virtual, reg *virtual.RegisterReq) error {
			logs.Debug("注册信息: ", r)
			return nil
		},
	}
	logs.Err(t.Run())

}
