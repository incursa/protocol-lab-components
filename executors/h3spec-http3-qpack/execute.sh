#!/usr/bin/env bash
set -euo pipefail

host_name="${H3SPEC_HOST:-127.0.0.1}"
port="${H3SPEC_PORT:-4433}"
h3spec_executable="${H3SPEC_EXECUTABLE:-h3spec}"
timeout_ms="${H3SPEC_TIMEOUT_MS:-5000}"
output_root="${H3SPEC_OUTPUT_ROOT:-artifacts/h3spec-http3-qpack}"
plan_only="${H3SPEC_PLAN_ONLY:-false}"
match_values=(${H3SPEC_MATCH:-HTTP/3 QPACK})
skip_values=(${H3SPEC_SKIP:-})

mkdir -p "$output_root"

stdout_path="$output_root/h3spec-stdout.txt"
stderr_path="$output_root/h3spec-stderr.txt"
metadata_path="$output_root/h3spec-metadata.json"
results_path="$output_root/h3spec-results.json"
report_path="$output_root/h3spec-report.md"

args=()
if [[ "${H3SPEC_NO_VALIDATE:-false}" == "true" ]]; then
  args+=(--no-validate)
fi

for item in "${match_values[@]}"; do
  args+=(--match "$item")
done

for item in "${skip_values[@]}"; do
  args+=(--skip "$item")
done

args+=(--timeout "$timeout_ms" "$host_name" "$port")

exit_code=""
status="not-run"
if [[ "$plan_only" == "true" ]]; then
  printf 'plan-only: %s %s\n' "$h3spec_executable" "${args[*]}" > "$stdout_path"
  : > "$stderr_path"
else
  set +e
  "$h3spec_executable" "${args[@]}" > "$stdout_path" 2> "$stderr_path"
  exit_code="$?"
  set -e
  if [[ "$exit_code" == "0" ]]; then
    status="passed"
  else
    status="failed"
  fi
fi

python_bin="$(command -v python3 || command -v python)"
"$python_bin" - "$metadata_path" "$host_name" "$port" "$timeout_ms" "$plan_only" "$status" "$exit_code" "${match_values[@]}" <<'PY'
import json
import sys

path, host, port, timeout_ms, plan_only, status, exit_code, *matches = sys.argv[1:]
payload = {
    "executor": "h3spec-http3-qpack",
    "tool": "h3spec",
    "toolVersion": "v0.1.13",
    "arguments": [],
    "match": matches,
    "skip": [],
    "timeoutMilliseconds": int(timeout_ms),
    "host": host,
    "h3specTargetHost": host,
    "port": int(port),
    "planOnly": plan_only == "true",
    "exitCode": None if exit_code == "" else int(exit_code),
    "status": status,
}
with open(path, "w", encoding="utf-8") as handle:
    json.dump(payload, handle, indent=2)
PY

"$python_bin" scripts/parse-h3spec-results.py \
  --stdout "$stdout_path" \
  --stderr "$stderr_path" \
  --metadata "$metadata_path" \
  --json-output "$results_path" \
  --markdown-output "$report_path"

echo "h3spec executor artifacts written to $output_root"
