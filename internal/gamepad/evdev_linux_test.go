package gamepad

import "testing"

// Sem controle conectado (ou sem permissão em /dev/input), a leitura não entrega eventos nem falha.
func TestEvdevSemControleNaoGeraEventos(t *testing.T) {
	src, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		src.Poll()
	}
	if got := src.Poll(); len(got) != 0 {
		t.Log("há um controle real conectado:", len(got), "eventos")
	}
}

func TestTamanhoDoEventoDoKernel(t *testing.T) {
	e, _ := NewEvdev()
	// struct input_event: timeval (16 bytes em 64 bits, 8 em 32 bits) + tipo + código + valor.
	if e.evSize != 24 && e.evSize != 16 {
		t.Fatal(e.evSize)
	}
}
