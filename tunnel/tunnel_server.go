package tunnel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/injoyai/base/maps/wait"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"net"
	"sync"
	"time"
)

type Tunnel struct {
	Port       int                                    //客户端连接的端口
	m          sync.Map                               //tun的虚拟设备
	OnRegister func(c net.Conn, r *RegisterReq) error //注册事件
}

func (this *Tunnel) ListenTCP() error {
	return core.Listen("tcp", this.Port, core.WithListenLog, this.Handler)
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
	if err := json.Unmarshal(p.Data, register); err != nil {
		return nil, err
	}

	//注册事件
	if this.OnRegister != nil {
		if err := this.OnRegister(tun, register); err != nil {

			logs.Tracef("[:%d] 代理注册失败: %s\n", register.Port, err.Error())
			core.Write(tun, &Packet{
				Code: Response | Fail,
				Type: Register,
				Data: []byte(err.Error()),
			})

			return nil, err
		}
	}

	logs.Tracef("[:%d] 代理注册成功...\n", register.Port)
	core.Write(tun, &Packet{
		Code: Response | Success,
		Type: Register,
	})

	return register, nil
}

// copyTun2Conn 从tun
func (this *Tunnel) copyTun2Conn(tun net.Conn) {
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
		if val, _ := this.m.Load(p.Key); val != nil {
			c := val.(*core.Virtual)
			switch p.Type {
			case Write:
				//写入数据到客户端
				c.Write(p.Data)

			case Close:
				//关闭客户端
				//设置写超时,会等缓存的数据写完,再进行Close操作
				//c.SetWriteDeadline(time.Time{})
				c.Close()

			case Open:

				logs.Debug(p)
				if p.IsResponse() && p.Success() {
					wait.Done(p.Key, nil)
				}

			}
		} else {
			//客户端已断开连接
			//向tun发送个关闭的数据包
			core.Write(tun, &Packet{Type: Open, Key: p.Key})

		}
	}
}

func (this *Tunnel) Handler(tunListen net.Listener, tun net.Conn) error {
	defer tun.Close()

	//解析注册信息
	register, err := this.handlerRegister(tun)
	if err != nil {
		return err
	}

	//根据注册信息,进行连接操作
	err = core.Listen("tcp", register.Port, func(l net.Listener) {
		go func() {
			logs.Infof("[:%d] 开始监听...\n", register.Port)
			defer logs.Infof("[:%d] 关闭监听...\n", register.Port)
			defer l.Close()
			this.copyTun2Conn(tun)
		}()

	}, func(l net.Listener, c net.Conn) error {
		logs.Trace("新的客户端连接: ", c.RemoteAddr().String())
		key := c.RemoteAddr().String()

		//建立连接请求
		if _, err := core.Write(tun, &Packet{Type: Open, Key: key}); err != nil {
			logs.Trace("建立连接请求错误: ", err.Error())
			return err
		}
		//等待建立连接结果
		if _, err := wait.Wait(key, time.Second*2); err != nil {
			logs.Trace("等待连接结果错误: ", err.Error())
			return err
		}

		//新建虚拟io设备
		v := newVirtual(key, tun)
		defer logs.Trace("关闭虚拟设备: ", key)
		defer v.Close()

		//储存到缓存中
		this.m.Store(key, v)
		defer this.m.Delete(key)

		return core.Swap(v, c)
	})

	tun.Write((&Packet{
		Type: Info,
		Data: []byte(fmt.Sprintf("监听端口[:%d]失败: %s", register.Port, err.Error())),
	}).Bytes())
	logs.Errf("[%s] 监听端口[:%d]失败: %s\n", tun.RemoteAddr().String(), register.Port, err.Error())
	return err
}

/*



 */
