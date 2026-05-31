// Package proctree runs shell and exec commands in an isolated process group or tree,
// streams stdout/stderr, and kills the full child tree on context cancellation or timeout.
//
// Introspection helpers (Inspect, Children, Descendants) read OS process tables directly.
// See README "Non-goals" for features intentionally out of scope (PTY, pipelines).
//
// Platform notes:
//   - Linux: Setpgid isolation; KillTreeByPID sends SIGKILL to -pid; inspect via /proc.
//   - macOS: Setpgid isolation; KillTreeByPID signals the group and leader pid; inspect
//     via sysctl (KinfoProc, procargs2); Alive skips zombies.
//   - Windows: new process group plus kill-on-close Job Object on Run; tree kill via
//     TerminateJobObject with descendant-walk fallback; inspect via Toolhelp and NT APIs.
//   - VerifyOwnership and ReconcilePID fail closed when cmdline or create-time cannot
//     be confirmed.
package proctree
