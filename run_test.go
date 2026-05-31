package proctree

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunExecStreamsStdout(t *testing.T) {
	ctx := context.Background()
	var lines []string
	spec := Spec{Path: "/bin/echo", Args: []string{"hello"}}
	if runtime.GOOS == "windows" {
		spec = Spec{Shell: "echo hello"}
	}
	res, err := Run(ctx, &spec, &Options{OnStdout: func(line string) { lines = append(lines, line) }})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	if len(lines) != 1 || lines[0] != "hello" {
		t.Fatalf("stdout=%v", lines)
	}
}

func TestRunExecStreamsStderr(t *testing.T) {
	ctx := context.Background()
	var lines []string
	spec := Spec{Shell: "echo err-msg 1>&2"}
	if runtime.GOOS == "windows" {
		spec = Spec{Shell: "echo err-msg 1>&2"}
	}
	_, err := Run(ctx, &spec, &Options{OnStderr: func(line string) { lines = append(lines, line) }})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "err-msg") {
		t.Fatalf("stderr=%v", lines)
	}
}

func TestRunContextCancelKillsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep-based cancel test skipped on windows")
	}
	ctx, cancel := context.WithCancel(context.Background())
	spec := Spec{Shell: "sleep 300"}
	pidCh := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		_, _ = Run(ctx, &spec, &Options{OnStart: func(pid int) { pidCh <- pid }})
		close(done)
	}()
	var pid int
	select {
	case pid = <-pidCh:
	case <-time.After(time.Second):
		t.Fatal("process did not start")
	}
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = KillTreeByPID(pid)
		t.Fatal("run did not finish after cancel")
	}
	time.Sleep(100 * time.Millisecond)
	if Alive(pid) {
		_ = KillTreeByPID(pid)
		t.Fatalf("pid %d still alive after cancel", pid)
	}
}

func TestRunTimeoutKillsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("slow sleep-based timeout test skipped on windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	res, err := Run(ctx, &Spec{Shell: "sleep 300"}, &Options{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !res.TimedOut || !res.Canceled {
		t.Fatalf("result=%+v err=%v", res, err)
	}
}

func TestKillTreeByPIDShellRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-oriented kill test")
	}
	cmd := NewCommand(&Spec{Shell: "sleep 300"})
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := KillTreeByPID(cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}
	if Alive(cmd.Process.Pid) {
		t.Fatalf("pid %d still alive after group kill", cmd.Process.Pid)
	}
}

func TestVerifyOwnedShellRun(t *testing.T) {
	spec := Spec{Shell: "sleep 300"}
	cmd := NewCommand(&spec)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = KillTreeByPID(cmd.Process.Pid) }()
	time.Sleep(200 * time.Millisecond)
	if !VerifyOwned(cmd.Process.Pid, &spec) {
		t.Fatalf("verify failed for pid=%d", cmd.Process.Pid)
	}
}

func TestConcurrentRuns(t *testing.T) {
	ctx := context.Background()
	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spec := Spec{Path: "/bin/echo", Args: []string{"ok"}}
			if runtime.GOOS == "windows" {
				spec = Spec{Shell: "echo ok"}
			}
			_, err := Run(ctx, &spec, &Options{})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}
