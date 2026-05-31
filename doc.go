// Package proctree runs shell and exec commands in an isolated process group or tree,
// streams stdout/stderr, and kills the full child tree on context cancellation or timeout.
//
// Introspection helpers (Inspect, Children, Descendants) read OS process tables directly.
// See README "Non-goals" for features intentionally out of scope (PTY, pipelines, Job Objects).
//
// Platform notes:
//   - Unix (Linux, macOS): commands start with Setpgid; KillTreeByPID sends SIGKILL to -pid.
//   - macOS: Alive ignores zombie processes; KillTreeByPID also signals the leader pid.
//   - Windows: commands start in a new process group; KillTreeByPID uses taskkill /T /F.
//     Inspect uses Toolhelp snapshots and NT APIs (NtQueryInformationProcess, GetProcessTimes).
//   - VerifyOwnership fails closed when cmdline or create-time cannot be confirmed.
package proctree
