//go:build windows

package proctree

func verifyProcessGroup(_ int) bool {
	// Windows lacks a cheap pgid==pid invariant; rely on cmdline + create time.
	return true
}
