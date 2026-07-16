[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'))
$ErrorActionPreference='Stop'
& node --check (Join-Path $Root 'implementations/node-http2/server.js')
if($LASTEXITCODE-ne 0){throw 'node-http2 syntax validation failed.'}
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath 'implementations/node-http2'
