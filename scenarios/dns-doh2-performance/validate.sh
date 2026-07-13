#!/usr/bin/env sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
pwsh -NoProfile -File "$root/validate.ps1"
