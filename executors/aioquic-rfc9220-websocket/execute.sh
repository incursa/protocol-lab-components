#!/usr/bin/env bash
set -euo pipefail

component_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
scenario_id="${PLAB_SCENARIO_ID:-}"
profile_id="${PLAB_LOAD_PROFILE_ID:-}"
target_url="${PLAB_TARGET_BASE_URL:-https://host.docker.internal:4435/websocket-proof}"
output_root="${PLAB_ARTIFACT_DIR:-$component_root/artifacts/aioquic-rfc9220-websocket}"
image="${AIOQUIC_RFC9220_WEBSOCKET_IMAGE:-incursa-protocol-lab-aioquic-rfc9220-websocket:0.3.0}"
mkdir -p "$output_root/qlog" "$output_root/sslkeylog"

case "$scenario_id" in
  http3.websocket.rfc9220.fragmented-binary-echo) expected_profile=diagnostic; concurrency=8; duration=10; cooldown=1; timeout=10 ;;
  http3.websocket.rfc9220.extended-connect|http3.websocket.rfc9220.control-frames|http3.websocket.rfc9220.text-echo|http3.websocket.rfc9220.binary-echo|http3.websocket.rfc9220.close) expected_profile=websocket-smoke; concurrency=1; duration=5; cooldown=0; timeout=5 ;;
  websocket.echo|http1.websocket.*|http2.websocket.*) printf '{"schemaVersion":"protocol-lab.rfc9220-executor-result.v1","executorId":"aioquic-rfc9220-websocket","executorVersion":"0.3.0","scenarioId":"%s","status":"unsupported","passed":false}\n' "$scenario_id" | tee "$output_root/result.json"; exit 3 ;;
  *) printf '{"schemaVersion":"protocol-lab.rfc9220-executor-result.v1","executorId":"aioquic-rfc9220-websocket","executorVersion":"0.3.0","scenarioId":"%s","status":"unknown","passed":false}\n' "$scenario_id" | tee "$output_root/result.json"; exit 2 ;;
esac
profile_id="${profile_id:-$expected_profile}"
[[ "$profile_id" == "$expected_profile" ]] || { printf 'exact load profile mismatch\n' >&2; exit 3; }
for name in PLAB_SCENARIO_PACKAGE_SHA256 PLAB_EXECUTOR_PACKAGE_SHA256 PLAB_TARGET_PACKAGE_SHA256; do [[ "${!name:-}" =~ ^[0-9a-fA-F]{64}$ ]] || { printf '%s is required\n' "$name" >&2; exit 1; }; done
[[ "${PLAB_TARGET_IMAGE_ID:-}" =~ ^sha256:[0-9a-fA-F]{64}$ ]] || { printf 'PLAB_TARGET_IMAGE_ID is required\n' >&2; exit 1; }
executor_image_id="$(docker image inspect --format '{{.Id}}' "$image")"
args=(run --rm --add-host=host.docker.internal:host-gateway -v "$output_root:/proof" -e QLOGDIR=/proof/qlog -e SSLKEYLOGFILE=/proof/sslkeylog/keys.log "$image" /usr/local/bin/aioquic-http3-websocket-client "$target_url" /proof/client-result.json --scenario-id "$scenario_id" --load-profile-id "$profile_id" --concurrency "$concurrency" --warmup 1 --duration "$duration" --cooldown "$cooldown" --timeout "$timeout" --scenario-package-sha256 "$PLAB_SCENARIO_PACKAGE_SHA256" --executor-package-sha256 "$PLAB_EXECUTOR_PACKAGE_SHA256" --target-package-sha256 "$PLAB_TARGET_PACKAGE_SHA256" --executor-image-id "$executor_image_id" --target-image-id "$PLAB_TARGET_IMAGE_ID")
printf 'docker %q ' "${args[@]}" > "$output_root/command.txt"
docker "${args[@]}" > "$output_root/load.stdout.log" 2> "$output_root/load.stderr.log"
cp "$output_root/client-result.json" "$output_root/result.json"
cat "$output_root/result.json"
