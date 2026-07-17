#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
test "$(sha256sum "$root/scenarios/webtransport/session-bidi-echo.yaml" | cut -d' ' -f1)" = "a4f92892214382c839740696c5ddfb1d6eee7afb1088498606c3ed7fa9e82a00"
test "$(sha256sum "$root/load-profiles/webtransport-smoke.yaml" | cut -d' ' -f1)" = "6a2bf607380a218b57b7bf2c959a6ed9c4b70590c482f9970df60b643d83011c"
test "$(sha256sum "$root/suites/webtransport-performance-smoke.yaml" | cut -d' ' -f1)" = "6caff4c72e5e839d88aea502f6f56788839132136a3b202f3573a6941ec3666f"
echo "Validated WebTransport scenario package authority lock."
