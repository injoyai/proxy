package main

import (
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/goutil/other/command"
	"github.com/injoyai/goutil/script"
	"github.com/injoyai/goutil/script/js"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/tunnel"
	"github.com/injoyai/proxy/forward"
	. "github.com/injoyai/proxy/tunnel"
	"github.com/spf13/cobra"
	"io"
	"strings"
	"time"
)

func init() {
	//设置日志前缀为时间(不包括日期)
	core.DefaultLog.(interface{ SetFormatterWithTime() }).SetFormatterWithTime()

	//设置日志不显示颜色
	core.DefaultLog.(interface{ SetShowColor(b ...bool) }).SetShowColor(false)
}

var (
	Script = js.NewPool(20)
)

func main() {

	//初始化配置信息,优先获取flag,然后尝试从配置文件获取
	cfg.Init(cfg.WithYaml("./config/config.yaml"))

	root := command.Command{
		Flag: []*command.Flag{{Name: "log.level", Memo: "日志等级,info,debug等"}},
		Child: []*command.Command{
			{
				Command: cobra.Command{
					Use:     "forward",
					Short:   "转发模式",
					Example: "proxy forward 8080->:80",
				},
				Flag: []*command.Flag{
					{Name: "port", Memo: "监听端口", Short: "p", Default: cfg.GetString("port", "80")},
					{Name: "proxy", Memo: "转发地址", Default: cfg.GetString("proxy")},
				},
				Run: func(cmd *cobra.Command, args []string, flag *command.Flags) {
					SetLevel(flag)

					port := flag.GetString("port")
					proxy := flag.GetString("proxy")
					timeout := flag.GetDuration("timeout", 5*time.Second)

					if len(args) > 0 {
						if ls := strings.SplitN(args[0], "->", 2); len(ls) == 2 {
							port = ls[0]
							proxy = ls[1]
						} else if ls := strings.SplitN(args[0], "<-", 2); len(ls) == 2 {
							port = ls[1]
							proxy = ls[0]
						}
					}

					p := forward.Forward{
						Listen:  &core.Listen{Port: port},
						Forward: core.NewDialTCP(proxy, timeout),
					}

					core.DefaultLog.Errf("listen err: %v", p.ListenTCP())
				},
			},
			{
				Command: cobra.Command{
					Use:     "client",
					Short:   "代理客户端",
					Example: "proxy client xxx.xxx.xxx.xxx:7000 :10001<-20001",
				},
				Flag: []*command.Flag{
					{Name: "port", Memo: "想让服务端监听的端口", Short: "p", Default: cfg.GetString("port", "80")},
					{Name: "proxy", Memo: "客户端代理地址", Default: cfg.GetString("proxy")},
					{Name: "timeout", Memo: "超时时间", Short: "t", Default: cfg.GetString("timeout")},
					{Name: "username", Memo: "用户名", Short: "u", Default: cfg.GetString("username")},
					{Name: "password", Memo: "密码", Default: cfg.GetString("password")},
					{Name: "key", Memo: "唯一标识", Default: cfg.GetString("key")},
				},
				Run: func(cmd *cobra.Command, args []string, flag *command.Flags) {
					SetLevel(flag)

					if len(args) == 0 {
						fmt.Println("未填写服务地址")
						return
					}

					port := flag.GetInt("port")
					proxy := flag.GetString("proxy")
					timeout := flag.GetDuration("timeout", 5*time.Second)
					username := flag.GetString("username")
					password := flag.GetString("password")
					key := flag.GetString("key")

					if len(args) > 1 {
						if ls := strings.SplitN(args[1], "<-", 2); len(ls) == 2 {
							port = conv.Int(ls[1])
							proxy = ls[0]
						} else if ls := strings.SplitN(args[1], "->", 2); len(ls) == 2 {
							port = conv.Int(ls[0])
							proxy = ls[1]
						}
					}

					t := Client{
						Dialer: core.NewDialTCP(args[0], timeout),
						Register: &tunnel.RegisterReq{
							Listen:   core.NewListenTCP(port),
							Username: username,
							Password: password,
						},
					}
					ops := []tunnel.Option{func(v *tunnel.Tunnel) {
						if len(key) > 0 {
							v.SetKey(key)
						}
					}}
					if len(proxy) > 0 {
						ops = append(ops, tunnel.WithDialTCP(proxy, timeout))
					}
					for {
						core.DefaultLog.Errf("dial err: %v", t.Run(ops...))
						<-time.After(time.Second * 5)
					}
				},
			},
			{
				Command: cobra.Command{
					Use:     "server",
					Short:   "代理服务端",
					Example: "proxy server -p=7000 :10001<-20001",
				},
				Flag: []*command.Flag{
					{Name: "port", Memo: "监听端口", Short: "p", Default: cfg.GetString("port", "7000")},
					{Name: "proxy", Memo: "代理地址", Default: cfg.GetString("proxy")},
					{Name: "timeout", Memo: "超时时间", Short: "t", Default: cfg.GetString("timeout")},
					{Name: "listen", Memo: "监听端口", Default: cfg.GetString("listen")},
				},
				Run: func(cmd *cobra.Command, args []string, flag *command.Flags) {
					SetLevel(flag)

					port := flag.GetInt("port")
					proxy := flag.GetString("proxy")
					listen := flag.GetInt("listen")
					onRegister := flag.GetString("onRegister")

					if len(args) > 0 {
						if ls := strings.SplitN(args[0], "->", 2); len(ls) == 2 {
							listen = conv.Int(ls[0])
							proxy = ls[1]
						} else if ls := strings.SplitN(args[0], "<-", 2); len(ls) == 2 {
							listen = conv.Int(ls[1])
							proxy = ls[0]
						}
					}

					t := Server{
						Listen: core.NewListenTCP(port),
						OnProxy: func(r io.ReadWriteCloser) (*core.Dial, []byte, error) {
							if len(proxy) == 0 {
								return nil, nil, nil
							}
							return core.NewDialTCP(proxy), nil, nil
						},
						OnRegister: func(r io.ReadWriteCloser, v *tunnel.Tunnel, reg *tunnel.RegisterReq) error {
							if listen > 0 {
								reg.Listen = core.NewListenTCP(listen)
							}
							if len(onRegister) == 0 {
								return nil
							}
							result, err := Script.Exec(onRegister, func(i script.Client) {
								i.Set("key", v.Key())
								i.Set("username", reg.Username)
								i.Set("password", reg.Password)
								for k, v := range reg.Param {
									i.Set(k, v)
								}
							})
							if k := conv.String(result); len(k) > 0 {
								v.SetKey(k)
							}
							return err
						},
					}
					core.DefaultLog.Errf("listen err: %v", t.Run())
				},
			},
		},
	}

	root.Execute()

}

func SetLevel(flag *command.Flags) {
	core.DefaultLog.(interface {
		SetLevelStr(level string)
	}).SetLevelStr(flag.GetString("log.level"))
}
