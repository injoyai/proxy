package core

import (
	"io"
	"net"
	"time"

	"github.com/injoyai/base/maps/wait"
	"github.com/injoyai/conv"
)

type OptionTunnel func(v *Tunnel)

func WithKey(k string) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.k = k
	}
}

func WithFrame(f Frame) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.f = f
	}
}

func WithWait(w *wait.Entity) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.wait = w
	}
}

func WithWaitTimeout(timeout time.Duration) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.wait.SetTimeout(timeout)
	}
}

func WithRegister(f func(v *Tunnel, p Packet) (interface{}, error)) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.onRegister = f
	}
}

func WithDialed(f func(d *Dial, key string)) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.onDialed = f
	}
}

func WithDial(f func(d *Dial) (io.ReadWriteCloser, string, error)) func(v *Tunnel) {
	if f == nil {
		return WithDialDefault()
	}
	return func(v *Tunnel) { v.dial = f }
}

func WithDialTCP(address string, timeout ...time.Duration) func(v *Tunnel) {
	return WithDial(func(d *Dial) (io.ReadWriteCloser, string, error) {
		_timeout := conv.Default(0, timeout...)
		c, err := net.DialTimeout("tcp", address, _timeout)
		if err != nil {
			return nil, "", err
		}
		return c, c.LocalAddr().String(), nil
	})
}

func WithDialDefault() func(v *Tunnel) {
	return WithDial(func(d *Dial) (io.ReadWriteCloser, string, error) {
		return d.Dial()
	})
}

func WithRegistered(b ...bool) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.registered.Store(len(b) > 0 && b[0])
	}
}
