package proctree

import (
	"bytes"
	"context"
	"testing"
	"time"
)

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
