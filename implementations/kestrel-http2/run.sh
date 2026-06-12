#!/usr/bin/env bash
set -euo pipefail

port="${PLAB_HTTP_PORT:-${PORT:-8082}}"
configuration="${CONFIGURATION:-Release}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export PLAB_HTTP_PORT="$port"
dotnet run --project "$script_dir/src/KestrelHttp2.csproj" --configuration "$configuration" --no-launch-profile
