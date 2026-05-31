package proctree

import (
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func startSpec(t *testing.T, spec *Spec) *exec.Cmd {
	t.Helper()
	cmd := NewCommand(spec)
	if err := Start(cmd); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() {
		_ = KillTreeByPID(pid)
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if Alive(pid) {
			time.Sleep(50 * time.Millisecond)
			return cmd
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("process did not start")
	return nil
}

func startLongRunning(t *testing.T) *exec.Cmd {
	t.Helper()
	spec := longRunningSpec()
	return startSpec(t, &spec)
}

func longRunningSpec() Spec {
	if runtime.GOOS == "windows" {
		return Spec{Shell: "ping -n 600 127.0.0.1"}
	}
	// Keep the shell parent alive; macOS /bin/sh may exec simple `-c sleep …`.
	return Spec{Shell: "sleep 300 & wait"}
}

func exitSpec(code int) Spec {
	if runtime.GOOS == "windows" {
		return Spec{Shell: "exit /b " + strconv.Itoa(code)}
	}
	if code == 0 {
		return Spec{Shell: "true"}
	}
	return Spec{Shell: "exit " + strconv.Itoa(code)}
}

func waitUntilNotAlive(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	if !WaitNotAlive(pid, timeout) {
		t.Fatalf("pid %d still alive after %s", pid, timeout)
	}
}
