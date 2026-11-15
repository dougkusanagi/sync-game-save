//go:build !windows

package main

import "unsafe"

func janelaEmPrimeiroPlano(_ unsafe.Pointer, focada func() bool) bool { return focada() }
