package proctree

import (
	"bytes"
	"strings"
	"testing"
)

func TestTeeLineWritesAndCallbacks(t *testing.T) {
	var buf bytes.Buffer
	var lines []string
	handler := TeeLine(&buf, func(line string) { lines = append(lines, line) })
	handler("hello")
	handler("world")
	if strings.TrimSpace(buf.String()) != "hello\nworld" {
		t.Fatalf("buf=%q", buf.String())
	}
	if len(lines) != 2 || lines[0] != "hello" || lines[1] != "world" {
		t.Fatalf("lines=%v", lines)
	}
}

func TestTeeOptionsSeparateStreams(t *testing.T) {
	t.Run("stdout", func(t *testing.T) {
		var out bytes.Buffer
		var errBuf bytes.Buffer
		var stdoutLines, stderrLines []string
		opts := TeeOptions(&out, &errBuf,
			func(line string) { stdoutLines = append(stdoutLines, line) },
			func(line string) { stderrLines = append(stderrLines, line) },
		)
		opts.OnStdout("a")
		opts.OnStderr("b")
		if out.String() != "a\n" {
			t.Fatalf("stdout=%q", out.String())
		}
		if errBuf.String() != "b\n" {
			t.Fatalf("stderr=%q", errBuf.String())
		}
		if len(stdoutLines) != 1 || len(stderrLines) != 1 {
			t.Fatalf("stdout=%v stderr=%v", stdoutLines, stderrLines)
		}
	})
}
