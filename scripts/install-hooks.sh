#!/usr/bin/env bash
# Install lefthook and sync git hooks (pre-push -> scripts/check.sh).
set -euo pipefail

cd "$(dirname "$0")/.."

export PATH="$(go env GOPATH)/bin:${PATH}"

go install github.com/evilmartians/lefthook@latest
lefthook install

echo "hooks installed: pre-push runs ./scripts/check.sh"
