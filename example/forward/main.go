package main

import (
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/forward"
)

/*
将本地端口20002的TCP数据转发至局域网192.168.10.187:10001上
*/
func main() {
	f := forward.Forward{
		Listen:  core.NewListenTCP(20002),
		Forward: core.NewDialTCP("192.168.10.187:10001"),
	}
	f.ListenTCP()
}
