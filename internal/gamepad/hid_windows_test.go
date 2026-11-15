package gamepad

import "testing"

func TestTipoDosControlesConectadosNaoQuebra(t *testing.T) {
	if k := tipoDosControlesConectados(); k != "xbox" && k != "ps" {
		t.Fatalf("tipo inesperado: %q", k)
	}
}
