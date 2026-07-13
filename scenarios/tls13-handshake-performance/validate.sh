#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${root}"
python3 - <<'PY'
import hashlib, json, pathlib
root = pathlib.Path('.')
lock = json.loads((root / 'authority-lock.json').read_text(encoding='utf-8'))
assert lock['commit'] == '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'
for relative, expected in lock['files'].items():
    actual = hashlib.sha256((root / relative).read_bytes()).hexdigest()
    if actual != expected:
        raise SystemExit(f'authority-lock mismatch for {relative}: expected {expected}, observed {actual}')
print(f"Validated TLS scenario package authority lock at {lock['commit']}.")
PY
