#!/usr/bin/env bash
set -euo pipefail
python3 - "$PWD" <<'PY'
import hashlib,json,pathlib,sys
r=pathlib.Path(sys.argv[1]); lock=json.loads((r/'authority-lock.json').read_text())
assert lock['commit']=='c0475b05cb80362760ac57e58ecfa1610a766c10'
for p,h in lock['files'].items(): assert hashlib.sha256((r/p).read_bytes()).hexdigest()==h,p
assert len(json.loads((r/'protocol-lab-package.json').read_text())['providedScenarios'])==8
PY
