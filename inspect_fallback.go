//go:build !linux && !darwin && !windows

package proctree

import (
	"fmt"
	"time"
)

func readProcessInfo(pid int) (ProcessInfo, error) {
	if !Alive(pid) {
		return ProcessInfo{}, ErrProcessNotFound
	}
	return ProcessInfo{PID: pid, Status: "unknown"}, nil
}

func readCmdlineParts(pid int) ([]string, error) {
	return nil, fmt.Errorf("cmdline unavailable on this GOOS")
}

func readCreateTime(pid int) (time.Time, error) {
	return time.Time{}, fmt.Errorf("create time unavailable on this GOOS")
}

func listChildPIDs(pid int) ([]int, error) {
	return nil, fmt.Errorf("children unavailable on this GOOS")
}

func shellPayloadFromParts([]string) string { return "" }

func shellPayloadFromCommandLine(string) string { return "" }
