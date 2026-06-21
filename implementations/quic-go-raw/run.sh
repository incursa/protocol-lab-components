#!/usr/bin/env bash
set -euo pipefail

port="${PLAB_QUIC_PORT:-5447}"
plan_only="${PLAB_PLAN_ONLY:-0}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/quic-go-raw}"

mkdir -p "$artifact_root"
command="bin/linux-x64/quic-go-raw"
printf '%s\n' "$command" > "$artifact_root/command.txt"

if [ "$plan_only" = "1" ]; then
  printf '{"status":"planned","port":%s,"command":"%s"}\n' "$port" "$command" > "$artifact_root/result.json"
  echo "Planned quic-go raw QUIC command at $artifact_root/command.txt"
  exit 0
fi

export PLAB_QUIC_PORT="$port"
exec "$command"
