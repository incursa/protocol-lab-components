#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
test "$(sha256sum "$root/scenarios/webtransport/session-bidi-echo.yaml" | cut -d' ' -f1)" = "a4f92892214382c839740696c5ddfb1d6eee7afb1088498606c3ed7fa9e82a00"
test "$(sha256sum "$root/scenarios/webtransport/session-datagram-echo.yaml" | cut -d' ' -f1)" = "d9c8bd82527d9a61b3c03c085dfd1b9991dc604d6e5ccbc1c80c0f383d382888"
test "$(sha256sum "$root/load-profiles/webtransport-smoke.yaml" | cut -d' ' -f1)" = "6a2bf607380a218b57b7bf2c959a6ed9c4b70590c482f9970df60b643d83011c"
test "$(sha256sum "$root/load-profiles/webtransport-datagram-smoke.yaml" | cut -d' ' -f1)" = "13bfc3201e420ebff27499b13d54d3c089074307d2d754e828c937bb283671a7"
test "$(sha256sum "$root/suites/webtransport-performance-smoke.yaml" | cut -d' ' -f1)" = "6caff4c72e5e839d88aea502f6f56788839132136a3b202f3573a6941ec3666f"
test "$(sha256sum "$root/suites/webtransport-datagram-performance-smoke.yaml" | cut -d' ' -f1)" = "d49f5c5b71cba631748e0a7164b50cc7a35d895e3b7fb49a6820124a1d869054"
echo "Validated WebTransport scenario package authority lock."
