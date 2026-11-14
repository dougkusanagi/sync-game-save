package core

import (
	"testing"
)

func TestActiveProcessesPreservaExecucoesDoMesmoPrograma(t *testing.T) {
	const prog = "teste-registro"
	first, second := &stopControl{}, &stopControl{}
	endFirst := registerProcess(prog, first)
	endSecond := registerProcess(prog, second)
	if !ProcessRunning(prog) || ProcessRunning(prog+"-outro") {
		t.Fatal("registro incorreto")
	}
	endFirst()
	if !ProcessRunning(prog) || first.request() {
		t.Fatal("encerrou outra execução ou aceitou pedido após o fim")
	}
	endSecond()
	if ProcessRunning(prog) {
		t.Fatal("execução permaneceu registrada")
	}
}
