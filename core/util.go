package core

import (
	"io"
	"net"
)

func Bridge(c1, c2 io.ReadWriteCloser) error {
	defer c1.Close()
	defer c2.Close()
	ch := make(chan error, 2)
	go func() {
		_, err := io.Copy(c1, c2)
		ch <- err
	}()
	go func() {
		_, err := io.Copy(c2, c1)
		ch <- err
	}()
	return <-ch
}

func DefaultDial(d *Dial) (io.ReadWriteCloser, string, error) {
	c, err := net.DialTimeout("tcp", d.Address, d.Timeout)
	if err != nil {
		return nil, "", err
	}
	return c, c.LocalAddr().String(), nil
}
