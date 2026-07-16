#!/usr/bin/env bash
set -euo pipefail

port="${PLAB_QUIC_PORT:-5450}"
plan_only="${PLAB_PLAN_ONLY:-0}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/picoquic-raw}"

mkdir -p "$artifact_root"
command="bin/linux-x64/picoquic-raw"
printf '%s\n' "$command" > "$artifact_root/command.txt"

if [ "$plan_only" = "1" ]; then
  printf '{"status":"planned","port":%s,"command":"%s"}\n' "$port" "$command" > "$artifact_root/result.json"
  echo "Planned picoquic raw QUIC command at $artifact_root/command.txt"
  exit 0
fi

export PLAB_QUIC_PORT="$port"
export PLAB_CERT_FILE="${PLAB_CERT_FILE:-certs/cert.pem}"
export PLAB_KEY_FILE="${PLAB_KEY_FILE:-certs/key.pem}"
export LD_LIBRARY_PATH="${PWD}/lib${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}"
exec "$command"
