package main

import (
	"github.com/injoyai/logs"
	"os"
)

func main() {
	logs.SetWriter(os.Stdout)
	go RunTunnel(10007)
	listen := &Listen{Port: 8200}
	listen.Run()
}
