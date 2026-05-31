//go:build linux

package proctree

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// Alive reports whether pid refers to a running (non-zombie) process.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}
	_, state, _, _, err := parseProcStat(stat)
	if err != nil || state == 'Z' {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// KillTreeByPID sends SIGKILL to the process group rooted at pid.
func KillTreeByPID(pid int) error {
	if pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		return err
	}
	WaitNotAlive(pid, 250*time.Millisecond)
	return nil
}
