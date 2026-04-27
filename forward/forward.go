package forward

import (
	"context"
	"net"

	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
)

type Forward struct {
	Listen  *core.Listen //监听配置
	Forward *core.Dial   //转发配置
}

func (this *Forward) Run(ctx ...context.Context) error {
	this.Listen.OnConnected(this.Handler)
	return this.Listen.ListenAndRun(ctx...)
}

func (this *Forward) Handler(l net.Listener, c net.Conn) {
	logs.Infof("[%s] 转发至 [%s]\n", c.RemoteAddr().String(), this.Forward.Address)
	defer c.Close()

	newConn, _, err := this.Forward.Dial()
	if err != nil {
		logs.Err(err)
		return
	}
	defer newConn.Close()

	err = core.Bridge(c, newConn)
	_ = err
}
