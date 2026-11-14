package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Engine struct {
	GameDir          string
	Roots            []Root
	StateDir, Device string
	Now              uint64
}
type BeforeStatus int

const (
	Ready BeforeStatus = iota
	RestoredFromCloud
	UploadedLocal
	SyncConflict
	LockedByOther
)

type BeforeResult struct {
	Status BeforeStatus
	Device string
}
type Choice int

const (
	UseLocal Choice = iota
	UseCloud
)

func (e Engine) base() (*string, error) {
	b, err := os.ReadFile(filepath.Join(e.StateDir, "base.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var v struct {
		Digest *string `json:"digest"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v.Digest, nil
}
func (e Engine) setBase(digest string) error {
	if err := os.MkdirAll(e.StateDir, 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Digest *string `json:"digest"`
	}{&digest}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.StateDir, "base.json"), b, 0644)
}
func (e Engine) snapshotID() (string, error) {
	base := fmt.Sprintf("%d_%s", e.Now, e.Device)
	for n := 0; ; n++ {
		id := base
		if n > 0 {
			id = fmt.Sprintf("%s~sgs-%d", base, n)
		}
		_, err := os.Stat(filepath.Join(e.GameDir, "snapshots", id))
		if errors.Is(err, os.ErrNotExist) {
			return id, nil
		}
		if err != nil {
			return "", err
		}
	}
}
func (e Engine) upload(expected *Index) error {
	id, err := e.snapshotID()
	if err != nil {
		return err
	}
	head, err := CreateSnapshot(e.GameDir, e.Roots, id, e.Device, expected)
	if err != nil {
		return err
	}
	return e.setBase(head.IndexDigest)
}
func (e Engine) restore(head *Head, reason string) error {
	if head == nil || !safeKey(head.Snapshot) {
		return errors.New("versão da nuvem inválida")
	}
	if err := RestoreSnapshot(filepath.Join(e.GameDir, "snapshots", head.Snapshot), e.Roots, filepath.Join(e.StateDir, "backups", fmt.Sprintf("%d_%s", e.Now, reason))); err != nil {
		return err
	}
	return e.setBase(head.IndexDigest)
}
func (e Engine) BeforeLaunch(force bool) (BeforeResult, error) {
	status, lock, err := LockState(e.GameDir, e.Device, e.Now)
	if err != nil {
		return BeforeResult{}, err
	}
	if status == LockHeldByOther && !force {
		return BeforeResult{LockedByOther, lock.Device}, nil
	}
	head, err := ReadHead(e.GameDir)
	if err != nil {
		return BeforeResult{}, err
	}
	local, err := ScanRoots(e.Roots)
	if err != nil {
		return BeforeResult{}, err
	}
	base, err := e.base()
	if err != nil {
		return BeforeResult{}, err
	}
	var cloud *string
	if head != nil {
		cloud = &head.IndexDigest
	}
	result := BeforeResult{Status: Ready}
	switch Decide(base, local, cloud) {
	case Nothing:
		if cloud != nil {
			if err := e.setBase(*cloud); err != nil {
				return BeforeResult{}, err
			}
		}
	case RestoreFromCloud:
		if err := e.restore(head, "antes-de-restaurar"); err != nil {
			return BeforeResult{}, err
		}
		result.Status = RestoredFromCloud
	case UploadLocal:
		if len(local.Files) > 0 {
			if err := e.upload(nil); err != nil {
				return BeforeResult{}, err
			}
			result.Status = UploadedLocal
		}
	case Conflict:
		return BeforeResult{Status: SyncConflict}, nil
	}
	if err := AcquireLock(e.GameDir, e.Device, e.Now); err != nil {
		return BeforeResult{}, err
	}
	return result, nil
}
func (e Engine) Resolve(choice Choice) error {
	switch choice {
	case UseLocal:
		if err := e.upload(nil); err != nil {
			return err
		}
	case UseCloud:
		head, err := ReadHead(e.GameDir)
		if err != nil {
			return err
		}
		if err := e.restore(head, "conflito"); err != nil {
			return err
		}
	default:
		return errors.New("escolha de conflito inválida")
	}
	return AcquireLock(e.GameDir, e.Device, e.Now)
}
func (e Engine) AfterExit(beat func()) (bool, error) { return e.afterExit(beat, time.Sleep) }
func (e Engine) afterExit(beat func(), sleep func(time.Duration)) (bool, error) {
	wait := func() {
		sleep(time.Second)
		if beat != nil {
			beat()
		}
	}
	for n := 0; n < 3; n++ {
		wait()
	}
	local, err := ScanRoots(e.Roots)
	if err != nil {
		return false, err
	}
	stable := false
	for n := 0; n < 60; n++ {
		wait()
		next, err := ScanRoots(e.Roots)
		if err != nil {
			return false, err
		}
		if local.Digest() == next.Digest() {
			local = next
			stable = true
			break
		}
		local = next
	}
	if !stable {
		return false, errors.New("os arquivos de save continuam mudando; feche o jogo e tente sincronizar novamente")
	}
	base, err := e.base()
	if err != nil {
		return false, err
	}
	digest := local.Digest()
	created := false
	if (base == nil || *base != digest) && len(local.Files) > 0 {
		if err := e.upload(&local); err != nil {
			return false, err
		}
		created = true
	}
	if err := ReleaseLock(e.GameDir, e.Device); err != nil {
		return false, err
	}
	return created, nil
}
