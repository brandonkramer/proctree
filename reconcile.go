package proctree

// ReconcileOutcome describes the result of ReconcilePID.
type ReconcileOutcome int

const (
	// ReconcileNotRunning indicates pid is absent or not a live supervised process.
	ReconcileNotRunning ReconcileOutcome = iota
	// ReconcileKilled indicates pid matched own and was terminated.
	ReconcileKilled
	// ReconcileUnverified indicates pid is alive but ownership could not be confirmed.
	// The process is left running (fail closed).
	ReconcileUnverified
)

// ReconcileResult is the outcome of a recovery reconcile attempt.
type ReconcileResult struct {
	Outcome ReconcileOutcome
}

// ReconcilePID verifies ownership of pid and kills the process tree when confirmed.
// When pid is not alive, returns ReconcileNotRunning.
// When pid is alive but ownership cannot be verified, returns ReconcileUnverified
// without sending signals (fail closed).
func ReconcilePID(pid int, own *Ownership) ReconcileResult {
	return ReconcilePIDOpts(pid, own, nil)
}

// ReconcilePIDOpts is ReconcilePID with explicit verification options.
func ReconcilePIDOpts(pid int, own *Ownership, opts *VerifyOptions) ReconcileResult {
	if pid <= 0 || !Alive(pid) {
		return ReconcileResult{Outcome: ReconcileNotRunning}
	}
	if !VerifyOwnershipOpts(pid, own, opts) {
		return ReconcileResult{Outcome: ReconcileUnverified}
	}
	_ = KillTreeByPID(pid)
	return ReconcileResult{Outcome: ReconcileKilled}
}
