package tunnel

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"net"
	"sync"
	"time"
)

type Client struct {
	Address  string
	Port     int
	Username string //用户名
	Password string //密码
	m        sync.Map
}

func (this *Client) Dial() error {

	tun, err := net.Dial("tcp", this.Address)
	if err != nil {
		return err
	}
	defer tun.Close()

	//进行注册认证
	if err := this.Register(tun); err != nil {
		return err
	}

	r := bufio.NewReader(tun)
	for {

		bs, err := core.Read(r)
		if err != nil {
			logs.Error(err)
			return err
		}
		logs.Debug(string(bs))

		p, err := Decode(bs)
		if err != nil {
			continue
		}

		switch p.Type {
		case Open:
			c, err := net.Dial("tcp", this.Address)
			if err != nil {
				//关闭
				core.Write(tun, &Packet{
					Type: Close,
					Body: conv.Bytes(fmt.Sprintf("%s", err)),
				})
				continue
			}

			v := &virtual{
				c:   c,
				buf: bytes.NewBuffer(nil),
			}

			go func() {
				this.m.Store(p.Address, v)
				defer this.m.Delete(p.Address)
				defer v.Close()
				core.Swap(tun, v)
			}()

		case Write:

			if v, _ := this.m.Load(p.Address); v != nil {
				v.(*virtual).Write(p.Body)
			}

		case Close:

			if v, _ := this.m.Load(p.Address); v != nil {
				v.(*virtual).Close()
			}

		}

	}

}

func (this *Client) Register(c net.Conn) error {

	_, err := core.Write(c, &Packet{
		Type: Register,
		Body: conv.Bytes(RegisterReq{
			Port:     this.Port,
			Username: this.Username,
			Password: this.Password,
		}),
	})
	if err != nil {
		return err
	}

	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	bs, err := core.Read(bufio.NewReader(c))
	if err != nil {
		return err
	}
	c.SetReadDeadline(time.Time{})

	p, err := Decode(bs)
	if err != nil {
		return err
	}

	if p.IsResponse() && p.Success() {
		return nil
	}

	return errors.New(string(p.Body))
}

type RegisterReq struct {
	Port     int    //监听端口
	Username string //用户名
	Password string //密码
}

type virtual struct {
	c   net.Conn
	buf *bytes.Buffer
}

func (this *virtual) Read(p []byte) (n int, err error) {
	return this.buf.Read(p)
}

func (this *virtual) Write(p []byte) (n int, err error) {
	return core.Write(this.c, &Packet{
		Type: Write,
		Body: p,
	})
}

func (this *virtual) Close() error {
	if _, err := core.Write(this.c, &Packet{Type: Close}); err != nil {
		return err
	}
	//是否需要等待关闭成功?
	return this.c.Close()
}
