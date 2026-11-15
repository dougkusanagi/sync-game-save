package gamepad

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const rescanEvery = 2 * time.Second

// Evdev lê os controles de /dev/input/event*. Novos controles são encontrados a cada poucos
// segundos; um controle removido gera o evento Disconnected.
type Evdev struct {
	devs     map[string]*evdevDev
	lastScan time.Time
	evSize   int
}

type evdevDev struct {
	f   *os.File
	dec *evdevDecoder
	buf []byte
}

func NewEvdev() (*Evdev, error) {
	return &Evdev{devs: map[string]*evdevDev{}, evSize: int(unsafe.Sizeof(syscall.Timeval{})) + 8}, nil
}

// ioctlRead monta e executa um ioctl de leitura (_IOC_READ) do evdev.
func ioctlRead(fd uintptr, nr, size uintptr, p unsafe.Pointer) error {
	req := uintptr(2)<<30 | size<<16 | uintptr('E')<<8 | nr
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, req, uintptr(p)); errno != 0 {
		return errno
	}
	return nil
}

func hasBit(bits []byte, n int) bool { return n/8 < len(bits) && bits[n/8]&(1<<(uint(n)%8)) != 0 }

// openGamepad abre o dispositivo se ele for um controle (tem o botão A/Sul).
func openGamepad(path string) (*evdevDev, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	fd := f.Fd()
	keys := make([]byte, (0x2ff+7)/8)
	if err := ioctlRead(fd, 0x20+evKey, uintptr(len(keys)), unsafe.Pointer(&keys[0])); err != nil || !hasBit(keys, btnSouth) {
		f.Close()
		return nil, errors.New("não é um controle")
	}
	name := make([]byte, 256)
	_ = ioctlRead(fd, 0x06, uintptr(len(name)), unsafe.Pointer(&name[0]))
	n := 0
	for n < len(name) && name[n] != 0 {
		n++
	}
	ranges := map[uint16]absRange{}
	for _, code := range []uint16{absX, absY, absZ, absRZ} {
		var info [6]int32 // value, min, max, fuzz, flat, resolution
		if ioctlRead(fd, 0x40+uintptr(code), unsafe.Sizeof(info), unsafe.Pointer(&info[0])) == nil {
			ranges[code] = absRange{min: info[1], max: info[2]}
		}
	}
	return &evdevDev{f: f, dec: newEvdevDecoder(DeviceKind(string(name[:n])), ranges)}, nil
}

func (e *Evdev) scan() {
	paths, _ := filepath.Glob("/dev/input/event*")
	for _, p := range paths {
		if _, ok := e.devs[p]; ok {
			continue
		}
		if d, err := openGamepad(p); err == nil {
			d.buf = make([]byte, e.evSize*64)
			e.devs[p] = d
		}
	}
}

func (e *Evdev) Poll() []Event {
	if time.Since(e.lastScan) >= rescanEvery {
		e.lastScan = time.Now()
		e.scan()
	}
	var out []Event
	for path, d := range e.devs {
		for {
			n, err := d.f.Read(d.buf)
			if err != nil {
				if errors.Is(err, syscall.EAGAIN) || errors.Is(err, os.ErrDeadlineExceeded) {
					break
				}
				// O controle saiu (ENODEV) ou o arquivo não é mais legível.
				d.f.Close()
				delete(e.devs, path)
				out = append(out, Event{Kind: Disconnected})
				break
			}
			for off := 0; off+e.evSize <= n; off += e.evSize {
				rec := d.buf[off+e.evSize-8 : off+e.evSize]
				typ := binary.LittleEndian.Uint16(rec[0:2])
				code := binary.LittleEndian.Uint16(rec[2:4])
				val := int32(binary.LittleEndian.Uint32(rec[4:8]))
				out = append(out, d.dec.Decode(typ, code, val)...)
			}
			if n < len(d.buf) {
				break
			}
		}
	}
	return out
}
