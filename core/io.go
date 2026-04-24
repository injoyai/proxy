// Package core 提供隧道代理的核心功能
package core

import (
	"io"

	"github.com/injoyai/base/chans"
	"github.com/injoyai/base/safe"
)

// 编译期检查 IO 是否实现了 io.ReadWriteCloser 接口
var _ io.ReadWriteCloser = (*IO)(nil)

// IOOption IO 的配置函数类型
type IOOption func(v *IO)

// NewIO 创建一个新的虚拟IO实例
// w 是底层写入通道(即隧道连接),数据通过它发送到对端
// op 是可选的配置函数,用于设置 OnWrite 和 OnClose 等回调
func NewIO(w io.Writer, op ...IOOption) *IO {
	i := &IO{
		writer: w,
		reader: chans.NewIO(20),
		Closer: safe.NewCloser(),
	}
	for _, v := range op {
		v(i)
	}
	i.SetCloseFunc(func(error) error {
		if i.OnClose != nil {
			return i.OnClose(i, i.Err())
		}
		return i.reader.Close()
	})
	return i
}

// IO 虚拟IO,是隧道的核心组件之一
// 每个IO代表一条独立的虚拟通道,多条IO可以复用同一条物理隧道连接
// 数据流向:
//   - 写入: 调用 Write() -> OnWrite回调 -> 打包 -> 通过writer发送到隧道
//   - 读取: 对端数据通过 ToRead() 写入 -> reader缓冲 -> 调用 Read() 读取
type IO struct {
	writer       io.Writer                    // writer 虚拟(公共)写入通道,即隧道连接
	reader       *chans.IO                    // reader 虚拟读取通道,带缓冲
	*safe.Closer                              // Closer 安全关闭控制器
	OnWrite      func([]byte) ([]byte, error) // OnWrite 写入回调,用于数据打包和日志记录
	OnClose      func(v *IO, err error) error // OnClose 关闭回调,用于通知对端和清理资源
}

// ToRead 将数据写入内部缓冲区,数据会流转到 Read 方法
// 该方法由隧道收到 Write 类型数据包时调用,将数据注入到虚拟IO中
func (this *IO) ToRead(p []byte) error {
	if this.Closed() {
		return this.Err()
	}
	_, err := this.reader.Write(p)
	return err
}

// Read 从虚拟IO中读取数据
// 如果IO已关闭则返回错误,否则阻塞等待数据到达
func (this *IO) Read(p []byte) (n int, err error) {
	if this.Closed() {
		return 0, this.Err()
	}
	return this.reader.Read(p)
}

// Write 向虚拟IO中写入数据
// 数据会经过 OnWrite 回调处理(打包成帧),然后通过底层隧道连接发送
// 返回值为原始数据长度,内部细节对外透明
func (this *IO) Write(p []byte) (n int, err error) {
	if this.Closed() {
		return 0, this.Err()
	}
	if this.reader.Closed() {
		return 0, io.EOF
	}
	// 取原始长度,外部调用者不关心内部细节
	n = len(p)
	if this.OnWrite != nil {
		p, err = this.OnWrite(p)
		if err != nil {
			return 0, err
		}
		if p == nil {
			return
		}
	}
	_, err = this.writer.Write(p)
	return
}
