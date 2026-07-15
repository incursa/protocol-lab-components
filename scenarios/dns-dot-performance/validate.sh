#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
python3 - <<'PY'
import hashlib,json,pathlib
root=pathlib.Path('.')
lock=json.loads((root/'authority-lock.json').read_text())
assert lock['commit']=='c0475b05cb80362760ac57e58ecfa1610a766c10'
for name,expected in lock['files'].items():
    path=root/name
    assert path.is_file(),f'missing {name}'
    actual=hashlib.sha256(path.read_bytes()).hexdigest()
    assert actual==expected,f'{name}: expected {expected}, observed {actual}'
print(f"Validated DNS-over-TLS scenario package authority lock at {lock['commit']}.")
PY
