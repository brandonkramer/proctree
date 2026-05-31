package proctree

import (
	"testing"
	"time"
)

func TestReconcileNotRunning(t *testing.T) {
	spec := longRunningSpec()
	own := Ownership{Spec: spec, StartedAt: time.Now()}
	got := ReconcilePID(999_999_999, &own)
	if got.Outcome != ReconcileNotRunning {
		t.Fatalf("outcome=%v", got.Outcome)
	}
}

func TestReconcileUnverifiedLeavesProcessAlive(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()
	own := Ownership{
		Spec:      Spec{Shell: "echo not-this-process"},
		StartedAt: time.Now(),
	}
	got := ReconcilePID(cmd.Process.Pid, &own)
	if got.Outcome != ReconcileUnverified {
		t.Fatalf("outcome=%v", got.Outcome)
	}
	if !Alive(cmd.Process.Pid) {
		t.Fatal("unverified reconcile must not kill process")
	}
}

func TestReconcileKillsVerifiedProcess(t *testing.T) {
	spec := longRunningSpec()
	cmd := NewCommand(&spec)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	own := Ownership{Spec: spec, StartedAt: time.Now().Add(-time.Second)}
	time.Sleep(100 * time.Millisecond)

	got := ReconcilePID(cmd.Process.Pid, &own)
	if got.Outcome != ReconcileKilled {
		t.Fatalf("outcome=%v", got.Outcome)
	}
	if !WaitNotAlive(cmd.Process.Pid, 2*time.Second) {
		t.Fatal("expected process to be dead after reconcile kill")
	}
}

func TestReconcileZeroPID(t *testing.T) {
	got := ReconcilePID(0, &Ownership{Spec: longRunningSpec()})
	if got.Outcome != ReconcileNotRunning {
		t.Fatalf("outcome=%v", got.Outcome)
	}
}

func TestReconcileUnverifiedNilOwnership(t *testing.T) {
	cmd, cleanup := startLongRunning(t)
	defer cleanup()
	got := ReconcilePID(cmd.Process.Pid, nil)
	if got.Outcome != ReconcileUnverified {
		t.Fatalf("outcome=%v", got.Outcome)
	}
	if !Alive(cmd.Process.Pid) {
		t.Fatal("nil ownership must not kill live process")
	}
}
