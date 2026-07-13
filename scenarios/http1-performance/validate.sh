#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${root}"
printf '%s  %s\n' '04ef584037f88efc89d0aca23354f02b3507ed7e6ce62d60f138216d93e41fef' 'scenarios/http1/core/plaintext.yaml' | sha256sum --check --status
printf '%s  %s\n' '838ee7b8b6b4e1088a83dd5681b88d52f636c284e8b03df1f20ffc75354beebf' 'scenarios/http1/core/json.yaml' | sha256sum --check --status
printf '%s  %s\n' '85d0d895037410b33150eeeb73ac52b826919bbbd9eb1d6a19dab4b32b070beb' 'load-profiles/http1-smoke.yaml' | sha256sum --check --status
echo 'Validated HTTP/1 scenario package authority lock at a4dcd74e5c8907907ccc58808da92d2b920b2fbc.'
