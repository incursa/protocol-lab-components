#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
test "$(sha256sum "$root/scenarios/masque/connect-udp-tunnel.yaml" | cut -d' ' -f1)" = "9d4019cf6fa8624e4863b248ebd393c754db9f94e4e5ea167032dc4f9f4b6df4"
test "$(sha256sum "$root/load-profiles/masque-connect-udp-comparison.yaml" | cut -d' ' -f1)" = "5feb40a66cf6de70d744a96055968eba040b9458087445e866cbcd41780ad304"
test "$(sha256sum "$root/suites/masque-connect-udp-performance-comparison.yaml" | cut -d' ' -f1)" = "2e0f7ae0e46afe75cc797b5bad98af9abb9e58649493c75064cd388094b7ca4d"
echo "Validated MASQUE scenario package authority lock."
