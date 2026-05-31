package proctree

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestClassifyContext(t *testing.T) {
	t.Parallel()
	if canceled, timedOut := classifyContext(context.Background()); canceled || timedOut {
		t.Fatal("expected no flags for live context")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceled, timedOut := classifyContext(ctx)
	if !canceled || timedOut {
		t.Fatalf("cancel: canceled=%v timedOut=%v", canceled, timedOut)
	}
	deadlineCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	canceled, timedOut = classifyContext(deadlineCtx)
	if !canceled || !timedOut {
		t.Fatalf("deadline: canceled=%v timedOut=%v", canceled, timedOut)
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
