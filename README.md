# proctree

Cross-platform Go helpers for running shell and exec commands in an isolated process group or tree, streaming stdout/stderr, killing the full child tree on context cancellation or timeout, and inspecting process state.

## Install

From [pkg.go.dev](https://pkg.go.dev/github.com/brandonkramer/proctree):

```bash
go get github.com/brandonkramer/proctree
```

## Quick start

### Shell command

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

var buf bytes.Buffer
spec := &proctree.Spec{Shell: "make test"}
opts := &proctree.Options{
    OnStdout: func(line string) { fmt.Println("out:", line) },
    Stdout:   &buf,
}
res, err := proctree.Run(ctx, spec, opts)
if err != nil {
    // context.Canceled, context.DeadlineExceeded, or non-zero exit
}
fmt.Println("exit", res.ExitCode, "pid", res.PID, "started", res.StartedAt)
```

### Exec (argv) — preferred when possible

```go
res, err := proctree.Run(ctx, &proctree.Spec{
    Path: "/usr/bin/git",
    Args: []string{"status", "--short"},
    Dir:  "/path/to/repo",
    Env:  []string{"GIT_TERMINAL_PROMPT=0"},
}, &proctree.Options{
    Stdin: strings.NewReader(""),
})
```


### Recovery reconcile

```go
res := proctree.ReconcilePID(pid, &proctree.Ownership{
    Spec:      *spec,
    StartedAt: startedAt,
})
switch res.Outcome {
case proctree.ReconcileKilled:
    // verified and terminated
case proctree.ReconcileUnverified:
    // alive but not ours — left running (fail closed)
case proctree.ReconcileNotRunning:
    // already gone
}
```

### Tee stdout/stderr to logs + callbacks

```go
opts := proctree.TeeOptions(stdoutFile, stderrFile,
    func(line string) { emit("stdout", line) },
    func(line string) { emit("stderr", line) },
)
proctree.Run(ctx, spec, opts)
```

### Ownership verification (PID reuse safe)

```go
own := &proctree.Ownership{
    Spec:      *spec,
    StartedAt: res.StartedAt,
}
if proctree.VerifyOwnership(pid, own) {
    _ = proctree.KillTreeByPID(pid)
}
```

Custom matcher for processes that do not match `Spec` shape:

```go
own := &proctree.Ownership{
    StartedAt: res.StartedAt,
    Match: func(parts []string) bool {
        return len(parts) > 0 && strings.Contains(parts[0], "my-worker")
    },
}
proctree.VerifyOwnershipOpts(pid, own, &proctree.VerifyOptions{
    SkipProcessGroup: true,
})
```

`VerifyOwnership` checks cmdline match, Unix process-group leader (when applicable), and optional create-time window. Fails closed when uncertain.

### Introspection and recovery

```go
info, err := proctree.Inspect(pid)
// info.Cmdline, info.CreateTime, info.MemoryRSS, info.Status, info.OpenFiles (linux)

kids, _ := proctree.Children(pid)
tree, _ := proctree.InspectTree(pid) // pid + descendants
```

### Low-level helpers

```go
cmd := proctree.NewCommand(&proctree.Spec{Shell: "sleep 300"})
cmd.Start()
proctree.KillTree(cmd)

proctree.KillTreeByPID(pid)
proctree.Alive(pid)
proctree.VerifyOwned(pid, spec)
```

## Platform behavior

| Platform | Process isolation | Tree kill | Alive | Inspect sources | Ownership verify |
|----------|-------------------|-----------|-------|-----------------|------------------|
| Linux    | `Setpgid` | `SIGKILL` to `-pid` | `/proc` stat (skip zombies) | `/proc` | cmdline + pgid + create time |
| macOS    | `Setpgid` | `SIGKILL` to `-pid` and `pid` | sysctl stat (skip zombies) | sysctl (`KinfoProc`, procargs2) | cmdline + pgid + create time |
| Windows  | process group + Job Object | `TerminateJobObject` (fallback: descendant walk) | `OpenProcess` | Toolhelp + NT APIs | cmdline + create time |

### Windows notes

- `Run` assigns each child to a kill-on-close Job Object when the OS allows it.
- Inspect uses `NtQueryInformationProcess`, `GetProcessTimes`, and `GetProcessMemoryInfo` (no `wmic`).

### Unix zombie processes

`Alive` consults `/proc` on Linux and sysctl process state on macOS so zombies count as not running. Use `VerifyOwnership` or `ReconcilePID` before killing stale pids.

## API surface

**Execution**
- `Run(ctx, spec, opts)` — context-first execution with streaming, optional stdin/writer sinks, tree kill on cancel/timeout
- `NewCommand(spec)` — build a configured `*exec.Cmd` without starting
- `KillTree(cmd)` / `KillTreeByPID(pid)` — terminate process trees
- `Alive(pid)` — liveness probe

**Recovery**
- `ReconcilePID(pid, own)` / `ReconcilePIDOpts(pid, own, opts)` — verify-and-kill for daemon recovery
- `WaitNotAlive(pid, timeout)` — poll until process exits

**Streaming helpers**
- `TeeLine(w, fn)` / `TeeOptions(stdoutW, stderrW, onStdout, onStderr)` — mirror lines to writers and callbacks

**Verification**
- `VerifyOwned(pid, spec)` — cmdline + platform checks
- `VerifyOwnership(pid, own)` — adds create-time window and optional `CmdlineMatcher`
- `VerifyOwnershipOpts(pid, own, opts)` — tune timing, matcher, and `SkipProcessGroup`

**Introspection**
- `Inspect(pid)` — point-in-time `ProcessInfo` snapshot
- `InspectTree(root)` — snapshots for root + descendants
- `Children(pid)` / `Descendants(root)` — process tree discovery
- `Cmdline(pid)` / `CreateTime(pid)` — convenience accessors

## Development

Run the same checks as CI before pushing:

```bash
./scripts/check.sh
```

Git hooks via [lefthook](https://github.com/evilmartians/lefthook) (once per clone):

```bash
./scripts/install-hooks.sh
```

`git push` then runs `./scripts/check.sh` automatically. Skip with `LEFTHOOK=0 git push`.

`scripts/check.sh` runs `go test -race`, a Linux cross-compile check, and `golangci-lint` installed with your local Go toolchain (must be >= `go.mod`). CI pins `GOLANGCI_LINT_VERSION` in `.github/workflows/test.yml`.

## Releases

Tagged semver releases are published to [pkg.go.dev](https://pkg.go.dev/github.com/brandonkramer/proctree). See [GitHub releases](https://github.com/brandonkramer/proctree/releases) for notes.

## License

MIT
