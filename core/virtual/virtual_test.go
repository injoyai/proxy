package virtual

import (
	"github.com/injoyai/proxy/core"
	"net"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	listenAddr := ":7000"

	listen, err := net.Listen("tcp", listenAddr)
	if err != nil {
		t.Error(err)
		return
	}

	go func() {
		for {
			tun, err := listen.Accept()
			if err != nil {
				t.Error(err)
				return
			}
			New(tun, WithOpenTCP(":10086"))
		}
	}()

	<-time.After(time.Second * 3)

	tun, err := net.Dial("tcp", listenAddr)
	if err != nil {
		t.Error(err)
		return
	}

	v := New(tun, WithOpenTCP(":10086"))

	for {
		<-time.After(time.Second)
		c, err := v.Open(&core.Dial{Address: ":10086"}, nil)
		if err != nil {
			t.Error(err)
			continue
		}

		for {
			<-time.After(time.Second * 5)
			if _, err := c.Write([]byte("ping")); err != nil {
				t.Error(err)
				break
			}

		}
	}
}

func TestDialTCP(t *testing.T) {

	for {
		<-time.After(time.Second)
		c, err := net.Dial("tcp", ":10086")
		if err != nil {
			t.Error(err)
			continue
		}

		for {
			<-time.After(time.Second * 5)
			if _, err := c.Write([]byte("哈哈哈")); err != nil {
				t.Error(err)
				break
			}

		}
	}
}
