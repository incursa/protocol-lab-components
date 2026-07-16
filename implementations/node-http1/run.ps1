$ErrorActionPreference = 'Stop'
& node (Join-Path $PSScriptRoot 'server.js')
exit $LASTEXITCODE
