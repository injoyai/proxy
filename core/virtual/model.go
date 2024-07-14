package virtual

import (
	"fmt"
	"github.com/injoyai/goutil/g"
	"strings"
)

type RegisterReq struct {
	Port     int    `json:"port"`     //监听端口
	Username string `json:"username"` //用户名
	Password string `json:"password"` //密码
	Param    g.Map  `json:"param"`    //其他参数
}

func (this *RegisterReq) String() string {
	return fmt.Sprintf("监听: %d, 用户名: %s, 密码: %s", this.Port, this.Username, this.Password)
}

type Listen struct {
	Type    string `json:"type"`
	Port    string `json:"port"`
	Param   g.Map  `json:"param"`
	Handler func()
}

func (this *Listen) Listen() {
	switch strings.ToLower(this.Type) {
	case "tcp":
	default:

	}
}
