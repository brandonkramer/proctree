package proctree

import (
	"path/filepath"
	"strings"
)

func cmdlineMatchesSpec(parts []string, spec *Spec) bool {
	if spec.Path != "" {
		if len(parts) == 0 || parts[0] != spec.Path {
			return false
		}
		for i, arg := range spec.Args {
			if i+1 >= len(parts) || parts[i+1] != arg {
				return false
			}
		}
		return true
	}
	payload := shellPayloadFromParts(parts)
	return payload == spec.Shell
}

func commandLineMatchesSpec(line string, spec *Spec) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	if spec.Path == "" && line == spec.Shell {
		return true
	}
	if spec.Path != "" {
		if !strings.HasPrefix(line, spec.Path) {
			return false
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, spec.Path))
		if len(spec.Args) == 0 {
			return rest == ""
		}
		got := strings.Fields(rest)
		if len(got) != len(spec.Args) {
			return false
		}
		for i := range spec.Args {
			if got[i] != spec.Args[i] {
				return false
			}
		}
		return true
	}
	return shellPayloadFromCommandLine(line) == spec.Shell
}

func isShellExecutable(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return base == "sh" || base == "bash" || base == "zsh" || base == "cmd.exe" || base == "cmd"
}

func cmdlineMatchesPartsPtr(parts []string, spec *Spec) bool {
	if cmdlineMatchesSpec(parts, spec) {
		return true
	}
	if len(parts) == 1 {
		return commandLineMatchesSpec(parts[0], spec)
	}
	joined := strings.TrimSpace(strings.Join(parts, " "))
	return commandLineMatchesSpec(joined, spec)
}
