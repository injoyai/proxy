package virtual

type RegisterReq struct {
	Port     int    `json:"port"`     //监听端口
	Username string `json:"username"` //用户名
	Password string `json:"password"` //密码
}
