package proctree

import "io"

// TeeLine returns a line handler that invokes fn and writes line plus newline to w.
func TeeLine(w io.Writer, fn func(line string)) func(string) {
	return func(line string) {
		if fn != nil {
			fn(line)
		}
		if w != nil {
			_, _ = io.WriteString(w, line)
			_, _ = io.WriteString(w, "\n")
		}
	}
}

// TeeOptions returns Options that tee stdout/stderr to writers and callbacks.
// Either writer or callback may be nil.
func TeeOptions(stdoutW, stderrW io.Writer, onStdout, onStderr func(line string)) *Options {
	return &Options{
		OnStdout: TeeLine(stdoutW, onStdout),
		OnStderr: TeeLine(stderrW, onStderr),
	}
}
