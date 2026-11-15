// Package gamepad lê o controle (Xbox, PlayStation etc.) e transforma os botões nas mesmas
// "ações" do teclado. A lógica é pura e recebe o tempo de fora, para testar sem controle real.
package gamepad

import (
	"strings"
	"time"
)

const (
	firstRepeat = 350 * time.Millisecond
	repeatEvery = 80 * time.Millisecond
	// AnalogThreshold é o limiar do analógico: abaixo dele vale a zona morta.
	AnalogThreshold = 0.5
)

type Button int

const (
	South Button = iota
	East
	West
	North
	LeftTrigger
	RightTrigger
	LeftTrigger2
	RightTrigger2
	Select
	Start
	Mode
	LeftThumb
	RightThumb
	DPadUp
	DPadDown
	DPadLeft
	DPadRight
)

type Axis int

const (
	LeftStickX Axis = iota
	LeftStickY
	RightStickX
	RightStickY
)

// Action devolve a ação da interface para o botão, se houver.
func Action(b Button) (string, bool) {
	switch b {
	case South:
		return "confirm", true
	case East:
		return "back", true
	case West:
		return "filter", true
	case North:
		return "options", true
	case LeftTrigger:
		return "prevTab", true
	case RightTrigger:
		return "nextTab", true
	case LeftTrigger2:
		return "pageUp", true
	case RightTrigger2:
		return "pageDown", true
	case Select:
		return "search", true
	case Start:
		return "menu", true
	case Mode, RightThumb:
		return "quick", true
	case LeftThumb:
		return "help", true
	case DPadUp:
		return "up", true
	case DPadDown:
		return "down", true
	case DPadLeft:
		return "left", true
	case DPadRight:
		return "right", true
	}
	return "", false
}

func repeats(action string) bool {
	switch action {
	case "up", "down", "left", "right", "pageUp", "pageDown":
		return true
	}
	return false
}

// Direction devolve a direção que um eixo analógico indica (ou false perto do centro).
// Para cima é positivo.
func Direction(axis Axis, value float32) (string, bool) {
	switch {
	case axis == LeftStickX && value > AnalogThreshold:
		return "right", true
	case axis == LeftStickX && value < -AnalogThreshold:
		return "left", true
	case axis == LeftStickY && value > AnalogThreshold:
		return "up", true
	case axis == LeftStickY && value < -AnalogThreshold:
		return "down", true
	}
	return "", false
}

// Repeater controla quais ações estão seguradas e quando repetir.
type Repeater struct{ held map[string]time.Time }

// Press registra um aperto. Devolve true se a ação deve ser emitida agora.
func (r *Repeater) Press(action string, now time.Time) bool {
	if r.held == nil {
		r.held = map[string]time.Time{}
	}
	if _, ok := r.held[action]; ok {
		return false
	}
	if repeats(action) {
		r.held[action] = now.Add(firstRepeat)
	} else {
		// Ações sem repetição ficam marcadas até soltar, para não disparar duas vezes.
		r.held[action] = now.Add(time.Hour)
	}
	return true
}

func (r *Repeater) Release(action string) { delete(r.held, action) }

// Repeats devolve as ações que devem repetir agora.
func (r *Repeater) Repeats(now time.Time) []string {
	var out []string
	for action, next := range r.held {
		if repeats(action) && !now.Before(next) {
			r.held[action] = now.Add(repeatEvery)
			out = append(out, action)
		}
	}
	return out
}

// DeviceKind diz se o controle é "xbox" ou "ps", para a interface mostrar os botões certos.
func DeviceKind(name string) string {
	n := strings.ToLower(name)
	for _, p := range []string{"dualsense", "dualshock", "playstation", "wireless controller", "054c"} {
		if strings.Contains(n, p) {
			return "ps"
		}
	}
	return "xbox"
}

type EventKind int

const (
	ButtonPressed EventKind = iota
	ButtonReleased
	AxisChanged
	Disconnected
)

type Event struct {
	Kind   EventKind
	Button Button
	Axis   Axis
	Value  float32
	Device string
}

// Source entrega os eventos novos do controle, sem bloquear.
type Source interface{ Poll() []Event }

// Output é o que a interface recebe.
type Output struct {
	Acao        string `json:"acao"`
	Dispositivo string `json:"dispositivo"`
}

// Loop processa os eventos de src até stop fechar. emit só é chamado quando ready() é true:
// um controle usado dentro de um jogo nunca deve clicar em "Remover" no launcher escondido.
func Loop(src Source, ready func() bool, emit func(Output), now func() time.Time, stop <-chan struct{}, every time.Duration) {
	var rep Repeater
	analog := map[Axis]string{}
	device := "xbox"
	send := func(action string) {
		if ready() {
			emit(Output{Acao: action, Dispositivo: device})
		}
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		for _, ev := range src.Poll() {
			if ev.Device != "" {
				device = ev.Device
			}
			t := now()
			switch ev.Kind {
			case ButtonPressed:
				if a, ok := Action(ev.Button); ok && rep.Press(a, t) {
					send(a)
				}
			case ButtonReleased:
				if a, ok := Action(ev.Button); ok {
					rep.Release(a)
				}
			case AxisChanged:
				next, has := Direction(ev.Axis, ev.Value)
				old, hadOld := analog[ev.Axis]
				if has == hadOld && next == old {
					continue
				}
				if hadOld {
					rep.Release(old)
					delete(analog, ev.Axis)
				}
				if has {
					analog[ev.Axis] = next
					if rep.Press(next, t) {
						send(next)
					}
				}
			case Disconnected:
				rep = Repeater{}
				analog = map[Axis]string{}
			}
		}
		for _, a := range rep.Repeats(now()) {
			send(a)
		}
		select {
		case <-stop:
			return
		case <-ticker.C:
		}
	}
}
