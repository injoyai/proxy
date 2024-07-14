package forward

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"net"
)

type Forward struct {
	Listen  *core.Listen //监听配置
	Forward *core.Dial   //转发配置
}

func (this *Forward) ListenTCP() error {
	logs.Infof("[:%s] 开始监听...\n", this.Listen.Port)
	defer logs.Infof("[:%s] 关闭监听...\n", this.Listen.Port)
	return this.Listen.Listen(nil, this.Handler)
}

func (this *Forward) Handler(l net.Listener, c net.Conn) error {
	logs.Infof("[%s] 转发至 [%s]\n", c.RemoteAddr().String(), this.Forward.Address)
	defer c.Close()

	newConn, _, err := this.Forward.Dial()
	if err != nil {
		return err
	}
	defer newConn.Close()

	return core.Swap(c, newConn)
}
