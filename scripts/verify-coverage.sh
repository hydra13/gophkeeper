#!/bin/bash
# verify-coverage.sh — быстрая проверка покрытия тестами.
# Воспроизводит ту же логику, что и make cover-check и CI.
set -euo pipefail

echo "==> Running tests with coverage..."
PACKAGES=$(go list ./... | grep -v '/pbv1' | grep -v 'proto/v1')
go test -race -coverprofile=coverage.out $PACKAGES

echo "==> Filtering coverage profile..."
grep -ve '/mocks/' -e '\.pb\.go' -e '/proto/v1/' -e '/pbv1/' coverage.out > coverage_filtered.out

COVERAGE=$(go tool cover -func=coverage_filtered.out | tail -1 | awk '{print $NF}' | tr -d '%')
rm -f coverage_filtered.out

echo "==> Coverage (excl. mocks, generated): ${COVERAGE}%"
THRESHOLD=70
if [ "$(echo "$COVERAGE < $THRESHOLD" | bc -l)" -eq 1 ]; then
    echo "FAIL: coverage ${COVERAGE}% is below ${THRESHOLD}% threshold"
    exit 1
fi
echo "PASS: coverage ${COVERAGE}% meets ${THRESHOLD}% threshold"
