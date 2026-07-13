#!/usr/bin/env bash
set -euo pipefail
python3 - "$PWD" <<'PY'
import hashlib,json,pathlib,sys
r=pathlib.Path(sys.argv[1]); lock=json.loads((r/'authority-lock.json').read_text())
assert lock['commit']=='8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'
for p,h in lock['files'].items(): assert hashlib.sha256((r/p).read_bytes()).hexdigest()==h,p
assert len(json.loads((r/'protocol-lab-package.json').read_text())['providedScenarios'])==7
PY
