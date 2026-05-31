package proctree

import (
	"runtime"
	"testing"
)

func TestCmdlineMatchesSpec(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		spec  Spec
		parts []string
		want  bool
	}{
		{name: "shell match", spec: Spec{Shell: "sleep 5"}, parts: []string{"sh", "-c", "sleep 5"}, want: true},
		{name: "shell mismatch", spec: Spec{Shell: "sleep 5"}, parts: []string{"sh", "-c", "echo nope"}, want: false},
		{name: "exec match", spec: Spec{Path: "/bin/echo", Args: []string{"hi"}}, parts: []string{"/bin/echo", "hi"}, want: true},
		{name: "exec extra args", spec: Spec{Path: "/bin/echo", Args: []string{"hi"}}, parts: []string{"/bin/echo", "hi", "extra"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := cmdlineMatchesSpec(tc.parts, &tc.spec); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestCommandLineMatchesSpec(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		spec Spec
		line string
		want bool
	}{
		{name: "darwin shell", spec: Spec{Shell: "sleep 300"}, line: "/bin/sh -c sleep 300", want: true},
		{name: "bare shell", spec: Spec{Shell: "make test"}, line: "make test", want: true},
		{name: "exec line", spec: Spec{Path: "/usr/bin/git", Args: []string{"status", "--short"}}, line: "/usr/bin/git status --short", want: true},
		{name: "shell mismatch", spec: Spec{Shell: "sleep 300"}, line: "/bin/sh -c echo nope", want: false},
		{name: "empty line", spec: Spec{Shell: "sleep 1"}, line: "   ", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := commandLineMatchesSpec(tc.line, &tc.spec); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestCmdlineMatchesPartsPtrJoinedFallback(t *testing.T) {
	spec := Spec{Shell: "sleep 300"}
	parts := []string{"/bin/sh -c sleep 300"}
	if !cmdlineMatchesPartsPtr(parts, &spec) {
		t.Fatal("expected joined fallback match")
	}
}

func TestIsShellExecutable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		want bool
	}{
		{name: "/bin/sh", want: true},
		{name: "/usr/local/bin/bash", want: true},
		{name: "zsh", want: true},
		{name: "cmd.exe", want: true},
		{name: "python3", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isShellExecutable(tc.name); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
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
