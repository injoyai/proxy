package core

import "errors"

var (
	ErrNotRegister = errors.New("未注册")
	ErrRemoteClose = errors.New("远程意外关闭连接")
	ErrDialInvalid = errors.New("无效的连接函数")
)
