// Package core 提供隧道代理的核心功能
package core

import (
	"io"
	"net"
	"time"

	"github.com/injoyai/base/maps/wait"
	"github.com/injoyai/conv"
)

// OptionTunnel 隧道配置函数类型,用于 NewTunnel 和 SetOption 方法
type OptionTunnel func(v *Tunnel)

// WithKey 设置隧道的唯一标识
// 默认使用连接的内存地址作为标识
func WithKey(k string) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.k = k
	}
}

// WithFrame 设置自定义帧协议
// 默认使用 DefaultFrame
func WithFrame(f Frame) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.f = f
	}
}

// WithWait 设置异步等待机制
// 用于等待请求的响应,默认超时时间为5秒
func WithWait(w *wait.Entity) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.wait = w
	}
}

// WithWaitTimeout 设置异步等待的超时时间
func WithWaitTimeout(timeout time.Duration) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.wait.SetTimeout(timeout)
	}
}

// WithRegister 设置注册回调函数
// 服务端使用此函数验证客户端的注册信息
// 返回 nil 表示注册成功,返回 error 表示注册失败
func WithRegister(f func(v *Tunnel, p Packet) (interface{}, error)) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.onRegister = f
	}
}

// WithDialed 设置连接成功回调函数
// 当通过 OnDial 成功建立到目标的连接后调用
func WithDialed(f func(d *Dial, key string)) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.onDialed = f
	}
}

// WithDial 设置自定义拨号函数
// 用于控制隧道如何建立到目标地址的连接
// 如果 f 为 nil 则使用默认的拨号方式
func WithDial(f func(d *Dial) (io.ReadWriteCloser, string, error)) func(v *Tunnel) {
	if f == nil {
		return WithDialDefault()
	}
	return func(v *Tunnel) { v.dial = f }
}

// WithDialTCP 设置 TCP 拨号函数
// 忽略服务端下发的连接配置,直接使用指定的地址和超时时间
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

// WithDialDefault 使用默认的拨号方式
// 即使用 Dial 结构体中指定的配置进行连接
func WithDialDefault() func(v *Tunnel) {
	return WithDial(func(d *Dial) (io.ReadWriteCloser, string, error) {
		return d.Dial()
	})
}

// WithRegistered 设置隧道的注册状态
// 可用于跳过注册流程,适用于不需要认证的场景
func WithRegistered(b ...bool) func(v *Tunnel) {
	return func(v *Tunnel) {
		v.registered.Store(len(b) > 0 && b[0])
	}
}
