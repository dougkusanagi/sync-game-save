package gamepad

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32Raw             = windows.NewLazySystemDLL("user32.dll")
	getRawInputDeviceList = user32Raw.NewProc("GetRawInputDeviceList")
	getRawInputDeviceInfo = user32Raw.NewProc("GetRawInputDeviceInfoW")
)

const (
	rimTypeHID     = 2
	ridiDeviceName = 0x20000007
)

type rawInputDeviceList struct {
	Device uintptr
	Type   uint32
}

// tipoDosControlesConectados olha os dispositivos HID ligados ao PC e devolve "ps" se houver um
// controle da Sony. O XInput não diz o fabricante; sem isso, um DualSense seria mostrado como Xbox.
// Se houver controles dos dois tipos ao mesmo tempo, vale "ps" (não dá para saber qual está em uso).
func tipoDosControlesConectados() string {
	var n uint32
	size := uint32(unsafe.Sizeof(rawInputDeviceList{}))
	if r, _, _ := getRawInputDeviceList.Call(0, uintptr(unsafe.Pointer(&n)), uintptr(size)); r != 0 || n == 0 {
		return "xbox"
	}
	lista := make([]rawInputDeviceList, n)
	r, _, _ := getRawInputDeviceList.Call(uintptr(unsafe.Pointer(&lista[0])), uintptr(unsafe.Pointer(&n)), uintptr(size))
	if int32(r) < 0 {
		return "xbox"
	}
	for _, d := range lista[:min(int(r), len(lista))] {
		if d.Type != rimTypeHID {
			continue
		}
		var chars uint32
		if r, _, _ := getRawInputDeviceInfo.Call(d.Device, ridiDeviceName, 0, uintptr(unsafe.Pointer(&chars))); int32(r) < 0 || chars == 0 {
			continue
		}
		nome := make([]uint16, chars)
		if r, _, _ := getRawInputDeviceInfo.Call(d.Device, ridiDeviceName, uintptr(unsafe.Pointer(&nome[0])), uintptr(unsafe.Pointer(&chars))); int32(r) < 0 {
			continue
		}
		if KindFromHIDPath(windows.UTF16ToString(nome)) == "ps" {
			return "ps"
		}
	}
	return "xbox"
}
