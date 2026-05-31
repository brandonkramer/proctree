//go:build windows

package proctree

import (
	"context"
	"fmt"
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

func verifyProcessGroup(_ int) bool {
	// Windows lacks a cheap pgid==pid invariant; rely on cmdline + create time.
	return true
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
	defer syscall.CloseHandle(handle)
	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == stillActive
}

const (
	processQueryLimitedInformation = 0x1000
	stillActive                    = 259
)

// KillTreeByPID terminates pid and all known descendants.
func KillTreeByPID(pid int) error {
	if pid <= 0 {
		return nil
	}
	if killWindowsJob(pid) {
		waitProcessExit(pid, 2*time.Second)
		return nil
	}
	pids, err := Descendants(pid)
	if err != nil {
		_ = terminateProcess(pid)
	} else {
		for i := len(pids) - 1; i >= 0; i-- {
			_ = terminateProcess(pids[i])
		}
	}
	if waitProcessExit(pid, 250*time.Millisecond) {
		return nil
	}
	_ = taskKillTree(pid)
	waitProcessExit(pid, 2*time.Second)
	return nil
}

func waitProcessExit(pid int, timeout time.Duration) bool {
	return WaitNotAlive(pid, timeout)
}

func taskKillTree(pid int) error {
	cmd := exec.Command("taskkill", "/PID", fmt.Sprint(pid), "/T", "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return nil
		}
		return err
	}
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
