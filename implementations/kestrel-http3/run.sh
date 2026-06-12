#!/usr/bin/env bash
set -euo pipefail

export PLAB_HTTP_PORT="${PLAB_HTTP_PORT:-8443}"
cd "$(dirname "${BASH_SOURCE[0]}")"
dotnet run --project src/KestrelHttp3.csproj
