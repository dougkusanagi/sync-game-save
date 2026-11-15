package gamepad

import (
	"slices"
	"testing"
	"time"
)

func TestBotoesViramAcoes(t *testing.T) {
	for b, want := range map[Button]string{South: "confirm", East: "back", DPadLeft: "left", Mode: "quick", RightThumb: "quick", LeftThumb: "help", Select: "search", Start: "menu"} {
		if got, ok := Action(b); !ok || got != want {
			t.Fatal(b, got)
		}
	}
	if _, ok := Action(Button(99)); ok {
		t.Fatal("botão desconhecido")
	}
}

func TestAnalogicoUsaLimiarECimaEPositivo(t *testing.T) {
	for _, c := range []struct {
		axis Axis
		v    float32
		want string
	}{{LeftStickX, 0.9, "right"}, {LeftStickX, -0.9, "left"}, {LeftStickY, 0.9, "up"}, {LeftStickY, -0.9, "down"}, {LeftStickX, 0.3, ""}, {RightStickX, 0.9, ""}} {
		if got, _ := Direction(c.axis, c.v); got != c.want {
			t.Fatal(c, got)
		}
	}
}

func TestDirecionalRepeteDepoisDeUmAtrasoEConfirmarNao(t *testing.T) {
	t0 := time.Now()
	var r Repeater
	if !r.Press("right", t0) || r.Press("right", t0) {
		t.Fatal("segurar não emite de novo")
	}
	if len(r.Repeats(t0.Add(100*time.Millisecond))) != 0 {
		t.Fatal("repetiu cedo")
	}
	if got := r.Repeats(t0.Add(400 * time.Millisecond)); !slices.Equal(got, []string{"right"}) {
		t.Fatal(got)
	}
	if len(r.Repeats(t0.Add(450*time.Millisecond))) != 0 {
		t.Fatal("deve esperar o intervalo")
	}
	if got := r.Repeats(t0.Add(500 * time.Millisecond)); !slices.Equal(got, []string{"right"}) {
		t.Fatal(got)
	}
	r.Release("right")
	if len(r.Repeats(t0.Add(5*time.Second))) != 0 {
		t.Fatal("repetiu depois de soltar")
	}
	if !r.Press("confirm", t0) || len(r.Repeats(t0.Add(5*time.Second))) != 0 {
		t.Fatal("confirmar não repete")
	}
	r.Release("confirm")
	if !r.Press("confirm", t0) {
		t.Fatal("pode apertar de novo depois de soltar")
	}
}

func TestReconheceControlePlayStation(t *testing.T) {
	if DeviceKind("Sony DualSense Wireless Controller") != "ps" || DeviceKind("Xbox Series X Controller") != "xbox" {
		t.Fatal()
	}
}

type fila struct{ lotes [][]Event }

func (f *fila) Poll() []Event {
	if len(f.lotes) == 0 {
		return nil
	}
	l := f.lotes[0]
	f.lotes = f.lotes[1:]
	return l
}

func TestLoopSoEmiteComAJanelaPronta(t *testing.T) {
	src := &fila{lotes: [][]Event{
		{{Kind: ButtonPressed, Button: South, Device: "ps"}},
		{{Kind: ButtonReleased, Button: South}, {Kind: AxisChanged, Axis: LeftStickX, Value: 0.9}},
		{{Kind: AxisChanged, Axis: LeftStickX, Value: 0.1}},
	}}
	pronta := false
	var got []Output
	stop := make(chan struct{})
	agora := time.Now()
	n := 0
	ready := func() bool { n++; return pronta }
	done := make(chan struct{})
	go func() {
		Loop(src, ready, func(o Output) { got = append(got, o) }, func() time.Time { return agora }, stop, time.Millisecond)
		close(done)
	}()
	time.Sleep(30 * time.Millisecond)
	close(stop)
	<-done
	if len(got) != 0 || n == 0 {
		t.Fatal("emitiu com a janela escondida", got)
	}

	pronta = true
	src = &fila{lotes: [][]Event{
		{{Kind: ButtonPressed, Button: South, Device: "ps"}},
		{{Kind: ButtonReleased, Button: South}, {Kind: AxisChanged, Axis: LeftStickX, Value: 0.9}},
		{{Kind: AxisChanged, Axis: LeftStickX, Value: 0.1}},
	}}
	got = nil
	stop = make(chan struct{})
	done = make(chan struct{})
	go func() {
		Loop(src, ready, func(o Output) { got = append(got, o) }, func() time.Time { return agora }, stop, time.Millisecond)
		close(done)
	}()
	time.Sleep(30 * time.Millisecond)
	close(stop)
	<-done
	want := []Output{{"confirm", "ps"}, {"right", "ps"}}
	if !slices.Equal(got, want) {
		t.Fatal(got)
	}
}
