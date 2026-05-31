package proctree

import (
	"runtime"
	"testing"
	"time"
)

func TestInspectRunningProcess(t *testing.T) {
	spec := Spec{Shell: "sleep 300"}
	cmd := NewCommand(&spec)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = KillTreeByPID(cmd.Process.Pid) }()
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

func TestVerifyOwnershipWithCreateTime(t *testing.T) {
	spec := Spec{Shell: "sleep 300"}
	started := time.Now()
	cmd := NewCommand(&spec)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = KillTreeByPID(cmd.Process.Pid) }()
	time.Sleep(200 * time.Millisecond)

	own := Ownership{Spec: spec, StartedAt: started}
	if !VerifyOwnership(cmd.Process.Pid, &own) {
		t.Fatal("expected ownership match")
	}
	own.StartedAt = started.Add(-time.Hour)
	if VerifyOwnership(cmd.Process.Pid, &own) {
		t.Fatal("expected stale start rejection")
	}
}

func TestChildrenAndDescendants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-oriented children test")
	}
	cmd := NewCommand(&Spec{Shell: "sleep 300 & sleep 300 & wait"})
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = KillTreeByPID(cmd.Process.Pid) }()
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
