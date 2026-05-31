package proctree

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Stream identifies stdout or stderr output.
type Stream int

const (
	Stdout Stream = iota + 1
	Stderr
)

// Spec describes a command to run. Prefer Path/Args over Shell when possible.
type Spec struct {
	// Shell runs command through the platform shell (sh -c / cmd /C).
	Shell string
	// Path is the executable for argv mode. When set, Shell is ignored.
	Path string
	Args []string
	Dir  string
	// Env entries are appended to os.Environ() when non-empty.
	Env []string
}

// Options configure streaming and lifecycle hooks for Run.
type Options struct {
	OnStdout func(line string)
	OnStderr func(line string)
	// OnLine receives every stdout/stderr line when set.
	OnLine func(stream Stream, line string)
	// OnStart is invoked with the child pid immediately after Start succeeds.
	OnStart func(pid int)
	// Stdin is attached to the child process stdin when non-nil.
	Stdin io.Reader
	// Stdout receives each stdout line (with newline) when non-nil.
	Stdout io.Writer
	// Stderr receives each stderr line (with newline) when non-nil.
	Stderr io.Writer
}

// Result is the outcome of Run after the process exits or is killed.
type Result struct {
	PID       int
	StartedAt time.Time
	ExitCode  int
	// Canceled is true when ctx ended before a natural exit (cancel or deadline).
	Canceled bool
	// TimedOut is true when ctx ended due to deadline exceeded.
	TimedOut bool
}

// NewCommand builds a not-yet-started exec.Cmd with an isolated process group.
func NewCommand(spec *Spec) *exec.Cmd {
	return NewCommandContext(context.Background(), spec)
}

// NewCommandContext is like NewCommand but binds the command to ctx.
func NewCommandContext(ctx context.Context, spec *Spec) *exec.Cmd {
	var cmd *exec.Cmd
	if spec.Path != "" {
		cmd = execPathCommand(ctx, spec.Path, spec.Args)
	} else {
		cmd = newShellCommand(ctx, spec.Shell)
	}
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	if len(spec.Env) > 0 {
		cmd.Env = append(os.Environ(), spec.Env...)
	}
	setProcessGroup(cmd)
	return cmd
}

// Start starts cmd. On Windows, assigns a kill-on-close job object when possible
// so later tree kills can use TerminateJobObject.
func Start(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = attachRunJob(cmd)
	return nil
}

// KillTree terminates the command process tree.
func KillTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = KillTreeByPID(cmd.Process.Pid)
}

func execPathCommand(ctx context.Context, path string, args []string) *exec.Cmd {
	path = strings.TrimSpace(path)
	if !filepath.IsAbs(path) {
		if resolved, err := exec.LookPath(path); err == nil {
			path = resolved
		}
	}
	return exec.CommandContext(ctx, path, args...)
}

func validateSpec(spec *Spec) error {
	if spec.Path != "" {
		if strings.Contains(spec.Path, "\x00") {
			return fmt.Errorf("proctree: invalid path")
		}
		for _, arg := range spec.Args {
			if strings.Contains(arg, "\x00") {
				return fmt.Errorf("proctree: invalid argument")
			}
		}
	}
	return nil
}

// emitLine invokes the configured output handlers for one line.
func (o *Options) emitLine(stream Stream, line string) {
	if o.OnLine != nil {
		o.OnLine(stream, line)
	}
	switch stream {
	case Stdout:
		if o.OnStdout != nil {
			o.OnStdout(line)
		}
	case Stderr:
		if o.OnStderr != nil {
			o.OnStderr(line)
		}
	}
}

func (o *Options) sink(stream Stream) io.Writer {
	switch stream {
	case Stdout:
		return o.Stdout
	case Stderr:
		return o.Stderr
	default:
		return nil
	}
}

// classifyContext maps ctx.Err() to Result flags.
func classifyContext(ctx context.Context) (canceled, timedOut bool) {
	if ctx.Err() == nil {
		return false, false
	}
	if ctx.Err() == context.DeadlineExceeded {
		return true, true
	}
	return true, false
}
