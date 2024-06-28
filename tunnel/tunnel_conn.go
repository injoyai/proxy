package tunnel

import (
	"bufio"
	"errors"
	"github.com/injoyai/base/maps/wait/v2"
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"net"
	"sync"
	"time"
)

func NewConn(c net.Conn, timeout time.Duration) *Conn {
	return &Conn{
		c:       c,
		buf:     bufio.NewReader(c),
		wait:    wait.New(timeout),
		timeout: timeout,
	}
}

type Conn struct {
	c       net.Conn
	buf     *bufio.Reader
	wait    *wait.Entity  //超时机制
	m       sync.Map      //虚拟IO
	timeout time.Duration //超时时间
}

func (this *Conn) Key() string {
	return this.c.RemoteAddr().String()
}

func (this *Conn) newVirtual(key string) (v *core.Virtual) {
	v = core.NewVirtual(
		key,
		this.c,
		core.NewChanIO(20),
		func(bs []byte) ([]byte, error) {
			p := &Packet{
				Code: Request,
				Key:  key,
				Type: Write,
				Data: bs,
			}
			pp := &core.Packet{Data: p.Bytes()}
			return pp.Bytes(), nil

		}, func(v *core.Virtual, err error) error {
			this.m.Delete(key)
			this.WriteClosePacket(err)
			return nil
		},
	)
	this.m.Store(key, v)
	return v
}

func (this *Conn) runRead(proxy string) error {
	for {

		//读取数据包
		p, err := this.ReadPacket()
		if err != nil {
			return err
		}

		switch p.Type {

		case Open:

			if p.IsRequest() {
				logs.Tracef("[%s] 建立代理连接: %s\n", p.Key, proxy)
				c, err := net.Dial("tcp", proxy)
				if err != nil {
					logs.Trace("建立代理失败: ", err.Error())
					//响应失败
					this.WritePacket(p.Resp(Fail, err))
					continue
				}
				logs.Tracef("[%s] 建立代理连接成功...\n", p.Key)

				//响应连接成功
				this.WritePacket(p.Resp(Success, nil))

				//数据通讯
				go this.Swap(c, nil)

			} else if p.Success() {
				this.wait.Done(p.Key, p)
			}

		case Write:

			if val, _ := this.m.Load(p.Key); val != nil {
				//写入数据到虚拟客户端
				val.(*core.Virtual).ToBuffer(p.Data)
			} else {
				//客户端已断开连接
				//向tun发送个关闭的数据包
				this.WritePacket(&Packet{Type: Close, Key: p.Key, Data: []byte("客户端已断开连接")})
			}

		case Close:

			//关闭客户端
			//设置写超时,会等缓存的数据写完,再进行Close操作
			if val, _ := this.m.Load(p.Key); val != nil {
				val.(*core.Virtual).CloseWithErr(p.Err())
			}

		}

	}
}

func (this *Conn) ReadPacket() (*Packet, error) {
	bs, err := core.Read(this.buf)
	if err != nil {
		return nil, err
	}
	p, err := Decode(bs)
	if err != nil {
		return nil, err
	}
	logs.Read(p)
	return p, nil
}

func (this *Conn) WritePacket(p *Packet) (n int, err error) {
	logs.Write(p)
	pp := &core.Packet{Data: p.Bytes()}
	return this.c.Write(pp.Bytes())
}

func (this *Conn) WriteClosePacket(err error) error {
	_, er := this.WritePacket(&Packet{Type: Close, Key: this.Key(), Data: conv.Bytes(err)})
	return er
}

func (this *Conn) Write(key string, p []byte) (int, error) {
	return this.WritePacket(&Packet{
		Code: Request,
		Key:  key,
		Type: Write,
		Data: p,
	})
}

func (this *Conn) CloseWithErr(err error) error {
	this.m.Range(func(key, value any) bool {
		value.(*core.Virtual).CloseWithErr(err)
		return true
	})
	this.m = sync.Map{}
	return this.c.Close()
}

func (this *Conn) Close() error {
	return this.CloseWithErr(errors.New("主动关闭"))
}

func (this *Conn) Swap(c net.Conn, handler func(tun *Conn, c net.Conn, key string) error) (err error) {
	defer c.Close()
	key := c.RemoteAddr().String()
	logs.Tracef("[%s] 新的客户端, 建立代理通道... \n", key)
	defer logs.Tracef("[%s] 关闭客户端连接: %v \n", key, err)
	//新建虚拟io设备
	v := this.newVirtual(key)
	defer v.Close()
	if handler != nil {
		if err = handler(this, c, key); err != nil {
			return err
		}
	}
	logs.Tracef("[%s] 代理通道建立成功 \n", key)

	return core.Swap(v, c)
}
