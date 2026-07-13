#!/usr/bin/env bash
set -euo pipefail
python3 - "$PWD" <<'PY'
import hashlib, json, pathlib, sys
root = pathlib.Path(sys.argv[1])
lock = json.loads((root / "authority-lock.json").read_text(encoding="utf-8"))
for relative, expected in lock["files"].items():
    path = root / relative
    if not path.is_file():
        raise SystemExit(f"Authority-locked file is missing: {relative}")
    actual = hashlib.sha256(path.read_bytes()).hexdigest()
    if actual != expected:
        raise SystemExit(f"Authority-locked file hash mismatch for {relative}: expected {expected}, observed {actual}")
print(f"Validated gRPC/H2 scenario package authority lock at {lock['commit']}.")
PY
