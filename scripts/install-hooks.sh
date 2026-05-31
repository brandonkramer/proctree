#!/usr/bin/env bash
# Install lefthook and sync git hooks (pre-push -> scripts/check.sh).
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

bin="$(go env GOPATH)/bin"
export PATH="${bin}:${PATH}"

go install github.com/evilmartians/lefthook@latest
lefthook install

hook="${root}/.git/hooks/pre-push"
lefthook_bin="${bin}/lefthook"
if ! grep -q 'LEFTHOOK_BIN=' "$hook"; then
  tmp="$(mktemp)"
  {
    echo "#!/bin/sh"
    echo "LEFTHOOK_BIN=\"${lefthook_bin}\""
    echo "export LEFTHOOK_BIN"
    tail -n +2 "$hook"
  } > "$tmp"
  mv "$tmp" "$hook"
  chmod +x "$hook"
fi

echo "hooks installed: pre-push runs ./scripts/check.sh via ${lefthook_bin}"
