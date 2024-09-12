package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
	"github.com/injoyai/proxy/core/virtual"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync/atomic"
	"time"
)

type Listen struct {
	Port           int
	IndexPath      string
	MsgOffline     string
	MsgUnSelect    string
	DefaultAddress string
	Select         *maps.Safe
}

func (this *Listen) Run() error {
	if this.Port <= 0 {
		this.Port = 8200
	}
	if len(this.IndexPath) == 0 {
		this.IndexPath = "/"
	}
	if len(this.MsgOffline) == 0 {
		this.MsgOffline = `HTTP/1.1 200 OK
Content-Type: application/json;charset=utf-8

{"code": 500, "msg": "客户端不在线"}`
	}
	if len(this.MsgUnSelect) == 0 {
		this.MsgUnSelect = `HTTP/1.1 200 OK
Content-Type: application/json;charset=utf-8

{"code": 500, "msg": "未选择客户端"}`
	}
	if len(this.DefaultAddress) == 0 {
		this.DefaultAddress = ":80"
	}
	if this.Select == nil {
		this.Select = maps.NewSafe()
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", this.Port))
	if err != nil {
		return err
	}

	co := atomic.Int32{}
	go func() {
		for {
			<-time.After(time.Second * 5)
			logs.Debug("客户端数量:", co.Load())
		}
	}()

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go func() {
			co.Add(1)
			defer c.Close()
			defer co.Add(-1)
			err = this.handler(c)
			logs.PrintErr(err)
		}()
	}
}

func (this *Listen) handler(c net.Conn) error {

	buf := bufio.NewReader(c)
	prefix, err := buf.Peek(80)
	if err != nil {
		return err
	}
	logs.Debug(strings.Split(string(prefix), "\n")[0])
	defer logs.Debug(strings.Split(string(prefix), "\n")[0], "close")

	switch {
	case bytes.HasPrefix(prefix, []byte("GET "+this.IndexPath+" ")) ||
		bytes.HasPrefix(prefix, []byte("GET "+this.IndexPath+"?")):
		r, err := http.ReadRequest(buf)
		if err != nil {
			return err
		}

		sn := r.URL.Query().Get("sn")
		address := r.URL.Query().Get("address")
		ipv4 := strings.Split(c.RemoteAddr().String(), ":")[0]

		info := &Info{
			SN:      sn,
			Address: conv.SelectString(address == "", this.DefaultAddress, address),
		}

		val, ok := this.Select.Get(ipv4)
		if len(sn) > 0 {
			this.Select.Set(ipv4, info)
		} else if ok {
			//存在且sn无效.使用缓存配置
			info = val.(*Info)
		} else {
			//不存在且sn无效
			c.Write([]byte(this.MsgUnSelect))
			return nil
		}

		v := Tunnel.Clients.MustGet(sn)
		if v == nil {
			c.Write([]byte(this.MsgOffline))
			return nil
		}

		bs, err := httputil.DumpRequest(r, true)
		if err != nil {
			return err
		}
		c1 := bytes.NewReader(bs)
		return v.(*virtual.Virtual).OpenAndSwap(c.RemoteAddr().String(), core.NewDialTCP(info.Address), struct {
			io.Reader
			io.Writer
			io.Closer
		}{io.MultiReader(c1, c), c, c})

	default:
		ipv4 := strings.Split(c.RemoteAddr().String(), ":")[0]
		val, ok := this.Select.Get(ipv4)
		if !ok {
			c.Write([]byte(this.MsgUnSelect))
			return nil
		}
		info := val.(*Info)
		v := Tunnel.Clients.MustGet(info.SN)
		if v == nil {
			c.Write([]byte(this.MsgOffline))
			return nil
		}
		return v.(*virtual.Virtual).OpenAndSwap(c.RemoteAddr().String(), core.NewDialTCP(info.Address), struct {
			io.Reader
			io.Writer
			io.Closer
		}{buf, c, c})

	}

}
