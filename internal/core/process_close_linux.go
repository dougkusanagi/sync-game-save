//go:build linux

package core

// A versão anterior também não envia pedido de fechamento normal no Linux.
func (*processJob) requestClose() error { return nil }
