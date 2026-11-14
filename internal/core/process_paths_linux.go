//go:build linux

package core

func protectedProcessDirectory(_ string) bool { return false }
