package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const FreshSeconds uint64 = 600

type Lock struct {
	Device    string `json:"device"`
	Started   uint64 `json:"started"`
	Heartbeat uint64 `json:"heartbeat"`
}
type LockStatus int

const (
	LockFree LockStatus = iota
	LockOwn
	LockHeldByOther
	LockStale
)

func ReadLock(dir string) (*Lock, error) {
	b, err := os.ReadFile(filepath.Join(dir, "lock.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var l struct {
		Device    *string `json:"device"`
		Started   *uint64 `json:"started"`
		Heartbeat *uint64 `json:"heartbeat"`
	}
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	if l.Device == nil || l.Started == nil || l.Heartbeat == nil {
		return nil, errors.New("trava inválida")
	}
	return &Lock{*l.Device, *l.Started, *l.Heartbeat}, nil
}
func LockState(dir, device string, now uint64) (LockStatus, *Lock, error) {
	l, err := ReadLock(dir)
	if err != nil {
		return LockFree, nil, err
	}
	if l == nil {
		return LockFree, nil, nil
	}
	if l.Device == device {
		return LockOwn, l, nil
	}
	age := uint64(0)
	if now > l.Heartbeat {
		age = now - l.Heartbeat
	}
	if age < FreshSeconds {
		return LockHeldByOther, l, nil
	}
	return LockStale, l, nil
}
func writeLock(dir string, l Lock) error {
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "lock.json.tmp")
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "lock.json"))
}
func AcquireLock(dir, device string, now uint64) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeLock(dir, Lock{device, now, now})
}
func HeartbeatLock(dir, device string, now uint64) error {
	l, err := ReadLock(dir)
	if err != nil {
		return err
	}
	if l != nil && l.Device == device {
		l.Heartbeat = now
		return writeLock(dir, *l)
	}
	return nil
}
func ReleaseLock(dir, device string) error {
	l, err := ReadLock(dir)
	if err != nil {
		return err
	}
	if l != nil && l.Device == device {
		return os.Remove(filepath.Join(dir, "lock.json"))
	}
	return nil
}
