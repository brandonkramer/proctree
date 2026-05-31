package proctree

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Run starts spec, streams stdout/stderr, waits for completion, and kills the
// process tree when ctx is canceled or its deadline passes.
func Run(ctx context.Context, spec *Spec, opts *Options) (Result, error) {
	if spec == nil {
		return Result{}, fmt.Errorf("proctree: spec is nil")
	}
	if spec.Path == "" && spec.Shell == "" {
		return Result{}, fmt.Errorf("proctree: spec requires Path or Shell")
	}
	if err := validateSpec(spec); err != nil {
		return Result{}, err
	}
	if opts == nil {
		opts = &Options{}
	}
	cmd := NewCommandContext(ctx, spec)
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := Start(cmd); err != nil {
		return Result{}, err
	}
	defer releaseRunJob(cmd.Process.Pid)
	res := Result{PID: cmd.Process.Pid, StartedAt: time.Now()}
	if opts.OnStart != nil {
		opts.OnStart(res.PID)
	}
	stopKill := context.AfterFunc(ctx, func() { KillTree(cmd) })
	defer stopKill()

	var wg sync.WaitGroup
	stream := func(stream Stream, rc io.ReadCloser) {
		defer wg.Done()
		defer func() { _ = rc.Close() }()
		sc := bufio.NewScanner(rc)
		for sc.Scan() {
			line := sc.Text()
			opts.emitLine(stream, line)
			if w := opts.sink(stream); w != nil {
				_, _ = io.WriteString(w, line)
				_, _ = io.WriteString(w, "\n")
			}
		}
	}
	wg.Add(2)
	go stream(Stdout, stdoutPipe)
	go stream(Stderr, stderrPipe)
	wg.Wait()

	waitErr := cmd.Wait()
	res.Canceled, res.TimedOut = classifyContext(ctx)
	switch {
	case res.Canceled:
		return res, ctx.Err()
	case waitErr != nil:
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
			return res, waitErr
		}
		return res, waitErr
	default:
		res.ExitCode = 0
		return res, nil
	}
}
