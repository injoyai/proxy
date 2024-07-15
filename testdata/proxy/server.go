package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"github.com/injoyai/proxy/proxy"
	"net"
)

func init() {
	//logs.SetFormatterWithTime()
	//logs.SetLevel(logs.LevelInfo)
	logs.SetWriter(logs.Stdout)
}

func main() {

	t := proxy.Server{
		Listen: &core.Listen{Port: "7000"},
		OnProxy: func(c net.Conn) (*core.Dial, []byte, error) {
			return &core.Dial{Address: ":80"}, nil, nil
		},
		OnRegister: func(c net.Conn, r *virtual.RegisterReq) error {
			logs.Debug("注册信息: ", r)
			return nil
		},
	}
	logs.Err(t.Run())

}
