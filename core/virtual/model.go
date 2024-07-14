package virtual

import (
	"encoding/json"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/logs"
	"github.com/injoyai/proxy/core"
)

type RegisterReq struct {
	Listen   *core.Listen `json:"listen"`             //监听信息
	Username string       `json:"username,omitempty"` //用户名
	Password string       `json:"password,omitempty"` //密码
	Param    g.Map        `json:"param,omitempty"`    //其他参数
}

func (this *RegisterReq) String() string {
	bs, err := json.Marshal(this)
	logs.PrintErr(err)
	return string(bs)
}
