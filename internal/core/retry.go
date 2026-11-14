package core

import (
	"errors"
	"os"
	"syscall"
	"time"
)

// Repete somente erros de acesso ou compartilhamento, como o núcleo Rust.
func retryFile(op func() error) error {
	for attempts := 0; ; attempts++ {
		err := op()
		if err == nil || attempts == 8 || !(errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.Errno(32)) || errors.Is(err, syscall.Errno(33))) {
			return err
		}
		time.Sleep(250 * time.Millisecond)
	}
}
