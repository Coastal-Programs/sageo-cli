#!/usr/bin/env bash
# Fail if any non-integration test file appears to make real HTTP calls.
#
# Rule: every *_test.go under internal/ that uses http.DefaultClient,
# http.Get(, or constructs a bare http.Client{} for outbound calls MUST start
# with the `//go:build integration` build tag. Unit tests must use
# httptest.NewServer or a mock HTTPClient instead (see
# internal/common/testutil/httpfake.go).
set -euo pipefail

cd "$(dirname "$0")/.."

# Collect candidates: test files under internal/ that do NOT have the
# integration build tag on their first line. Portable across macOS bash 3.2.
non_integration=$(find internal -name "*_test.go" -print0 \
  | xargs -0 grep -L "^//go:build integration" || true)

count=0
offenders=""
while IFS= read -r f; do
  [ -z "$f" ] && continue
  count=$((count + 1))
  # Any reference to these patterns in a non-integration test is a red flag.
  if grep -qE 'http\.DefaultClient|http\.Get\(|http\.Post\(|http\.Head\(' "$f"; then
    offenders="${offenders}  ${f}
"
  fi
done <<EOF
$non_integration
EOF

if [ -n "$offenders" ]; then
  echo "ERROR: non-integration tests making real HTTP calls:"
  printf '%s' "$offenders"
  echo ""
  echo "Fix: move the test behind //go:build integration + SAGEO_LIVE_TESTS,"
  echo "or refactor to use internal/common/testutil/httpfake.go."
  exit 1
fi

echo "check-no-live-tests: OK ($count unit test files scanned)"
