package proctree

import "time"

// WaitNotAlive blocks until pid is no longer running or timeout elapses.
// Returns true when the process is no longer alive.
func WaitNotAlive(pid int, timeout time.Duration) bool {
	if pid <= 0 {
		return true
	}
	if timeout <= 0 {
		return !Alive(pid)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return !Alive(pid)
}
