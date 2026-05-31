package proctree

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"testing"
)

func TestRunExitCodes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cases := []struct {
		name    string
		code    int
		wantErr bool
		skipWin bool
	}{
		{name: "non-zero", code: 1, wantErr: true},
		{name: "zero", code: 0, wantErr: false, skipWin: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.skipWin && runtime.GOOS == "windows" {
				t.Skip("uses /bin/true on unix")
			}
			spec := exitSpec(tc.code)
			res, err := Run(ctx, &spec, nil)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected non-zero exit error")
				}
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("expected ExitError, got %T: %v", err, err)
				}
				if res.ExitCode != tc.code {
					t.Fatalf("exit=%d", res.ExitCode)
				}
				if res.Canceled || res.TimedOut {
					t.Fatalf("result=%+v", res)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if res.ExitCode != 0 || res.Canceled || res.TimedOut {
				t.Fatalf("result=%+v", res)
			}
		})
	}
}

func TestKillTreeNoProcess(t *testing.T) {
	t.Run("nil cmd", func(t *testing.T) {
		KillTree(nil)
	})
	t.Run("empty cmd", func(t *testing.T) {
		KillTree(&exec.Cmd{})
	})
}
