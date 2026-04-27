package main

import (
	"context"
	"fmt"
	"time"

	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/forward"
)

/*
将本地端口{port}的TCP数据转发至局域网{ip}:{port}上
*/
func main() {
	f := forward.Forward{
		Listen:  core.NewListenTCP("127.0.0.1:20002"),
		Forward: core.NewDialTCP("baidu.com:80"),
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Second * 10)
		cancel()
	}()
	fmt.Println(f.Run(ctx))
}
