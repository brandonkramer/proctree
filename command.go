package proctree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
