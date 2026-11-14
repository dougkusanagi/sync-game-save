//go:build linux

package core

import "os/exec"

type processJob struct{}

func prepareProcess(_ *exec.Cmd) (*processJob, error) { return &processJob{}, nil }
func (*processJob) attach(_ *exec.Cmd) error          { return nil }
func (*processJob) active() (bool, error)             { return false, nil }
func (*processJob) close() error                      { return nil }
