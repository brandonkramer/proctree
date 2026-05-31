package proctree

import (
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func startLongRunning(t *testing.T) (cmd *exec.Cmd, cleanup func()) {
	t.Helper()
	spec := longRunningSpec()
	cmd = NewCommand(&spec)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	cleanup = func() {
		_ = KillTreeByPID(cmd.Process.Pid)
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if Alive(cmd.Process.Pid) {
			time.Sleep(50 * time.Millisecond)
			return cmd, cleanup
		}
		time.Sleep(10 * time.Millisecond)
	}
	cleanup()
	t.Fatal("process did not start")
	return nil, nil
}

func longRunningSpec() Spec {
	if runtime.GOOS == "windows" {
		return Spec{Shell: "ping -n 600 127.0.0.1 >nul"}
	}
	return Spec{Shell: "sleep 300"}
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
