package proctree

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunExecStreams(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cases := []struct {
		name   string
		spec   Spec
		stream Stream
		want   string
	}{
		{
			name: "stdout",
			spec: func() Spec {
				if runtime.GOOS == "windows" {
					return Spec{Shell: "echo hello"}
				}
				return Spec{Path: "/bin/echo", Args: []string{"hello"}}
			}(),
			stream: Stdout,
			want:   "hello",
		},
		{
			name:   "stderr",
			spec:   Spec{Shell: "echo err-msg 1>&2"},
			stream: Stderr,
			want:   "err-msg",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var lines []string
			opts := &Options{}
			switch tc.stream {
			case Stdout:
				opts.OnStdout = func(line string) { lines = append(lines, line) }
			case Stderr:
				opts.OnStderr = func(line string) { lines = append(lines, line) }
			}
			res, err := Run(ctx, &tc.spec, opts)
			if err != nil {
				t.Fatal(err)
			}
			if res.ExitCode != 0 {
				t.Fatalf("exit=%d", res.ExitCode)
			}
			if len(lines) != 1 || !strings.Contains(lines[0], tc.want) {
				t.Fatalf("lines=%v want %q", lines, tc.want)
			}
		})
	}
}

func TestRunContextCancelKillsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep-based cancel test skipped on windows")
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
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
	waitUntilNotAlive(t, pid, 2*time.Second)
}

func TestRunTimeoutKillsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("slow sleep-based timeout test skipped on windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	t.Cleanup(cancel)
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
	sleepSpec := Spec{Shell: "sleep 300"}
	cmd := startSpec(t, &sleepSpec)
	time.Sleep(200 * time.Millisecond)
	if err := KillTreeByPID(cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}
	waitUntilNotAlive(t, cmd.Process.Pid, 2*time.Second)
}

func TestVerifyOwnedShellRun(t *testing.T) {
	spec := Spec{Shell: "sleep 300"}
	cmd := startSpec(t, &spec)
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
