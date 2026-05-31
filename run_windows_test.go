//go:build windows

package proctree

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWindowsRunEcho(t *testing.T) {
	ctx := context.Background()
	var lines []string
	res, err := Run(ctx, &Spec{Shell: "echo hello"}, &Options{
		OnStdout: func(line string) { lines = append(lines, line) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "hello") {
		t.Fatalf("stdout=%v", lines)
	}
}

func TestWindowsKillTreeByPID(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()
	pid := cmd.Process.Pid
	if err := KillTreeByPID(pid); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("pid %d still alive after kill", pid)
}

func TestWindowsInspectProcess(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()

	info, err := Inspect(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if info.PID != cmd.Process.Pid {
		t.Fatalf("pid=%d", info.PID)
	}
	if len(info.Cmdline) == 0 {
		t.Fatal("expected cmdline")
	}
	if info.CreateTime.IsZero() {
		t.Fatal("expected create time")
	}
}

func TestWindowsVerifyOwned(t *testing.T) {
	spec := longRunningSpec()
	cmd := NewCommand(&spec)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = KillTreeByPID(cmd.Process.Pid) }()
	time.Sleep(200 * time.Millisecond)
	if !VerifyOwned(cmd.Process.Pid, &spec) {
		t.Fatal("expected ownership match")
	}
}

func TestWindowsRunTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	spec := longRunningSpec()
	res, err := Run(ctx, &spec, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !res.TimedOut || !res.Canceled {
		t.Fatalf("result=%+v err=%v", res, err)
	}
}

func TestWindowsRunCancelKillsProcess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	spec := longRunningSpec()
	pidCh := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		_, _ = Run(ctx, &spec, &Options{OnStart: func(pid int) { pidCh <- pid }})
		close(done)
	}()
	var pid int
	select {
	case pid = <-pidCh:
	case <-time.After(3 * time.Second):
		t.Fatal("process did not start")
	}
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = KillTreeByPID(pid)
		t.Fatal("run did not finish after cancel")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("pid %d still alive after cancel", pid)
}

func TestWindowsInspectTree(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()
	tree, err := InspectTree(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) < 1 {
		t.Fatal("expected tree snapshots")
	}
}

func TestWindowsRunNonZeroExit(t *testing.T) {
	ctx := context.Background()
	res, err := Run(ctx, &Spec{Shell: "exit /b 2"}, nil)
	if err == nil {
		t.Fatal("expected exit error")
	}
	if res.ExitCode != 2 {
		t.Fatalf("exit=%d", res.ExitCode)
	}
}
