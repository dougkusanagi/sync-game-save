package gamepad

// Decodificação dos eventos do Linux (evdev) para os botões e eixos do pacote. Fica fora do
// arquivo de leitura de dispositivos para poder ser testada em qualquer sistema, sem controle.

const (
	evKey = 0x01
	evAbs = 0x03

	btnSouth  = 0x130
	btnEast   = 0x131
	btnNorth  = 0x133
	btnWest   = 0x134
	btnTL     = 0x136
	btnTR     = 0x137
	btnTL2    = 0x138
	btnTR2    = 0x139
	btnSelect = 0x13a
	btnStart  = 0x13b
	btnMode   = 0x13c
	btnThumbL = 0x13d
	btnThumbR = 0x13e

	absX     = 0x00
	absY     = 0x01
	absZ     = 0x02 // gatilho esquerdo analógico, em alguns controles
	absRZ    = 0x05 // gatilho direito analógico
	absHat0X = 0x10
	absHat0Y = 0x11
)

var evdevButtons = map[uint16]Button{
	btnSouth: South, btnEast: East, btnNorth: North, btnWest: West,
	btnTL: LeftTrigger, btnTR: RightTrigger, btnTL2: LeftTrigger2, btnTR2: RightTrigger2,
	btnSelect: Select, btnStart: Start, btnMode: Mode, btnThumbL: LeftThumb, btnThumbR: RightThumb,
}

// absRange é o intervalo de valores de um eixo, lido do dispositivo.
type absRange struct{ min, max int32 }

// evdevDecoder guarda o estado entre eventos: o direcional chega como eixo (-1, 0, 1) e vira
// pressionar e soltar; os gatilhos analógicos viram botão com histerese.
type evdevDecoder struct {
	device     string
	ranges     map[uint16]absRange
	hatX, hatY int32
	lt, rt     bool
}

func newEvdevDecoder(device string, ranges map[uint16]absRange) *evdevDecoder {
	return &evdevDecoder{device: device, ranges: ranges}
}

func (d *evdevDecoder) press(b Button, down bool) Event {
	kind := ButtonReleased
	if down {
		kind = ButtonPressed
	}
	return Event{Kind: kind, Button: b, Device: d.device}
}

// hat converte o eixo do direcional em pressionar e soltar dos dois botões do eixo.
func (d *evdevDecoder) hat(old, now int32, minus, plus Button) []Event {
	var out []Event
	if old < 0 && now >= 0 {
		out = append(out, d.press(minus, false))
	}
	if old > 0 && now <= 0 {
		out = append(out, d.press(plus, false))
	}
	if now < 0 && old >= 0 {
		out = append(out, d.press(minus, true))
	}
	if now > 0 && old <= 0 {
		out = append(out, d.press(plus, true))
	}
	return out
}

// normalize leva o valor do eixo ao intervalo de -1 a 1.
func (d *evdevDecoder) normalize(code uint16, v int32) float32 {
	r, ok := d.ranges[code]
	if !ok || r.max <= r.min {
		return 0
	}
	mid := float32(r.min+r.max) / 2
	half := float32(r.max-r.min) / 2
	return (float32(v) - mid) / half
}

func (d *evdevDecoder) trigger(was *bool, code uint16, v int32, b Button) []Event {
	r, ok := d.ranges[code]
	if !ok || r.max <= r.min {
		return nil
	}
	frac := float32(v-r.min) / float32(r.max-r.min)
	is := *was
	if frac >= 0.5 {
		is = true
	} else if frac < 0.25 {
		is = false
	}
	if is == *was {
		return nil
	}
	*was = is
	return []Event{d.press(b, is)}
}

// Decode converte um evento do evdev nos eventos do pacote.
func (d *evdevDecoder) Decode(typ, code uint16, value int32) []Event {
	switch typ {
	case evKey:
		if b, ok := evdevButtons[code]; ok && (value == 0 || value == 1) {
			return []Event{d.press(b, value == 1)}
		}
	case evAbs:
		switch code {
		case absHat0X:
			old := d.hatX
			d.hatX = value
			return d.hat(old, value, DPadLeft, DPadRight)
		case absHat0Y:
			// No evdev, para cima é negativo.
			old := d.hatY
			d.hatY = value
			return d.hat(old, value, DPadUp, DPadDown)
		case absX:
			return []Event{{Kind: AxisChanged, Axis: LeftStickX, Value: d.normalize(code, value), Device: d.device}}
		case absY:
			// O pacote usa positivo para cima, como o XInput.
			return []Event{{Kind: AxisChanged, Axis: LeftStickY, Value: -d.normalize(code, value), Device: d.device}}
		case absZ:
			return d.trigger(&d.lt, code, value, LeftTrigger2)
		case absRZ:
			return d.trigger(&d.rt, code, value, RightTrigger2)
		}
	}
	return nil
}
