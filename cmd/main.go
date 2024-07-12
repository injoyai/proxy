package main

import (
	"fmt"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/goutil/script"
	"github.com/injoyai/goutil/script/js"
	"github.com/injoyai/logs"
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
}

var (
	Script = js.NewPool(20)
)

func main() {

	//初始化配置信息,优先获取flag,然后尝试从配置文件获取
	cfg.Init(
		cfg.WithFlag(
			&cfg.Flag{Name: "port", Usage: "监听端口"},
			&cfg.Flag{Name: "address", Usage: "服务地址(代理)"},
			&cfg.Flag{Name: "proxy", Usage: "代理地址"},
			&cfg.Flag{Name: "timeout", Usage: "超时时间"},
			&cfg.Flag{Name: "username", Usage: "用户名"},
			&cfg.Flag{Name: "password", Usage: "密码"},
			&cfg.Flag{Name: "log.level", Usage: "日志等级"},
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
			return logs.LevelAll
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
    proxy client [flags]  	代理客户端			
    proxy server [flags]  	代理服务端			

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
			Port:    port,
			Address: proxy,
		}
		logs.Err(p.ListenTCP())

	case "proxy":

		switch os.Args[2] {
		case "client":
			t := Client{
				Address:  address,
				Proxy:    proxy,
				Port:     port,
				Username: username,
				Password: password,
				Timeout:  timeout,
			}
			logs.Err(t.Dial())

		case "server":
			t := Server{
				Port:    port,
				Timeout: timeout,
				Proxy:   proxy,
				OnRegister: func(c net.Conn, r *virtual.RegisterReq) error {
					_, err := Script.Exec(onRegister, func(i script.Client) {
						i.Set("username", r.Username)
						i.Set("password", r.Password)
						for k, v := range r.Param {
							i.Set(k, v)
						}
					})
					return err
				},
			}
			logs.Err(t.ListenTCP())

		}

	}

}
