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
	Address      string //服务地址
	Port         int    //监听端口
	ProxyAddress string //代理地址
	Username     string //用户名
	Password     string //密码
	m            sync.Map
}

func (this *Client) Dial() error {

	tun, err := net.Dial("tcp", this.Address)
	if err != nil {
		return err
	}
	defer tun.Close()

	//进行注册认证
	if err := this.Register(tun); err != nil {
		logs.Trace("注册到服务错误: ", err.Error())
		return err
	}
	logs.Trace("注册到服务成功")

	r := bufio.NewReader(tun)
	for {

		bs, err := core.Read(r)
		if err != nil {
			logs.Error(err)
			return err
		}

		logs.Trace("读取到数据: ", string(bs))

		p, err := Decode(bs)
		if err != nil {
			logs.Error(err)
			continue
		}

		switch p.Type {
		case Open:

			logs.Trace("建立代理连接: ", this.ProxyAddress)
			c, err := net.Dial("tcp", this.ProxyAddress)
			if err != nil {
				logs.Trace("建立代理失败: ", err.Error())
				//关闭
				core.Write(tun, &Packet{
					Type: Close,
					Data: conv.Bytes(fmt.Sprintf("%s", err)),
					Key:  p.Key,
				})
				continue
			}
			logs.Trace("建立连接成功")

			//响应连接成功
			core.Write(tun, &Packet{
				Code: 0x80,
				Type: Open,
				Key:  p.Key,
			})

			go func(key string, c net.Conn) error {
				v := newVirtual(key, c)
				defer v.Close()
				this.m.Store(key, v)
				defer this.m.Delete(key)
				return core.Swap(tun, v)
			}(p.Key, c)

		case Write:

			if v, _ := this.m.Load(p.Key); v != nil {
				v.(*core.Virtual).Write(p.Data)
			}

		case Close:

			if v, _ := this.m.Load(p.Key); v != nil {
				v.(*core.Virtual).Close()
			}

		}

	}

}

func (this *Client) Register(c net.Conn) error {

	_, err := core.Write(c, &Packet{
		Type: Register,
		Data: conv.Bytes(RegisterReq{
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

	return errors.New(string(p.Data))
}

func (this *Client) handlerOpen(c net.Conn) {
	_, err := core.Write(c, &Packet{
		Type: Register,
		Data: conv.Bytes(RegisterReq{
			Port:     this.Port,
			Username: this.Username,
			Password: this.Password,
		}),
	})
}

type RegisterReq struct {
	Port     int    //监听端口
	Username string //用户名
	Password string //密码
}

func newVirtual(key string, tun net.Conn) *core.Virtual {
	return &core.Virtual{
		Key:    key,
		Tun:    tun,
		Buffer: bytes.NewBuffer(nil),
		OnWrite: func(p []byte) ([]byte, error) {
			return (&core.Packet{Data: (&Packet{
				Type: Write,
				Data: p,
			}).Bytes()}).Bytes(), nil
		},
		OnClose: func(v *core.Virtual) error {
			if _, err := core.Write(v.Tun, &Packet{Type: Close}); err != nil {
				return err
			}
			return nil
		},
	}
}
