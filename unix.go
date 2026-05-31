//go:build unix

package proctree

import (
	"context"
	"os/exec"
	"syscall"
)

func newShellCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "sh", "-c", command)
}

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func verifyProcessGroup(pid int) bool {
	pgid, err := syscall.Getpgid(pid)
	if err != nil || pgid != pid {
		return false
	}
	return true
}
