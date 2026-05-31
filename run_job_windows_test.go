//go:build windows

package proctree

import (
	"context"
	"testing"
	"time"
)

func TestRunAttachesWindowsJob(t *testing.T) {
	ctx := context.Background()
	spec := longRunningSpec()
	done := make(chan int, 1)
	go func() {
		runSpec := spec
		res, err := Run(ctx, &runSpec, &Options{OnStart: func(pid int) { done <- pid }})
		if err != nil && !res.Canceled {
			done <- 0
		}
	}()
	var pid int
	select {
	case pid = <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("process did not start")
	}
	if pid <= 0 {
		t.Fatal("missing pid")
	}
	t.Cleanup(func() { _ = KillTreeByPID(pid) })
	if _, ok := pidJobs.Load(pid); !ok {
		t.Fatal("expected windows job to be tracked for run pid")
	}
}
