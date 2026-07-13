param([string]$Listen = "127.0.0.1:18532")
$ErrorActionPreference = 'Stop'
& (Join-Path $PSScriptRoot 'bin/win-x64/go-dns-doq.exe') --listen $Listen
exit $LASTEXITCODE
