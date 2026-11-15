package core

import "errors"

// FirstBackupStatus é o resultado do primeiro backup de um jogo recém-adicionado.
type FirstBackupStatus int

const (
	// FirstBackupCreated: o save deste PC virou a primeira versão na pasta de sync.
	FirstBackupCreated FirstBackupStatus = iota
	// FirstBackupAlreadySynced: a pasta de sync já tem exatamente este save.
	FirstBackupAlreadySynced
	// FirstBackupNothing: não há arquivos de save ainda; nada foi feito.
	FirstBackupNothing
	// FirstBackupNeedsDecision: já existe uma versão diferente na pasta de sync; o usuário
	// decide ao abrir o jogo.
	FirstBackupNeedsDecision
	// FirstBackupLockedByOther: outro PC está jogando.
	FirstBackupLockedByOther
	// FirstBackupInUse: este PC tem uma trava ativa (jogo aberto ou sync em andamento).
	FirstBackupInUse
)

type FirstBackupResult struct {
	Status FirstBackupStatus
	// Device é o PC que segura a trava, quando Status é FirstBackupLockedByOther.
	Device string
}

// FirstBackup faz o primeiro backup sem abrir o jogo. Só cria a primeira versão: nunca
// restaura, nunca troca uma versão existente e recusa se houver trava. A trava é pega só
// durante a cópia e solta ao fim, mesmo em erro.
func (e Engine) FirstBackup() (FirstBackupResult, error) {
	status, lock, err := LockState(e.GameDir, e.Device, e.Now)
	if err != nil {
		return FirstBackupResult{}, err
	}
	switch status {
	case LockHeldByOther:
		return FirstBackupResult{Status: FirstBackupLockedByOther, Device: lock.Device}, nil
	case LockOwn:
		return FirstBackupResult{Status: FirstBackupInUse}, nil
	}
	head, err := ReadHead(e.GameDir)
	if err != nil {
		return FirstBackupResult{}, err
	}
	local, err := ScanRoots(e.Roots)
	if err != nil {
		return FirstBackupResult{}, err
	}
	if len(local.Files) == 0 {
		return FirstBackupResult{Status: FirstBackupNothing}, nil
	}
	var cloud *string
	if head != nil {
		cloud = &head.IndexDigest
	}
	base, err := e.base()
	if err != nil {
		return FirstBackupResult{}, err
	}
	switch Decide(base, local, cloud) {
	case Nothing:
		if cloud != nil {
			if err := e.setBase(*cloud); err != nil {
				return FirstBackupResult{}, err
			}
		}
		return FirstBackupResult{Status: FirstBackupAlreadySynced}, nil
	case UploadLocal:
		if err := AcquireLock(e.GameDir, e.Device, e.Now); err != nil {
			return FirstBackupResult{}, err
		}
		sent := e.upload(nil)
		released := ReleaseLock(e.GameDir, e.Device)
		if err := errors.Join(sent, released); err != nil {
			return FirstBackupResult{}, err
		}
		return FirstBackupResult{Status: FirstBackupCreated}, nil
	}
	return FirstBackupResult{Status: FirstBackupNeedsDecision}, nil
}
