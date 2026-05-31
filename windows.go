//go:build windows

package proctree

import (
	"context"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func newShellCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "cmd.exe", "/C", command)
}

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// KillTree terminates the command process tree.
func KillTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = KillTreeByPID(cmd.Process.Pid)
}

// Alive reports whether pid refers to a live process.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = syscall.CloseHandle(handle)
	return true
}

const processQueryLimitedInformation = 0x1000

// KillTreeByPID terminates pid and all known descendants.
func KillTreeByPID(pid int) error {
	if pid <= 0 {
		return nil
	}
	if killWindowsJob(pid) {
		WaitNotAlive(pid, 250*time.Millisecond)
		return nil
	}
	pids, err := Descendants(pid)
	if err != nil {
		return terminateProcess(pid)
	}
	for i := len(pids) - 1; i >= 0; i-- {
		_ = terminateProcess(pids[i])
	}
	WaitNotAlive(pid, 250*time.Millisecond)
	return nil
}

func terminateProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	return windows.TerminateProcess(handle, 1)
}
