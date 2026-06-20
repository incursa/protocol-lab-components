#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
target_url="${PLAB_TARGET_URL:-https://host.docker.internal:4435/websocket-proof}"
timeout_seconds="${PLAB_TIMEOUT_SECONDS:-20}"
image="${AIOQUIC_RFC9220_WEBSOCKET_IMAGE:-incursa-protocol-lab-aioquic-rfc9220-websocket:0.1.7}"
output_root="${PLAB_OUTPUT_ROOT:-artifacts/aioquic-rfc9220-websocket}"
skip_build="${PLAB_SKIP_BUILD:-0}"
plan_only="${PLAB_PLAN_ONLY:-0}"
docker_network="${PLAB_DOCKER_NETWORK:-}"

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    http://*|https://*) target_url="$1"; shift ;;
    --target-url) target_url="$2"; shift 2 ;;
    --timeout) timeout_seconds="$2"; shift 2 ;;
    --output-root) output_root="$2"; shift 2 ;;
    --skip-build) skip_build="1"; shift ;;
    --plan-only) plan_only="1"; shift ;;
    --docker-network) docker_network="$2"; shift 2 ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done

python_bin="$(command -v python3 || command -v python)"
target_url="$("$python_bin" - "$target_url" <<'PY'
import sys
from urllib.parse import urlsplit, urlunsplit

uri = urlsplit(sys.argv[1])
path = uri.path if uri.path and uri.path != "/" else "/websocket-proof"
host = uri.hostname or ""
port = f":{uri.port}" if uri.port is not None else ""
if host.lower() in {"localhost", "127.0.0.1", "::1"}:
    host = "host.docker.internal"
netloc = f"[{host}]{port}" if ":" in host and not host.startswith("[") else f"{host}{port}"
print(urlunsplit((uri.scheme, netloc, path, uri.query, uri.fragment)))
PY
)"

if [[ "$output_root" != /* ]]; then
  output_root="$script_dir/$output_root"
fi

mkdir -p "$output_root/qlog" "$output_root/sslkeylog"
command_file="$output_root/command.txt"
result_file="$output_root/result.json"
build_stdout="$output_root/build.stdout.txt"
build_stderr="$output_root/build.stderr.txt"

build_command=(docker build --build-arg AIOQUIC_VERSION=1.3.0 -f "$script_dir/docker/aioquic-rfc9220-websocket.Dockerfile" -t "$image" "$script_dir")
run_command=(
  docker run --rm
  --add-host=host.docker.internal:host-gateway
)
if [ -n "$docker_network" ]; then
  run_command+=(--network "$docker_network")
fi
run_command+=(
  -v "$output_root:/proof"
  -e QLOGDIR=/proof/qlog
  -e SSLKEYLOGFILE=/proof/sslkeylog/keys.log
  "$image"
  /usr/local/bin/aioquic-http3-websocket-client
  "$target_url"
  /proof/client-result.json
  --timeout "$timeout_seconds"
)

: > "$command_file"
if [ "$skip_build" != "1" ]; then
  printf '%q ' "${build_command[@]}" >> "$command_file"
  printf '\n' >> "$command_file"
fi
printf '%q ' "${run_command[@]}" >> "$command_file"
printf '\n' >> "$command_file"

if [ "$plan_only" = "1" ]; then
  cat > "$result_file" <<EOF
{"status":"planned","targetUrl":"$target_url","image":"$image"}
EOF
  "$python_bin" - "$result_file" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    payload = json.load(handle)

payload["tool"] = "aioquic-rfc9220-websocket"
payload["metrics"] = {"totalRequests": 0, "successfulRequests": 0, "failedRequests": 0}
with open(sys.argv[1], "w", encoding="utf-8") as handle:
    json.dump(payload, handle, separators=(",", ":"))
PY
  cat "$result_file"
  echo "Planned aioquic RFC9220 WebSocket executor command at $command_file" >&2
  exit 0
fi

if [ "$skip_build" != "1" ]; then
  "${build_command[@]}" > "$build_stdout" 2> "$build_stderr"
fi

set +e
"${run_command[@]}" > "$output_root/stdout.txt" 2> "$output_root/stderr.txt"
exit_code=$?
set -e

client_status="missing"
if [ -f "$output_root/client-result.json" ]; then
  client_status="$("$python_bin" -c 'import json,sys; print(json.load(open(sys.argv[1])).get("status", "missing"))' "$output_root/client-result.json")"
fi

if [ "$exit_code" -eq 0 ] && [ "$client_status" = "passed" ]; then
  cat > "$result_file" <<EOF
{"status":"passed","targetUrl":"$target_url","image":"$image","exitCode":$exit_code}
EOF
  "$python_bin" - "$result_file" "$output_root/client-result.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    payload = json.load(handle)
with open(sys.argv[2], encoding="utf-8") as handle:
    client = json.load(handle)

payload["tool"] = "aioquic-rfc9220-websocket"
payload["evidenceClass"] = client.get("evidenceClass", "local-external-aioquic-peer")
payload["statusCode"] = client.get("statusCode")
payload["proofScope"] = client.get("proofScope", [])
payload["metrics"] = {"totalRequests": 1, "successfulRequests": 1, "failedRequests": 0}
with open(sys.argv[1], "w", encoding="utf-8") as handle:
    json.dump(payload, handle, separators=(",", ":"))
PY
  cat "$result_file"
  echo "aioquic RFC9220 WebSocket executor passed for $target_url" >&2
  exit 0
fi

cat > "$result_file" <<EOF
{"status":"failed","targetUrl":"$target_url","image":"$image","exitCode":$exit_code,"clientStatus":"$client_status"}
EOF
"$python_bin" - "$result_file" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    payload = json.load(handle)

payload["tool"] = "aioquic-rfc9220-websocket"
payload["metrics"] = {"totalRequests": 1, "successfulRequests": 0, "failedRequests": 1}
payload["errors"] = [f"clientStatus={payload.get('clientStatus')}", f"exitCode={payload.get('exitCode')}"]
with open(sys.argv[1], "w", encoding="utf-8") as handle:
    json.dump(payload, handle, separators=(",", ":"))
PY
cat "$result_file"
exit 1
