[CmdletBinding()]
param([int]$Port = 18444)
$ErrorActionPreference = 'Stop'
$env:PLAB_LISTEN_ADDRESS = "127.0.0.1:$Port"
Push-Location (Join-Path $PSScriptRoot 'source')
try { & go run .; exit $LASTEXITCODE } finally { Pop-Location }
