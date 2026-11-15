package gamepad

import (
	"slices"
	"testing"
)

func novoDecoder() *evdevDecoder {
	return newEvdevDecoder("xbox", map[uint16]absRange{absX: {-32768, 32767}, absY: {-32768, 32767}, absZ: {0, 255}, absRZ: {0, 255}})
}

func TestEvdevBotoes(t *testing.T) {
	d := novoDecoder()
	if got := d.Decode(evKey, btnSouth, 1); !slices.Equal(got, []Event{{Kind: ButtonPressed, Button: South, Device: "xbox"}}) {
		t.Fatal(got)
	}
	if got := d.Decode(evKey, btnSouth, 0); !slices.Equal(got, []Event{{Kind: ButtonReleased, Button: South, Device: "xbox"}}) {
		t.Fatal(got)
	}
	if got := d.Decode(evKey, btnSouth, 2); len(got) != 0 {
		t.Fatal("a repetição automática do teclado (valor 2) não deve gerar evento", got)
	}
	if got := d.Decode(evKey, 0x2ff, 1); len(got) != 0 {
		t.Fatal("botão desconhecido", got)
	}
	// Cada botão do mapa vira a ação esperada.
	for code, b := range evdevButtons {
		if got := d.Decode(evKey, code, 1); len(got) != 1 || got[0].Button != b {
			t.Fatal(code, got)
		}
	}
}

func TestEvdevDirecionalViraBotoes(t *testing.T) {
	d := novoDecoder()
	if got := d.Decode(evAbs, absHat0X, 1); len(got) != 1 || got[0].Button != DPadRight || got[0].Kind != ButtonPressed {
		t.Fatal(got)
	}
	// Ir direto da direita para a esquerda solta um e aperta o outro.
	got := d.Decode(evAbs, absHat0X, -1)
	if len(got) != 2 || got[0].Button != DPadRight || got[0].Kind != ButtonReleased || got[1].Button != DPadLeft || got[1].Kind != ButtonPressed {
		t.Fatal(got)
	}
	if got := d.Decode(evAbs, absHat0X, 0); len(got) != 1 || got[0].Button != DPadLeft || got[0].Kind != ButtonReleased {
		t.Fatal(got)
	}
	// Para cima é negativo no evdev.
	if got := d.Decode(evAbs, absHat0Y, -1); len(got) != 1 || got[0].Button != DPadUp || got[0].Kind != ButtonPressed {
		t.Fatal(got)
	}
	if got := d.Decode(evAbs, absHat0Y, 1); len(got) != 2 || got[1].Button != DPadDown {
		t.Fatal(got)
	}
}

func TestEvdevAnalogicoNormalizadoECimaPositivo(t *testing.T) {
	d := novoDecoder()
	if got := d.Decode(evAbs, absX, 32767); len(got) != 1 || got[0].Axis != LeftStickX || got[0].Value < 0.99 {
		t.Fatal(got)
	}
	if got := d.Decode(evAbs, absX, -32768); got[0].Value > -0.99 {
		t.Fatal(got)
	}
	if got := d.Decode(evAbs, absX, 0); got[0].Value < -0.01 || got[0].Value > 0.01 {
		t.Fatal("o centro deve ser zero", got)
	}
	// Eixo Y do evdev cresce para baixo; o pacote usa positivo para cima.
	if got := d.Decode(evAbs, absY, -32768); got[0].Axis != LeftStickY || got[0].Value < 0.99 {
		t.Fatal(got)
	}
	// A direção chega à ação certa pelo mesmo caminho do XInput.
	if dir, ok := Direction(LeftStickY, d.Decode(evAbs, absY, -32768)[0].Value); !ok || dir != "up" {
		t.Fatal(dir)
	}
	// Sem intervalo conhecido, o eixo fica no centro em vez de disparar ações.
	if got := newEvdevDecoder("xbox", nil).Decode(evAbs, absX, 30000); got[0].Value != 0 {
		t.Fatal(got)
	}
}

func TestEvdevGatilhosAnalogicosTemHisterese(t *testing.T) {
	d := novoDecoder()
	if got := d.Decode(evAbs, absZ, 100); len(got) != 0 {
		t.Fatal("abaixo do limite", got)
	}
	if got := d.Decode(evAbs, absZ, 200); len(got) != 1 || got[0].Button != LeftTrigger2 || got[0].Kind != ButtonPressed {
		t.Fatal(got)
	}
	if got := d.Decode(evAbs, absZ, 100); len(got) != 0 {
		t.Fatal("entre os limites mantém pressionado", got)
	}
	if got := d.Decode(evAbs, absZ, 10); len(got) != 1 || got[0].Kind != ButtonReleased {
		t.Fatal(got)
	}
	if got := d.Decode(evAbs, absRZ, 255); len(got) != 1 || got[0].Button != RightTrigger2 {
		t.Fatal(got)
	}
}
