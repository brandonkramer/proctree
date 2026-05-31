package proctree

import (
	"testing"
	"time"
)

func TestWaitNotAlive(t *testing.T) {
	t.Run("after kill", func(t *testing.T) {
		cmd := startLongRunning(t)
		pid := cmd.Process.Pid
		if err := KillTreeByPID(pid); err != nil {
			t.Fatal(err)
		}
		if !WaitNotAlive(pid, 2*time.Second) {
			t.Fatal("expected process to become not alive")
		}
	})
	t.Run("zero pid", func(t *testing.T) {
		if !WaitNotAlive(0, time.Second) {
			t.Fatal("zero pid should be treated as not alive")
		}
	})
	t.Run("missing pid", func(t *testing.T) {
		if !WaitNotAlive(999_999_999, 100*time.Millisecond) {
			t.Fatal("missing pid should return true")
		}
	})
}
