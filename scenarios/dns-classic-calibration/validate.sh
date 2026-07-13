#!/usr/bin/env bash
set -euo pipefail
python3 - "$PWD" <<'PY'
import hashlib,json,pathlib,sys
root=pathlib.Path(sys.argv[1]); lock=json.loads((root/'authority-lock.json').read_text())
assert lock['commit']=='8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'
for name,expected in lock['files'].items():
    actual=hashlib.sha256((root/name).read_bytes()).hexdigest()
    if actual!=expected: raise SystemExit(f'authority-lock mismatch for {name}: {actual}')
print(f"Validated classic DNS scenario package authority lock at {lock['commit']}.")
PY
