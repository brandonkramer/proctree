package proctree

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunWithStdinAndWriterSinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cat test is unix-oriented")
	}
	ctx := context.Background()
	var stdout bytes.Buffer
	spec := Spec{Path: "/bin/cat"}
	_, err := Run(ctx, &spec, &Options{
		Stdin:  strings.NewReader("hello\n"),
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "hello\n" {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestVerifyOwnershipCustomMatcher(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sleep verify")
	}
	spec := Spec{Shell: "sleep 300"}
	started := time.Now()
	cmd := NewCommand(&spec)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = KillTreeByPID(cmd.Process.Pid) }()
	time.Sleep(200 * time.Millisecond)

	own := Ownership{
		StartedAt: started,
		Match: func(parts []string) bool {
			joined := strings.Join(parts, " ")
			return strings.Contains(joined, "sleep")
		},
	}
	if !VerifyOwnership(cmd.Process.Pid, &own) {
		t.Fatal("expected custom matcher to match")
	}
	own.Match = func([]string) bool { return false }
	if VerifyOwnership(cmd.Process.Pid, &own) {
		t.Fatal("expected custom matcher rejection")
	}
}

func TestVerifyOwnershipOptsMatcherOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sleep verify")
	}
	spec := Spec{Shell: "sleep 300"}
	cmd := NewCommand(&spec)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = KillTreeByPID(cmd.Process.Pid) }()
	time.Sleep(200 * time.Millisecond)

	own := Ownership{Spec: spec}
	opts := VerifyOptions{
		Match: func([]string) bool { return true },
	}
	if !VerifyOwnershipOpts(cmd.Process.Pid, &own, &opts) {
		t.Fatal("expected opts matcher override")
	}
}
