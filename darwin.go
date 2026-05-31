//go:build darwin

package proctree

import (
	"syscall"
	"time"
)

// Alive reports whether pid refers to a running (non-zombie) process.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	kp, err := kinfoProc(pid)
	if err != nil {
		return false
	}
	if kp.Proc.P_stat == darwinProcStateZombie {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// KillTreeByPID sends SIGKILL to the process group and leader pid.
func KillTreeByPID(pid int) error {
	if pid <= 0 {
		return nil
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
	WaitNotAlive(pid, 250*time.Millisecond)
	return nil
}
