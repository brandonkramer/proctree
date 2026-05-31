//go:build unix

package proctree

import "syscall"

func verifyProcessGroup(pid int) bool {
	pgid, err := syscall.Getpgid(pid)
	if err != nil || pgid != pid {
		return false
	}
	return true
}
