// Package core 提供隧道代理的核心功能
package core

import "errors"

// 预定义错误
var (
	// ErrNotRegister 当未注册的客户端发送非注册消息时返回此错误
	ErrNotRegister = errors.New("未注册")
	// ErrRemoteClose 当远程连接已关闭但本地仍尝试发送数据时返回此错误
	ErrRemoteClose = errors.New("远程意外关闭连接")
	// ErrDialInvalid 当拨号函数未设置或无效时返回此错误
	ErrDialInvalid = errors.New("无效的连接函数")
)
