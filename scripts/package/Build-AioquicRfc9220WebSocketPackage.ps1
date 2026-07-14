[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
$manifest = Get-Content (Join-Path $Root 'executors/aioquic-rfc9220-websocket/protocol-lab-package.json') -Raw | ConvertFrom-Json
if ($manifest.packageVersion -ne '0.3.0') { throw 'RFC9220 executor package version must be 0.3.0.' }
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath 'executors/aioquic-rfc9220-websocket' `
    -AllowDirtySource:$AllowDirtySource
