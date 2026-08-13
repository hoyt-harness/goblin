#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Single source of truth for CI/code-quality checks.
# Invoked locally by hooks/pre-push (blocking) and by .github/workflows/ci.yml.
set -e

echo "ci-check: running Go checks..."

go build ./...
echo "  build: OK"

go test ./...
echo "  tests: OK"

if command -v goimports >/dev/null 2>&1; then
    bad=$(goimports -l . 2>/dev/null | grep -v vendor || true)
    if [ -n "$bad" ]; then
        echo "  goimports: FAIL — run 'goimports -w .' to fix:"
        echo "$bad"
        exit 1
    fi
    echo "  goimports: OK"
fi

if command -v positronikal-check >/dev/null 2>&1; then
    positronikal-check .
    echo "  positronikal-check: OK"
fi

echo "ci-check passed."
