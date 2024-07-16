package main

import (
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/goutil/other/command"
	"github.com/injoyai/goutil/script"
	"github.com/injoyai/goutil/script/js"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"github.com/injoyai/proxy/forward"
	. "github.com/injoyai/proxy/proxy"
	"net"
	"os"
	"strings"
	"time"
)

func init() {
	//设置日志只输出到控制台
	logs.SetWriter(logs.Stdout)

	//设置日志前缀为时间(不包括日期)
	logs.SetFormatterWithTime()

	//设置日志不显示颜色
	logs.SetShowColor(false)
}

var (
	Script = js.NewPool(20)
)

func main() {

	//初始化配置信息,优先获取flag,然后尝试从配置文件获取
	cfg.Init(
		command.WithFlags(
			&command.Flag{Name: "port", Memo: "监听端口", Short: "p"},
			&command.Flag{Name: "address", Memo: "服务地址(代理)", Short: "a"},
			&command.Flag{Name: "proxy", Memo: "代理地址"},
			&command.Flag{Name: "timeout", Memo: "超时时间", Short: "t"},
			&command.Flag{Name: "username", Memo: "用户名", Short: "u"},
			&command.Flag{Name: "password", Memo: "密码"},
			&command.Flag{Name: "log.level", Memo: "日志等级", Short: "l"},
		),
		cfg.WithYaml("./config/config.yaml"),
	)

	args := []string(nil)
	for i := range os.Args {
		if !strings.HasPrefix(os.Args[i], "-") {
			args = append(args, os.Args[i])
		}
	}

	logs.SetLevel(func() logs.Level {
		switch strings.ToLower(cfg.GetString("log.level")) {
		case "all":
			return logs.LevelAll
		case "trace":
			return logs.LevelTrace
		case "debug":
			return logs.LevelDebug
		case "write":
			return logs.LevelWrite
		case "read":
			return logs.LevelRead
		case "info":
			return logs.LevelInfo
		case "warn":
			return logs.LevelWarn
		case "error", "err":
			return logs.LevelError
		case "none":
			return logs.LevelNone
		default:
			return logs.LevelInfo
		}
	}())

	address := cfg.GetString("address", "127.0.0.1:7000")
	port := cfg.GetInt("port", 10088)
	proxy := cfg.GetString("proxy", "127.0.0.1:10001")
	username := cfg.GetString("username")
	password := cfg.GetString("password")
	timeout := cfg.GetDuration("timeout", time.Second*2)
	onRegister := cfg.GetString("onRegister")

	help := `
使用
    forward [flags] 		转发模式(本地代理)		
    proxy client [flags]   	代理客户端			
    proxy server [flags]   	代理服务端			

Flags
    --proxy 	string		代理地址(默认127.0.0.1:10001)	
    --address 	string		服务地址(默认127.0.0.1:7000)		
    --port 		int			监听端口(默认10088)		
    --timeout 	string		超时时间(默认2s)		
    --username 	string		用户名
    --password 	string		密码
`

	if len(args) < 2 || (args[1] != "proxy" && args[1] != "forward") {
		fmt.Printf(help)
		return
	}

	if args[1] == "proxy" && len(args) < 3 {
		fmt.Printf(help)
		return
	}

	switch args[1] {

	case "forward":
		p := forward.Forward{
			Listen: &core.Listen{Port: conv.String(port)},
			Forward: &core.Dial{
				Address: proxy,
			},
		}
		logs.Err(p.ListenTCP())

	case "proxy":

		switch os.Args[2] {
		case "client":
			t := Client{
				Dial: core.NewDialTCP(address, timeout),
				Register: &virtual.RegisterReq{
					Listen:   &core.Listen{Port: conv.String(port)},
					Username: username,
					Password: password,
					Param:    nil,
				},
			}
			ops := []virtual.Option(nil)
			if len(proxy) > 0 {
				ops = append(ops, virtual.WithOpenTCP(proxy, timeout))
			}
			logs.Err(t.DialTCP(ops...))

		case "server":
			t := Server{
				Listen: &core.Listen{Port: conv.String(port)},
				OnProxy: func(c net.Conn) (*core.Dial, []byte, error) {
					return &core.Dial{
						Type:    "tcp",
						Address: proxy,
					}, nil, nil
				},
				OnRegister: func(c net.Conn, r *virtual.RegisterReq) (string, error) {
					_, err := Script.Exec(onRegister, func(i script.Client) {
						i.Set("username", r.Username)
						i.Set("password", r.Password)
						for k, v := range r.Param {
							i.Set(k, v)
						}
					})
					return c.RemoteAddr().String(), err
				},
			}
			logs.Err(t.Run())

		}

	}

}
