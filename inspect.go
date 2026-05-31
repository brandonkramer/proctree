package proctree

import (
	"errors"
	"fmt"
	"time"
)

// ErrProcessNotFound indicates pid does not refer to a live process we can read.
var ErrProcessNotFound = errors.New("proctree: process not found")

// CmdlineMatcher returns whether argv parts belong to the supervised process.
type CmdlineMatcher func(parts []string) bool

// ProcessInfo is a point-in-time snapshot of one process.
type ProcessInfo struct {
	PID        int
	PPID       int
	PGID       int
	Name       string
	Status     string
	Cmdline    []string
	CreateTime time.Time
	MemoryRSS  uint64
	OpenFiles  int
}

// Ownership describes a supervised process for conservative verification.
type Ownership struct {
	Spec      Spec
	StartedAt time.Time
	// Match, when set, overrides Spec-based cmdline matching.
	Match CmdlineMatcher
}

// VerifyOptions tune create-time and cmdline checks for VerifyOwnership.
type VerifyOptions struct {
	// MaxClockSkew allows the OS create time to be slightly after StartedAt.
	MaxClockSkew time.Duration
	// MaxStartLead allows the OS create time to be before StartedAt (spawn latency).
	MaxStartLead time.Duration
	// Match, when set, overrides Ownership.Match and Spec-based cmdline matching.
	Match CmdlineMatcher
	// SkipProcessGroup skips the Unix pgid-leader check when true.
	SkipProcessGroup bool
}

// DefaultVerifyOptions is used by VerifyOwnership.
func DefaultVerifyOptions() VerifyOptions {
	return VerifyOptions{
		MaxClockSkew: 2 * time.Second,
		MaxStartLead: 5 * time.Second,
	}
}

// Inspect returns a snapshot of pid. Fails when the process is gone or unreadable.
func Inspect(pid int) (ProcessInfo, error) {
	if pid <= 0 {
		return ProcessInfo{}, ErrProcessNotFound
	}
	return readProcessInfo(pid)
}

// CreateTime returns the process start time when available.
func CreateTime(pid int) (time.Time, error) {
	info, err := Inspect(pid)
	if err != nil {
		return time.Time{}, err
	}
	if info.CreateTime.IsZero() {
		return time.Time{}, fmt.Errorf("proctree: create time unavailable for pid %d", pid)
	}
	return info.CreateTime, nil
}

// Cmdline returns argv for pid when available.
func Cmdline(pid int) ([]string, error) {
	info, err := Inspect(pid)
	if err != nil {
		return nil, err
	}
	if len(info.Cmdline) == 0 {
		return nil, fmt.Errorf("proctree: cmdline unavailable for pid %d", pid)
	}
	return append([]string(nil), info.Cmdline...), nil
}

// Children returns direct child pids of pid.
func Children(pid int) ([]int, error) {
	if pid <= 0 {
		return nil, ErrProcessNotFound
	}
	return listChildPIDs(pid)
}

// Descendants returns pid and all descendant pids breadth-first.
func Descendants(root int) ([]int, error) {
	if root <= 0 {
		return nil, ErrProcessNotFound
	}
	seen := map[int]struct{}{root: {}}
	order := []int{root}
	for i := 0; i < len(order); i++ {
		kids, err := listChildPIDs(order[i])
		if err != nil {
			return nil, err
		}
		for _, kid := range kids {
			if _, ok := seen[kid]; ok {
				continue
			}
			seen[kid] = struct{}{}
			order = append(order, kid)
		}
	}
	return order, nil
}

// InspectTree returns snapshots for pid and all descendants.
func InspectTree(root int) ([]ProcessInfo, error) {
	ids, err := Descendants(root)
	if err != nil {
		return nil, err
	}
	out := make([]ProcessInfo, 0, len(ids))
	for _, id := range ids {
		info, err := readProcessInfo(id)
		if err != nil {
			if errors.Is(err, ErrProcessNotFound) {
				continue
			}
			return nil, err
		}
		out = append(out, info)
	}
	return out, nil
}

// VerifyOwnership reports whether pid still matches own using cmdline, optional
// create-time window, and platform group checks. Fails closed when uncertain.
func VerifyOwnership(pid int, own *Ownership) bool {
	return VerifyOwnershipOpts(pid, own, nil)
}

// VerifyOwnershipOpts is VerifyOwnership with explicit options.
func VerifyOwnershipOpts(pid int, own *Ownership, opts *VerifyOptions) bool {
	if own == nil {
		return false
	}
	if opts == nil {
		defaults := DefaultVerifyOptions()
		opts = &defaults
	}
	if pid <= 0 || !Alive(pid) {
		return false
	}
	if !opts.SkipProcessGroup && !verifyProcessGroup(pid) {
		return false
	}
	parts, err := readCmdlineParts(pid)
	if err != nil || !matchCmdline(parts, own, opts) {
		return false
	}
	if own.StartedAt.IsZero() {
		return true
	}
	created, err := readCreateTime(pid)
	if err != nil || created.IsZero() {
		return false
	}
	if opts.MaxClockSkew <= 0 {
		opts.MaxClockSkew = DefaultVerifyOptions().MaxClockSkew
	}
	if opts.MaxStartLead <= 0 {
		opts.MaxStartLead = DefaultVerifyOptions().MaxStartLead
	}
	earliest := own.StartedAt.Add(-opts.MaxStartLead)
	latest := own.StartedAt.Add(opts.MaxClockSkew)
	return !created.Before(earliest) && !created.After(latest)
}

func matchCmdline(parts []string, own *Ownership, opts *VerifyOptions) bool {
	switch {
	case opts != nil && opts.Match != nil:
		return opts.Match(parts)
	case own != nil && own.Match != nil:
		return own.Match(parts)
	default:
		return cmdlineMatchesPartsPtr(parts, &own.Spec)
	}
}
