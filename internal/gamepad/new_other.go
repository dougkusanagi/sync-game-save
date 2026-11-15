//go:build !windows && !linux

package gamepad

import "errors"

// New: neste sistema a leitura de controle não foi portada; o app segue só com teclado e mouse.
func New() (Source, error) { return nil, errors.New("leitura de controle indisponível neste sistema") }
