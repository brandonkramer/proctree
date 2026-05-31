package proctree

import (
	"errors"
	"runtime"
	"testing"
	"time"
)

func TestInspectInvalidPID(t *testing.T) {
	for _, pid := range []int{0, -1} {
		_, err := Inspect(pid)
		if !errors.Is(err, ErrProcessNotFound) {
			t.Fatalf("pid=%d err=%v", pid, err)
		}
	}
}

func TestChildrenAndDescendantsInvalidPID(t *testing.T) {
	if _, err := Children(0); !errors.Is(err, ErrProcessNotFound) {
		t.Fatalf("children err=%v", err)
	}
	if _, err := Descendants(-1); !errors.Is(err, ErrProcessNotFound) {
		t.Fatalf("descendants err=%v", err)
	}
}

func TestInspectDeadProcess(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()
	pid := cmd.Process.Pid
	if err := KillTreeByPID(pid); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, err := Inspect(pid)
	if !errors.Is(err, ErrProcessNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestCmdlineAndCreateTimeRunningProcess(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()

	parts, err := Cmdline(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) == 0 {
		t.Fatal("expected cmdline parts")
	}

	created, err := CreateTime(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if created.IsZero() {
		t.Fatal("expected create time")
	}
}

func TestCmdlineUnavailableForInvalidPID(t *testing.T) {
	_, err := Cmdline(0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInspectTreeIncludesRoot(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()

	tree, err := InspectTree(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) < 1 {
		t.Fatal("expected at least root snapshot")
	}
	found := false
	for _, info := range tree {
		if info.PID == cmd.Process.Pid {
			found = true
			if len(info.Cmdline) == 0 {
				t.Fatal("expected cmdline on root snapshot")
			}
		}
	}
	if !found {
		t.Fatalf("root pid %d missing from tree %v", cmd.Process.Pid, tree)
	}
}

func TestInspectTreeWithChildren(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-oriented shell background job tree")
	}
	spec := Spec{Shell: "sleep 300 & sleep 300 & wait"}
	cmd := NewCommand(&spec)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = KillTreeByPID(cmd.Process.Pid) }()
	time.Sleep(300 * time.Millisecond)

	tree, err := InspectTree(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) < 2 {
		t.Fatalf("expected root plus children, got %d nodes", len(tree))
	}
}

func TestVerifyOwnedNilSpec(t *testing.T) {
	if VerifyOwned(1234, nil) {
		t.Fatal("expected false for nil spec")
	}
}

func TestVerifyOwnershipNilOwnership(t *testing.T) {
	if VerifyOwnership(1234, nil) {
		t.Fatal("expected false for nil ownership")
	}
}

func TestVerifyOwnershipDeadPID(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()
	spec := longRunningSpec()
	own := Ownership{Spec: spec, StartedAt: time.Now()}
	pid := cmd.Process.Pid
	if err := KillTreeByPID(pid); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if VerifyOwnership(pid, &own) {
		t.Fatal("expected dead pid to fail verification")
	}
}
