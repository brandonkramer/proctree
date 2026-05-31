package proctree

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"testing"
)

func TestRunNonZeroExit(t *testing.T) {
	ctx := context.Background()
	spec := exitSpec(1)
	res, err := Run(ctx, &spec, nil)
	if err == nil {
		t.Fatal("expected non-zero exit error")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if res.ExitCode != 1 {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	if res.Canceled || res.TimedOut {
		t.Fatalf("result=%+v", res)
	}
}

func TestRunZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/true on unix")
	}
	ctx := context.Background()
	spec := exitSpec(0)
	res, err := Run(ctx, &spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || res.Canceled || res.TimedOut {
		t.Fatalf("result=%+v", res)
	}
}

func TestKillTreeNoProcess(t *testing.T) {
	KillTree(nil)
	KillTree(&exec.Cmd{})
}
