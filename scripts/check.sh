#!/usr/bin/env bash
# Run the same checks as CI before pushing.
set -euo pipefail

cd "$(dirname "$0")/.."

GOLANGCI_LINT_VERSION="${GOLANGCI_LINT_VERSION:-v2.12.2}"

lint_ready() {
	command -v golangci-lint >/dev/null || return 1
	golangci-lint version 2>&1 | grep -Eq 'go1\.(2[6-9]|[3-9][0-9])'
}

if ! lint_ready; then
	echo "==> installing golangci-lint ${GOLANGCI_LINT_VERSION} with $(go version)"
	go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"
	export PATH="$(go env GOPATH)/bin:${PATH}"
fi

echo "==> go test -race -cover ./..."
go test -race -cover ./...

echo "==> linux compile check"
GOOS=linux GOARCH=amd64 go test -c -o /dev/null ./...

echo "==> golangci-lint"
golangci-lint run ./...

echo "check: ok"
