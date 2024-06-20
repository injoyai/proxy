package tunnel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"net"
	"sync"
	"time"
)

type Tunnel struct {
	Port       int //客户端连接的端口
	m          sync.Map
	OnRegister func(c net.Conn, r *RegisterReq) error //注册事件
}

func (this *Tunnel) ListenTCP() error {
	return core.Listen("tcp", this.Port, this.Handler)
}

// handlerRegister 对客户端进行注册验证操作
func (this *Tunnel) handlerRegister(tun net.Conn) (*RegisterReq, error) {
	//读取注册数据
	tun.SetDeadline(time.Now().Add(10 * time.Second))
	registerBytes, err := core.Read(bufio.NewReader(tun))
	if err != nil {
		return nil, err
	}
	p, err := Decode(registerBytes)
	if err != nil {
		return nil, err
	}
	tun.SetDeadline(time.Time{})

	//解析注册数据
	register := new(RegisterReq)
	if err := json.Unmarshal(p.Body, register); err != nil {
		return nil, err
	}

	//注册事件
	if this.OnRegister != nil {
		if err := this.OnRegister(tun, register); err != nil {
			return nil, err
		}
	}

	core.Write(tun, &Packet{
		Code: 0x80,
		Type: Register,
	})
	return register, nil
}

func (this *Tunnel) Handler(l net.Listener, tun net.Conn) error {
	defer tun.Close()

	//解析注册信息
	register, err := this.handlerRegister(tun)
	if err != nil {
		return err
	}

	once := sync.Once{}

	//根据注册信息,进行连接操作
	err = core.Listen("tcp", register.Port, func(l net.Listener, c net.Conn) error {

		once.Do(func() {
			go func() {
				defer l.Close()
				r := bufio.NewReader(tun)
				for {
					//读取tun中的数据
					bs, err := core.Read(r)
					if err != nil {
						return
					}

					//解析tun包解析,忽略错误,经过认证,后续错误可以忽略
					p, err := Decode(bs)
					if err != nil {
						continue
					}

					//查询tun指向的的客户端
					//一个通道传输多个连接的数据,
					if val, _ := this.m.Load(p.Address); val != nil {
						c := val.(net.Conn)
						switch p.Type {
						case Write:
							//写入数据到客户端
							c.Write(p.Body)

						case Close:
							//关闭客户端
							//设置写超时,会等缓存的数据写完,再进行Close操作
							c.SetWriteDeadline(time.Time{})
							//c.Close()
						}
					} else {
						//客户端已断开连接
						//向tun发送个关闭的数据包
						tun.Write((&Packet{Type: Close, Address: p.Address}).Bytes())

					}
				}
			}()
		})

		key := c.RemoteAddr().String()
		this.m.Store(key, c)
		defer this.m.Delete(key)

		//建立连接
		tun.Write((&Packet{Type: Open}).Bytes())
		//todo 等待建立连接结果

		//复制c的数据到tun,并增加些头部信息
		rErr, wErr := core.CopyFunc(tun, c, func(bs []byte) ([]byte, error) {
			p := &Packet{
				Type:    Write,
				Address: key,
				Body:    bs,
			}
			return p.Bytes(), nil
		})
		if wErr != nil {
			l.Close()
		}
		return rErr
	})
	tun.Write((&Packet{
		Type: Info,
		Body: []byte(fmt.Sprintf("监听端口[:%d]失败: %s", register.Port, err.Error())),
	}).Bytes())
	logs.Errf("[%s] 监听端口[:%d]失败: %s\n", tun.RemoteAddr().String(), register.Port, err.Error())
	return err
}

/*



 */
