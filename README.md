# Proxy

一个基于 Go 的轻量级隧道代理框架，支持多客户端连接、端口转发、内网穿透等功能。

## 项目简介

本项目旨在解决远程设备管理和访问的需求。多个设备（单片机/边缘网关）可以连接到同一个服务端，通过服务端进行数据转发和桥接，实现：

- 多客户端共享服务端端口
- 基于注册信息的客户端验证
- 灵活的数据流向控制
- 内网穿透和远程代理

![代理示意](./docs/代理示意2.png)

## 核心特性

- **多客户端支持**：多个客户端可共用服务端端口，由服务端控制数据流向
- **客户端验证**：服务端可验证客户端身份
- **灵活转发**：支持端口转发、隧道代理、内网穿透
- **协议自定义**：支持自定义帧协议和数据转换

## 项目结构

```
proxy/
├── core/          # 核心包
│   ├── frame.go   # 帧协议定义
│   ├── io.go      # 虚拟IO
│   ├── model.go   # 数据模型
│   ├── tunnel.go  # 隧道核心
│   ├── option.go  # 隧道选项
│   ├── listen.go  # 网络监听
│   └── util.go    # 工具函数
├── tunnel/        # 隧道客户端和服务端
├── forward/       # 端口转发
├── special/       # 特殊模式（隧道和代理共用端口）
└── example/       # 示例代码
```

## 快速开始

### 端口转发

将本地端口转发到目标地址：

```go
package main

import (
    "context"
    "github.com/injoyai/proxy/core"
    "github.com/injoyai/proxy/forward"
)

func main() {
    f := forward.Forward{
        Listen:  core.NewListenTCP(20002),
        Forward: core.NewDialTCP("192.168.10.187:10001"),
    }
    f.Run(context.Background())
}
```

### 隧道代理

#### 服务端

```go
package main

import (
    "context"
    "github.com/injoyai/proxy/core"
    "github.com/injoyai/proxy/tunnel"
    "io"
)

func main() {
    s := tunnel.Server{
        Listen: core.NewListenTCP(7000),
        OnRegister: func(tun *core.Tunnel, reg *core.RegisterReqExtend) error {
            // 验证客户端注册信息
            return nil
        },
    }
    s.Run(context.Background())
}
```

#### 客户端

```go
package main

import (
    "github.com/injoyai/proxy/core"
    "github.com/injoyai/proxy/tunnel"
    "time"
)

func main() {
    c := tunnel.Client{
        Dialer: &core.Dial{
            Address: "127.0.0.1:7000",
            Timeout: time.Second * 2,
        },
        Register: &core.RegisterReq{
            Listen:   core.NewListenTCP(20001),
            Username: "username",
            Password: "password",
        },
    }
    c.Run(core.WithDialTCP(":80"))
}
```

### 特殊模式

隧道和代理共用同一个端口，根据帧头自动识别：

```go
package main

import (
    "fmt"
    "github.com/injoyai/proxy/core"
    "github.com/injoyai/proxy/special"
)

func main() {
    s := special.New(
        special.WithPort(7001),
        special.WithAddress(":80"),
        special.WithRegister(func(tun *core.Tunnel, reg *core.RegisterReqExtend) error {
            fmt.Printf("[%s] 注册成功\n", tun.Key())
            return nil
        }),
    )
    s.Run()
}
```

## 协议说明

默认帧协议格式：

```
| 0x89 | 0x89 | 长度(4字节) | MsgID | # | Code | Data |
```

- `0x8989`：帧头标识
- 长度：MsgID + Code + Data 的总长度
- MsgID：消息唯一标识
- `#`：分隔符
- Code：控制码（消息类型+请求/响应标识）
- Data：数据内容

## License

MIT
