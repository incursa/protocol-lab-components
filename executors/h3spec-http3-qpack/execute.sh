#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
host_name="${H3SPEC_HOST:-127.0.0.1}"
port="${H3SPEC_PORT:-4433}"
h3spec_executable="${H3SPEC_EXECUTABLE:-h3spec}"
timeout_ms="${H3SPEC_TIMEOUT_MS:-5000}"
output_root="${H3SPEC_OUTPUT_ROOT:-artifacts/h3spec-http3-qpack}"
target_url=""
plan_only="${H3SPEC_PLAN_ONLY:-false}"
mode="${H3SPEC_MODE:-focused}"
match_values=()
if [[ -n "${H3SPEC_MATCH:-}" ]]; then
  # shellcheck disable=SC2206
  match_values=(${H3SPEC_MATCH})
fi
skip_values=(${H3SPEC_SKIP:-})
no_validate="${H3SPEC_NO_VALIDATE:-false}"
acquire_h3spec="${H3SPEC_ACQUIRE:-false}"
fail_on_h3spec_failures="${H3SPEC_FAIL_ON_FAILURES:-false}"
h3spec_version="${H3SPEC_VERSION:-v0.1.13}"
python_bin_for_target="$(command -v python3 || command -v python)"

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --host) host_name="$2"; shift 2 ;;
    --port) port="$2"; shift 2 ;;
    --timeout-ms) timeout_ms="$2"; shift 2 ;;
    --output-root) output_root="$2"; shift 2 ;;
    --h3spec-executable) h3spec_executable="$2"; shift 2 ;;
    --mode) mode="$2"; shift 2 ;;
    --match) match_values+=("$2"); shift 2 ;;
    --skip) skip_values+=("$2"); shift 2 ;;
    --no-validate) no_validate="true"; shift ;;
    --plan-only) plan_only="true"; shift ;;
    --acquire-h3spec) acquire_h3spec="true"; shift ;;
    --h3spec-version) h3spec_version="$2"; shift 2 ;;
    --fail-on-h3spec-failures) fail_on_h3spec_failures="true"; shift ;;
    http://*|https://*) target_url="$1"; shift ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done

if [[ -n "$target_url" ]]; then
  parsed_target="$("$python_bin_for_target" - "$target_url" <<'PY'
import sys
from urllib.parse import urlparse

uri = urlparse(sys.argv[1])
host = uri.hostname or "127.0.0.1"
port = uri.port or (443 if uri.scheme == "https" else 80)
print(f"{host}\n{port}")
PY
  )"
  host_name="$(printf '%s\n' "$parsed_target" | sed -n '1p')"
  port="$(printf '%s\n' "$parsed_target" | sed -n '2p')"
fi

if [[ "$mode" == "full" ]]; then
  match_values=()
elif [[ "$mode" == "qpack" ]]; then
  match_values=("QPACK")
elif [[ "$mode" != "focused" ]]; then
  echo "Unsupported mode: $mode" >&2
  exit 2
elif [[ "${#match_values[@]}" -eq 0 ]]; then
  match_values=("HTTP/3" "QPACK")
fi

if [[ "$output_root" != /* ]]; then
  output_root="$script_dir/$output_root"
fi

mkdir -p "$output_root"

stdout_path="$output_root/h3spec-stdout.txt"
stderr_path="$output_root/h3spec-stderr.txt"
metadata_path="$output_root/h3spec-metadata.json"
results_path="$output_root/h3spec-results.json"
report_path="$output_root/h3spec-report.md"
tool_manifest_path=""

if [[ "$acquire_h3spec" == "true" ]]; then
  tool_root="$script_dir/artifacts/tools/h3spec-$h3spec_version"
  mkdir -p "$tool_root"
  asset_name="h3spec-linux-x86_64"
  download_url="https://github.com/kazu-yamamoto/h3spec/releases/download/$h3spec_version/$asset_name"
  binary_path="$tool_root/$asset_name"
  wrapper_path="$tool_root/h3spec-docker-wrapper.sh"
  if [[ ! -f "$binary_path" ]]; then
    if command -v curl >/dev/null 2>&1; then
      curl -fsSL "$download_url" -o "$binary_path"
    elif command -v wget >/dev/null 2>&1; then
      wget -q "$download_url" -O "$binary_path"
    else
      echo "curl or wget is required to acquire h3spec." >&2
      exit 2
    fi
  fi
  sha256="$(sha256sum "$binary_path" | awk '{print $1}')"
  cat > "$wrapper_path" <<EOF
#!/usr/bin/env bash
set -euo pipefail
tool_root="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
exec docker run --rm -v "\$tool_root:/tools:ro" --network host ubuntu:24.04 /bin/sh -lc 'cp /tools/$asset_name /tmp/h3spec && chmod +x /tmp/h3spec && exec /tmp/h3spec "\$@"' h3spec "\$@"
EOF
  chmod +x "$wrapper_path"
  h3spec_executable="$wrapper_path"
  tool_manifest_path="$tool_root/h3spec-tool-manifest.json"
  python_bin_for_manifest="$(command -v python3 || command -v python)"
  "$python_bin_for_manifest" - "$tool_manifest_path" "$h3spec_version" "$asset_name" "$download_url" "$sha256" "$wrapper_path" <<'PY'
import datetime
import json
import sys

path, version, asset, url, sha256, executable = sys.argv[1:]
payload = {
    "tool": "h3spec",
    "version": version,
    "asset": asset,
    "downloadUrl": url,
    "sha256": sha256,
    "acquiredUtc": datetime.datetime.utcnow().replace(microsecond=0).isoformat() + "Z",
    "executable": executable,
    "prefixArguments": [],
    "wrapperImage": "ubuntu:24.04",
    "recommendedHostName": "127.0.0.1",
}
with open(path, "w", encoding="utf-8") as handle:
    json.dump(payload, handle, indent=2)
PY
fi

args=()
if [[ "$no_validate" == "true" ]]; then
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
"$python_bin" - "$metadata_path" "$host_name" "$port" "$timeout_ms" "$plan_only" "$status" "$exit_code" "$mode" "$h3spec_version" "$h3spec_executable" "$tool_manifest_path" "${match_values[@]}" <<'PY'
import json
import sys

path, host, port, timeout_ms, plan_only, status, exit_code, mode, tool_version, executable, tool_manifest, *matches = sys.argv[1:]
payload = {
    "executor": "h3spec-http3-qpack",
    "mode": mode,
    "tool": "h3spec",
    "toolVersion": tool_version,
    "executable": executable,
    "toolManifest": tool_manifest,
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

"$python_bin" "$script_dir/scripts/parse-h3spec-results.py" \
  --stdout "$stdout_path" \
  --stderr "$stderr_path" \
  --metadata "$metadata_path" \
  --json-output "$results_path" \
  --markdown-output "$report_path"

"$python_bin" - "$results_path" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    result = json.load(handle)

summary = result.get("summary", {})
payload = {
    "tool": "h3spec",
    "status": summary.get("status", "unknown"),
    "classification": summary.get("classification", "unknown"),
    "metrics": {
        "totalRequests": int(summary.get("selectedCases") or 0),
        "successfulRequests": int(summary.get("passedCases") or 0),
        "failedRequests": int(summary.get("failedCases") or 0),
    },
    "warnings": [
        f"h3spec classification={summary.get('classification', 'unknown')}",
        f"h3spec exitCode={summary.get('exitCode', '')}",
    ],
}
print(json.dumps(payload, separators=(",", ":")))
PY

if [[ "$fail_on_h3spec_failures" == "true" && "$plan_only" != "true" ]]; then
  "$python_bin" - "$results_path" "$report_path" <<'PY'
import json
import sys

results_path, report_path = sys.argv[1:]
with open(results_path, encoding="utf-8") as handle:
    result = json.load(handle)
summary = result["summary"]
if summary["classification"] in {"no-selected-cases", "tooling-failure"}:
    raise SystemExit(f"h3spec produced {summary['classification']}. See {report_path}.")
if int(summary["failedCases"]) > 0 or (summary["exitCode"] not in (0, None)):
    raise SystemExit(f"h3spec reported failures. See {report_path}.")
PY
fi

echo "h3spec executor artifacts written to $output_root" >&2
