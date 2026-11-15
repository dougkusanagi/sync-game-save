package gamepad

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	xinputDLL      = windows.NewLazySystemDLL("xinput1_4.dll")
	xinputGetState = xinputDLL.NewProc("XInputGetState")
)

type xinputState struct {
	PacketNumber uint32
	Buttons      uint16
	LeftTrigger  uint8
	RightTrigger uint8
	ThumbLX      int16
	ThumbLY      int16
	ThumbRX      int16
	ThumbRY      int16
}

const (
	errSuccess      = 0
	triggerPressed  = 128
	triggerReleased = 64
)

var xinputButtons = []struct {
	mask   uint16
	button Button
}{
	{0x0001, DPadUp}, {0x0002, DPadDown}, {0x0004, DPadLeft}, {0x0008, DPadRight},
	{0x0010, Start}, {0x0020, Select}, {0x0040, LeftThumb}, {0x0080, RightThumb},
	{0x0100, LeftTrigger}, {0x0200, RightTrigger},
	{0x1000, South}, {0x2000, East}, {0x4000, West}, {0x8000, North},
}

type xinputPad struct {
	connected bool
	buttons   uint16
	lt, rt    bool
	lx, ly    float32
}

// XInput lê até quatro controles Xbox (e compatíveis) pela API do Windows.
type XInput struct{ pads [4]xinputPad }

func NewXInput() (*XInput, error) {
	if err := xinputGetState.Find(); err != nil {
		return nil, err
	}
	return &XInput{}, nil
}

func norm(v int16) float32 {
	if v < 0 {
		return float32(v) / 32768
	}
	return float32(v) / 32767
}

func (x *XInput) Poll() []Event {
	var out []Event
	for i := range x.pads {
		var st xinputState
		r, _, _ := xinputGetState.Call(uintptr(i), uintptr(unsafe.Pointer(&st)))
		p := &x.pads[i]
		if r != errSuccess {
			if p.connected {
				*p = xinputPad{}
				out = append(out, Event{Kind: Disconnected})
			}
			continue
		}
		p.connected = true
		for _, b := range xinputButtons {
			was, is := p.buttons&b.mask != 0, st.Buttons&b.mask != 0
			if was != is {
				kind := ButtonReleased
				if is {
					kind = ButtonPressed
				}
				out = append(out, Event{Kind: kind, Button: b.button, Device: "xbox"})
			}
		}
		p.buttons = st.Buttons
		trigger := func(was *bool, value uint8, b Button) {
			is := *was
			if value >= triggerPressed {
				is = true
			} else if value < triggerReleased {
				is = false
			}
			if is != *was {
				kind := ButtonReleased
				if is {
					kind = ButtonPressed
				}
				out = append(out, Event{Kind: kind, Button: b, Device: "xbox"})
				*was = is
			}
		}
		trigger(&p.lt, st.LeftTrigger, LeftTrigger2)
		trigger(&p.rt, st.RightTrigger, RightTrigger2)
		if lx := norm(st.ThumbLX); lx != p.lx {
			p.lx = lx
			out = append(out, Event{Kind: AxisChanged, Axis: LeftStickX, Value: lx, Device: "xbox"})
		}
		if ly := norm(st.ThumbLY); ly != p.ly {
			p.ly = ly
			out = append(out, Event{Kind: AxisChanged, Axis: LeftStickY, Value: ly, Device: "xbox"})
		}
	}
	return out
}
