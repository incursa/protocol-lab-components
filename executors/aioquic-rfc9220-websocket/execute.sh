#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
scenario_id="${PLAB_SCENARIO_ID:-}"
target_url="${PLAB_TARGET_BASE_URL:-${PLAB_TARGET_URL:-https://host.docker.internal:4435/websocket-proof}}"
timeout_seconds="${PLAB_TIMEOUT_SECONDS:-20}"
image="${AIOQUIC_RFC9220_WEBSOCKET_IMAGE:-incursa-protocol-lab-aioquic-rfc9220-websocket:0.2.1}"
output_root="${PLAB_ARTIFACT_DIR:-${PLAB_OUTPUT_ROOT:-artifacts/aioquic-rfc9220-websocket}}"
skip_build="${PLAB_SKIP_BUILD:-0}"
plan_only="${PLAB_PLAN_ONLY:-0}"
docker_network="${PLAB_DOCKER_NETWORK:-}"
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --scenario-id) scenario_id="$2"; shift 2 ;;
    --target-url) target_url="$2"; shift 2 ;;
    --timeout) timeout_seconds="$2"; shift 2 ;;
    --output-root) output_root="$2"; shift 2 ;;
    --skip-build) skip_build=1; shift ;;
    --plan-only) plan_only=1; shift ;;
    --docker-network) docker_network="$2"; shift 2 ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done
if [[ "$output_root" != /* ]]; then output_root="$script_dir/$output_root"; fi
mkdir -p "$output_root/qlog" "$output_root/sslkeylog"

supported=(http3.websocket.rfc9220.extended-connect http3.websocket.rfc9220.control-frames http3.websocket.rfc9220.text-echo http3.websocket.rfc9220.binary-echo http3.websocket.rfc9220.close http3.websocket.rfc9220.fragmented-binary-echo)
unsupported=(websocket.echo http1.websocket.rfc6455.cleartext.upgrade http1.websocket.rfc6455.cleartext.control-frames http1.websocket.rfc6455.cleartext.text-echo http1.websocket.rfc6455.cleartext.binary-echo http1.websocket.rfc6455.cleartext.close http1.websocket.rfc6455.tls.upgrade http1.websocket.rfc6455.tls.control-frames http1.websocket.rfc6455.tls.text-echo http1.websocket.rfc6455.tls.binary-echo http1.websocket.rfc6455.tls.close http1.websocket.rfc6455.tls.subprotocol-text-echo http1.websocket.rfc6455.tls.permessage-deflate-binary-echo http2.websocket.rfc8441.extended-connect http2.websocket.rfc8441.control-frames http2.websocket.rfc8441.text-echo http2.websocket.rfc8441.binary-echo http2.websocket.rfc8441.close http2.websocket.rfc8441.multi-message-text-echo)
contains() { local needle="$1"; shift; local value; for value in "$@"; do [[ "$value" == "$needle" ]] && return 0; done; return 1; }
if contains "$scenario_id" "${unsupported[@]}"; then
  printf '{"schemaVersion":"protocol-lab.aioquic-rfc9220-result.v2","scenarioId":"%s","status":"unsupported"}\n' "$scenario_id" > "$output_root/result.json"
  exit 3
fi
if [[ -z "$scenario_id" ]] || ! contains "$scenario_id" "${supported[@]}"; then
  printf '{"schemaVersion":"protocol-lab.aioquic-rfc9220-result.v2","scenarioId":"%s","status":"unknown"}\n' "$scenario_id" > "$output_root/result.json"
  exit 2
fi

python_bin="$(command -v python3 || command -v python)"
target_url="$($python_bin - "$target_url" <<'PY'
import sys
from urllib.parse import urlsplit, urlunsplit
uri = urlsplit(sys.argv[1])
host = "host.docker.internal" if (uri.hostname or "").lower() in {"localhost", "127.0.0.1", "::1"} else uri.hostname
port = f":{uri.port}" if uri.port is not None else ""
path = uri.path if uri.path and uri.path != "/" else "/websocket-proof"
print(urlunsplit((uri.scheme, f"{host}{port}", path, uri.query, uri.fragment)))
PY
)"
command_file="$output_root/command.txt"
result_file="$output_root/result.json"
build_command=(docker build --build-arg AIOQUIC_VERSION=1.3.0 -f "$script_dir/docker/aioquic-rfc9220-websocket.Dockerfile" -t "$image" "$script_dir")
run_command=(docker run --rm --add-host=host.docker.internal:host-gateway)
if [[ -n "$docker_network" ]]; then run_command+=(--network "$docker_network"); fi
run_command+=(-v "$output_root:/proof" -e QLOGDIR=/proof/qlog -e SSLKEYLOGFILE=/proof/sslkeylog/keys.log "$image" /usr/local/bin/aioquic-http3-websocket-client "$target_url" /proof/client-result.json --scenario-id "$scenario_id" --timeout "$timeout_seconds")
: > "$command_file"
if [[ "$skip_build" != 1 ]]; then printf '%q ' "${build_command[@]}" >> "$command_file"; printf '\n' >> "$command_file"; fi
printf '%q ' "${run_command[@]}" >> "$command_file"; printf '\n' >> "$command_file"
if [[ "$plan_only" == 1 ]]; then
  printf '{"schemaVersion":"protocol-lab.aioquic-rfc9220-result.v2","scenarioId":"%s","status":"planned","image":"%s"}\n' "$scenario_id" "$image" > "$result_file"
  exit 0
fi
if [[ "$skip_build" != 1 ]]; then "${build_command[@]}" > "$output_root/build.stdout.log" 2> "$output_root/build.stderr.log"; fi
set +e
"${run_command[@]}" > "$output_root/load.stdout.log" 2> "$output_root/load.stderr.log"
exit_code=$?
set -e
if [[ "$exit_code" -ne 0 ]] || [[ ! -f "$output_root/client-result.json" ]]; then exit 1; fi
"$python_bin" - "$output_root/client-result.json" "$result_file" "$scenario_id" <<'PY'
import json, sys
client = json.load(open(sys.argv[1], encoding="utf-8"))
if client.get("status") != "passed" or client.get("scenarioId") != sys.argv[3]:
    raise SystemExit("client identity or validation mismatch")
result = {
    "schemaVersion": "protocol-lab.aioquic-rfc9220-result.v2",
    "scenarioId": sys.argv[3],
    "status": "passed",
    "authorityCommit": "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574",
    "executor": {"id": "aioquic-rfc9220-websocket", "version": "0.2.1"},
    "validation": {"status": "passed"},
    "protocolProof": client["protocolProof"],
    "metrics": client["metrics"],
    "warnings": ["Local package-backed RFC 9220 evidence is diagnostic and non-publishable."],
}
json.dump(result, open(sys.argv[2], "w", encoding="utf-8"), indent=2)
PY
printf '{"id":"aioquic-rfc9220-websocket","version":"0.2.1","aioquicVersion":"1.3.0","role":"test-executor"}\n' > "$output_root/executor-identity.json"
