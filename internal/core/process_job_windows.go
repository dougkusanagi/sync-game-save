//go:build windows

package core

import (
	"errors"
	"fmt"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

type processJob struct{ handle windows.Handle }
type jobAccounting struct {
	TotalUserTime, TotalKernelTime, ThisPeriodUserTime, ThisPeriodKernelTime       int64
	TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses uint32
}

func newProcessJob() (*processJob, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("não foi possível criar o grupo do jogo: %w", err)
	}
	return &processJob{h}, nil
}
func (j *processJob) close() error { return windows.CloseHandle(j.handle) }
func (j *processJob) active() (bool, error) {
	var info jobAccounting
	err := windows.QueryInformationJobObject(j.handle, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), nil)
	if err != nil {
		return false, err
	}
	return info.ActiveProcesses > 0, nil
}
func (j *processJob) terminate() error { return windows.TerminateJobObject(j.handle, 1) }

// attachAndResume associa o processo suspenso antes que ele possa abrir descendentes.
// Em caso de falha, encerra e recolhe o processo para não deixá-lo suspenso.
func (j *processJob) attachAndResume(cmd *exec.Cmd) (err error) {
	defer func() {
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	if err = windows.AssignProcessToJobObject(j.handle, h); err != nil {
		return err
	}
	return resumePrimary(uint32(cmd.Process.Pid))
}
func resumePrimary(pid uint32) error {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snap)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	err = windows.Thread32First(snap, &entry)
	for err == nil {
		if entry.OwnerProcessID == pid {
			thread, e := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
			if e != nil {
				return e
			}
			defer windows.CloseHandle(thread)
			_, e = windows.ResumeThread(thread)
			return e
		}
		err = windows.Thread32Next(snap, &entry)
	}
	if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return errors.New("thread inicial não encontrada")
	}
	return err
}
