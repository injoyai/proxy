package core

import "io"

func Copy(c1, c2 io.ReadWriter) error {
	go io.Copy(c1, c2)
	_, err := io.Copy(c2, c1)
	return err
}
