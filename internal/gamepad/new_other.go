//go:build !windows

package gamepad

import "errors"

// New: no Linux a leitura por evdev ainda não foi portada; o app segue só com teclado e mouse.
func New() (Source, error) { return nil, errors.New("leitura de controle indisponível neste sistema") }
