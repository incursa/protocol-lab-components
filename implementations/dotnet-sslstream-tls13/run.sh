#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PLAB_TLS_PORT="${PLAB_TLS_PORT:-${PORT:-8443}}"
export PLAB_TLS_CERTIFICATE_PATH="$script_dir/certs/leaf.pem"
export PLAB_TLS_PRIVATE_KEY_PATH="$script_dir/certs/leaf-key.pem"
dotnet run --project "$script_dir/src/DotNetSslStreamTls13.csproj" --configuration "${CONFIGURATION:-Release}"
