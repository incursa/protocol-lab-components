#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
python3 - <<'PY'
import hashlib,json,pathlib
root=pathlib.Path('.')
lock=json.loads((root/'authority-lock.json').read_text())
assert lock['commit']=='8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'
for name,expected in lock['files'].items():
    path=root/name
    assert path.is_file(),f'missing {name}'
    actual=hashlib.sha256(path.read_bytes()).hexdigest()
    assert actual==expected,f'{name}: expected {expected}, observed {actual}'
print(f"Validated DNS-over-TLS scenario package authority lock at {lock['commit']}.")
PY
