package proctree

// VerifyOwned reports whether pid still refers to the process started for spec.
// Returns false when ownership cannot be confirmed (fail closed).
func VerifyOwned(pid int, spec *Spec) bool {
	if spec == nil {
		return false
	}
	return VerifyOwnership(pid, &Ownership{Spec: *spec})
}
