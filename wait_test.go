package proctree

import (
	"testing"
	"time"
)

func TestWaitNotAliveAfterKill(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()
	pid := cmd.Process.Pid
	if err := KillTreeByPID(pid); err != nil {
		t.Fatal(err)
	}
	if !WaitNotAlive(pid, 2*time.Second) {
		t.Fatal("expected process to become not alive")
	}
}

func TestWaitNotAliveZeroPID(t *testing.T) {
	if !WaitNotAlive(0, time.Second) {
		t.Fatal("zero pid should be treated as not alive")
	}
}

func TestWaitNotAliveImmediateDead(t *testing.T) {
	if !WaitNotAlive(999_999_999, 100*time.Millisecond) {
		t.Fatal("missing pid should return true")
	}
}
