package core

import (
	"github.com/fatih/color"
	"github.com/injoyai/logs"
	"strings"
)

type Level int

const (
	LevelTrace Level = iota
	LevelRead
	LevelWrite
	LevelInfo
	LevelError
	LevelNone
)

var DefaultLog Log = &_log{
	trace: logs.NewEntity("跟踪").SetSelfLevel(logs.LevelTrace).SetCaller(1).SetColor(color.FgGreen),
	read:  logs.NewEntity("读取").SetSelfLevel(logs.LevelRead).SetCaller(1).SetColor(color.FgBlue),
	write: logs.NewEntity("写入").SetSelfLevel(logs.LevelWrite).SetCaller(1).SetColor(color.FgBlue),
	info:  logs.NewEntity("信息").SetSelfLevel(logs.LevelInfo).SetCaller(1).SetColor(color.FgCyan),
	err:   logs.NewEntity("错误").SetSelfLevel(logs.LevelError).SetCaller(1).SetColor(color.FgRed),
}

type Log interface {
	Tracef(format string, args ...interface{})
	Read(args ...interface{})
	Write(args ...interface{})
	Infof(format string, args ...interface{})
	Errf(format string, args ...interface{})
	PrintErr(err error)
	SetLevel(level Level)
}

type _log struct {
	trace *logs.Entity
	read  *logs.Entity
	write *logs.Entity
	info  *logs.Entity
	err   *logs.Entity
}

func (this *_log) Tracef(format string, args ...interface{}) {
	this.trace.Printf(format, args...)
}

func (this *_log) Read(args ...interface{}) {
	this.read.Println(args...)
}

func (this *_log) Write(args ...interface{}) {
	this.write.Println(args...)
}

func (this *_log) Infof(format string, args ...interface{}) {
	this.info.Printf(format, args...)
}

func (this *_log) Errf(format string, args ...interface{}) {
	this.err.Printf(format, args...)
}

func (this *_log) PrintErr(err error) {
	if err != nil {
		this.err.Println(err)
	}
}

func (this *_log) SetLevel(level Level) {
	l := logs.LevelInfo
	switch level {
	case LevelTrace:
		l = logs.LevelTrace
	case LevelRead:
		l = logs.LevelRead
	case LevelWrite:
		l = logs.LevelWrite
	case LevelInfo:
		l = logs.LevelInfo
	case LevelError:
		l = logs.LevelError
	case LevelNone:
		l = logs.LevelNone
	}
	this.trace.SetLevel(l)
	this.read.SetLevel(l)
	this.write.SetLevel(l)
	this.info.SetLevel(l)
	this.err.SetLevel(l)
}

func (this *_log) SetLevelStr(level string) {
	l := logs.LevelInfo
	switch strings.ToLower(level) {
	case "all":
		l = logs.LevelAll
	case "trace":
		l = logs.LevelTrace
	case "debug":
		l = logs.LevelDebug
	case "write":
		l = logs.LevelWrite
	case "read":
		l = logs.LevelRead
	case "info":
		l = logs.LevelInfo
	case "warn":
		l = logs.LevelWarn
	case "error", "err":
		l = logs.LevelError
	case "none":
		l = logs.LevelNone
	default:
		l = logs.LevelInfo
	}
	this.trace.SetLevel(l)
	this.read.SetLevel(l)
	this.write.SetLevel(l)
	this.info.SetLevel(l)
	this.err.SetLevel(l)
}

func (this *_log) SetFormatterWithTime() {
	this.trace.SetFormatter(logs.TimeFormatter)
	this.read.SetFormatter(logs.TimeFormatter)
	this.write.SetFormatter(logs.TimeFormatter)
	this.info.SetFormatter(logs.TimeFormatter)
	this.err.SetFormatter(logs.TimeFormatter)
}

func (this *_log) SetShowColor(b ...bool) {
	this.trace.SetShowColor(b...)
	this.read.SetShowColor(b...)
	this.write.SetShowColor(b...)
	this.info.SetShowColor(b...)
	this.err.SetShowColor(b...)
}
