package main

import (
	"context"
	"fmt"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/forward"
	"time"
)

/*
将本地端口20002的TCP数据转发至局域网192.168.10.187:10001上
*/
func main() {
	f := forward.Forward{
		Listen:  core.NewListenTCP(20002),
		Forward: core.NewDialTCP("192.168.10.71:10001"),
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Second * 10)
		cancel()
	}()
	fmt.Println(f.Run(ctx))
}
