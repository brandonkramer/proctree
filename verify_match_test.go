package proctree

import (
	"runtime"
	"testing"
)

func TestCmdlineMatchesSpecShellCommand(t *testing.T) {
	spec := Spec{Shell: "sleep 5"}
	parts := []string{"sh", "-c", "sleep 5"}
	if !cmdlineMatchesSpec(parts, &spec) {
		t.Fatal("expected shell command match")
	}
}

func TestCmdlineMatchesSpecRejectsMismatchedShellCommand(t *testing.T) {
	spec := Spec{Shell: "sleep 5"}
	parts := []string{"sh", "-c", "echo nope"}
	if cmdlineMatchesSpec(parts, &spec) {
		t.Fatal("expected mismatch")
	}
}

func TestCmdlineMatchesSpecExecPath(t *testing.T) {
	spec := Spec{Path: "/bin/echo", Args: []string{"hi"}}
	parts := []string{"/bin/echo", "hi"}
	if !cmdlineMatchesSpec(parts, &spec) {
		t.Fatal("expected exec match")
	}
}

func TestCommandLineMatchesSpecDarwinStyleShell(t *testing.T) {
	spec := Spec{Shell: "sleep 300"}
	line := "/bin/sh -c sleep 300"
	if !commandLineMatchesSpec(line, &spec) {
		t.Fatal("expected ps-style shell match")
	}
}

func TestCommandLineMatchesSpecRejectsMismatchedShell(t *testing.T) {
	spec := Spec{Shell: "sleep 300"}
	if commandLineMatchesSpec("/bin/sh -c echo nope", &spec) {
		t.Fatal("expected mismatch")
	}
}

func TestShellPayloadFromCommandLineWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows shell payload")
	}
	line := `C:\Windows\system32\cmd.exe /C "sleep 300"`
	if got := shellPayloadFromCommandLine(line); got != "sleep 300" {
		t.Fatalf("payload=%q", got)
	}
}
