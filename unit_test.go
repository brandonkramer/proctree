package proctree

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCmdlineMatchesSpec(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		goos  string
		spec  Spec
		parts []string
		want  bool
	}{
		{name: "unix shell match", goos: "unix", spec: Spec{Shell: "sleep 5"}, parts: []string{"sh", "-c", "sleep 5"}, want: true},
		{name: "unix shell mismatch", goos: "unix", spec: Spec{Shell: "sleep 5"}, parts: []string{"sh", "-c", "echo nope"}, want: false},
		{
			name: "windows shell match",
			goos: "windows",
			spec: Spec{Shell: "sleep 5"},
			parts: []string{
				`C:\Windows\system32\cmd.exe /C "sleep 5"`,
			},
			want: true,
		},
		{name: "exec match", spec: Spec{Path: "/bin/echo", Args: []string{"hi"}}, parts: []string{"/bin/echo", "hi"}, want: true},
		{name: "exec extra args", spec: Spec{Path: "/bin/echo", Args: []string{"hi"}}, parts: []string{"/bin/echo", "hi", "extra"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.goos == "unix" && runtime.GOOS == "windows" {
				t.Skip("unix cmdline shape")
			}
			if tc.goos == "windows" && runtime.GOOS != "windows" {
				t.Skip("windows cmdline shape")
			}
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
		goos string
		spec Spec
		line string
		want bool
	}{
		{name: "unix shell", goos: "unix", spec: Spec{Shell: "sleep 300"}, line: "/bin/sh -c sleep 300", want: true},
		{name: "windows shell", goos: "windows", spec: Spec{Shell: "sleep 300"}, line: `C:\Windows\system32\cmd.exe /C "sleep 300"`, want: true},
		{name: "bare shell", spec: Spec{Shell: "make test"}, line: "make test", want: true},
		{name: "exec line", spec: Spec{Path: "/usr/bin/git", Args: []string{"status", "--short"}}, line: "/usr/bin/git status --short", want: true},
		{name: "unix shell mismatch", goos: "unix", spec: Spec{Shell: "sleep 300"}, line: "/bin/sh -c echo nope", want: false},
		{name: "empty line", spec: Spec{Shell: "sleep 1"}, line: "   ", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.goos == "unix" && runtime.GOOS == "windows" {
				t.Skip("unix cmdline shape")
			}
			if tc.goos == "windows" && runtime.GOOS != "windows" {
				t.Skip("windows cmdline shape")
			}
			if got := commandLineMatchesSpec(tc.line, &tc.spec); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestCmdlineMatchesPartsPtrJoinedFallback(t *testing.T) {
	spec := Spec{Shell: "sleep 300"}
	if runtime.GOOS == "windows" {
		parts := []string{`C:\Windows\system32\cmd.exe /C "sleep 300"`}
		if !cmdlineMatchesPartsPtr(parts, &spec) {
			t.Fatal("expected joined fallback match")
		}
		return
	}
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

func TestValidateSpec(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		spec    Spec
		wantErr string
	}{
		{name: "clean exec", spec: Spec{Path: "/bin/echo", Args: []string{"hi"}}, wantErr: ""},
		{name: "null path", spec: Spec{Path: "echo\x00bad"}, wantErr: "invalid path"},
		{name: "null arg", spec: Spec{Path: "/bin/echo", Args: []string{"ok\x00bad"}}, wantErr: "invalid argument"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateSpec(&tc.spec)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestRunSpecValidation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		spec    *Spec
		wantErr string
	}{
		{name: "nil spec", spec: nil, wantErr: "spec is nil"},
		{name: "empty spec", spec: &Spec{}, wantErr: "requires Path or Shell"},
		{name: "invalid path", spec: &Spec{Path: "bad\x00path"}, wantErr: "invalid path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Run(context.Background(), tc.spec, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestClassifyContext(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		setup        func() context.Context
		wantCanceled bool
		wantTimedOut bool
	}{
		{
			name:         "live context",
			setup:        context.Background,
			wantCanceled: false,
			wantTimedOut: false,
		},
		{
			name: "cancel",
			setup: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantCanceled: true,
			wantTimedOut: false,
		},
		{
			name: "deadline exceeded",
			setup: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				t.Cleanup(cancel)
				return ctx
			},
			wantCanceled: true,
			wantTimedOut: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			canceled, timedOut := classifyContext(tc.setup())
			if canceled != tc.wantCanceled || timedOut != tc.wantTimedOut {
				t.Fatalf("canceled=%v timedOut=%v", canceled, timedOut)
			}
		})
	}
}

func TestOptionsEmitLine(t *testing.T) {
	t.Parallel()
	var gotStream Stream
	var gotLine string
	opts := &Options{
		OnLine: func(stream Stream, line string) {
			gotStream = stream
			gotLine = line
		},
		OnStdout: func(line string) {
			if gotLine == "" {
				gotLine = line
			}
		},
	}
	opts.emitLine(Stdout, "hello")
	if gotStream != Stdout || gotLine != "hello" {
		t.Fatalf("stream=%v line=%q", gotStream, gotLine)
	}
	opts.emitLine(Stderr, "warn")
	if gotLine != "warn" {
		t.Fatalf("stderr line=%q", gotLine)
	}
}

func TestOptionsSink(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	opts := &Options{Stdout: stdout, Stderr: stderr}
	if opts.sink(Stdout) != stdout || opts.sink(Stderr) != stderr {
		t.Fatal("unexpected sink mapping")
	}
	if opts.sink(Stream(0)) != nil {
		t.Fatal("expected nil sink for unknown stream")
	}
}
