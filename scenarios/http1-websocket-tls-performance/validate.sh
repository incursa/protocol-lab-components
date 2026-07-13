#!/usr/bin/env bash
set -euo pipefail
pwsh -NoLogo -NoProfile -File "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/validate.ps1"
