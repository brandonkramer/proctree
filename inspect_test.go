package proctree

import (
	"errors"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestInspectRunningProcess(t *testing.T) {
	cmd := startLongRunning(t)
	time.Sleep(200 * time.Millisecond)

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

func TestInspectInvalidPID(t *testing.T) {
	t.Parallel()
	for _, pid := range []int{0, -1} {
		t.Run(strconv.Itoa(pid), func(t *testing.T) {
			t.Parallel()
			_, err := Inspect(pid)
			if !errors.Is(err, ErrProcessNotFound) {
				t.Fatalf("pid=%d err=%v", pid, err)
			}
		})
	}
}

func TestInspectDeadProcess(t *testing.T) {
	cmd := startLongRunning(t)
	pid := cmd.Process.Pid
	if err := KillTreeByPID(pid); err != nil {
		t.Fatal(err)
	}
	waitUntilNotAlive(t, pid, 2*time.Second)

	_, err := Inspect(pid)
	if !errors.Is(err, ErrProcessNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestCmdlineAndCreateTimeRunningProcess(t *testing.T) {
	cmd := startLongRunning(t)

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
	cmd := startLongRunning(t)

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
	childSpec := Spec{Shell: "sleep 300 & sleep 300 & wait"}
	cmd := startSpec(t, &childSpec)
	time.Sleep(300 * time.Millisecond)

	tree, err := InspectTree(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) < 2 {
		t.Fatalf("expected root plus children, got %d nodes", len(tree))
	}
}

func TestChildrenAndDescendants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-oriented children test")
	}
	childSpec := Spec{Shell: "sleep 300 & sleep 300 & wait"}
	cmd := startSpec(t, &childSpec)
	time.Sleep(300 * time.Millisecond)

	kids, err := Children(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(kids) < 1 {
		t.Fatalf("expected children, got %v", kids)
	}
	desc, err := Descendants(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(desc) < len(kids)+1 {
		t.Fatalf("desc=%v kids=%v", desc, kids)
	}
}

func TestChildrenAndDescendantsInvalidPID(t *testing.T) {
	t.Parallel()
	t.Run("children", func(t *testing.T) {
		t.Parallel()
		if _, err := Children(0); !errors.Is(err, ErrProcessNotFound) {
			t.Fatalf("children err=%v", err)
		}
	})
	t.Run("descendants", func(t *testing.T) {
		t.Parallel()
		if _, err := Descendants(-1); !errors.Is(err, ErrProcessNotFound) {
			t.Fatalf("descendants err=%v", err)
		}
	})
}

func TestVerifyOwnershipWithCreateTime(t *testing.T) {
	spec := longRunningSpec()
	cmd := startSpec(t, &spec)
	time.Sleep(200 * time.Millisecond)
	created, err := CreateTime(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("match", func(t *testing.T) {
		own := Ownership{Spec: spec, StartedAt: created}
		if !VerifyOwnership(cmd.Process.Pid, &own) {
			t.Fatal("expected ownership match")
		}
	})
	t.Run("stale start", func(t *testing.T) {
		own := Ownership{Spec: spec, StartedAt: created.Add(-time.Hour)}
		if VerifyOwnership(cmd.Process.Pid, &own) {
			t.Fatal("expected stale start rejection")
		}
	})
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
	cmd := startLongRunning(t)
	spec := longRunningSpec()
	own := Ownership{Spec: spec, StartedAt: time.Now()}
	pid := cmd.Process.Pid
	if err := KillTreeByPID(pid); err != nil {
		t.Fatal(err)
	}
	waitUntilNotAlive(t, pid, 2*time.Second)
	if VerifyOwnership(pid, &own) {
		t.Fatal("expected dead pid to fail verification")
	}
}
