//go:build windows

package core

import (
	"os/exec"
	"syscall"
)

func prepareProcess(cmd *exec.Cmd) (*processJob, error) {
	job, err := newProcessJob()
	if err != nil {
		return nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 4}
	return job, nil
}
func (j *processJob) attach(cmd *exec.Cmd) error { return j.attachAndResume(cmd) }
