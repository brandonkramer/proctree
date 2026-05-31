//go:build !windows

package proctree

import "os/exec"

func attachRunJob(_ *exec.Cmd) error { return nil }

func releaseRunJob(_ int) {}
