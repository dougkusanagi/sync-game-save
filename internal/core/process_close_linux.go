//go:build linux

package core

import "errors"

// A versão anterior também não envia pedido de fechamento normal no Linux.
func (*processJob) requestClose() error { return nil }

func forceStopAvailable() error {
	return errors.New("O fechamento forçado ainda não está disponível neste sistema.")
}
func (*processJob) forceStop() error { return forceStopAvailable() }
