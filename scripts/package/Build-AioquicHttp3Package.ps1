[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
$manifest = Get-Content (Join-Path $Root 'implementations/aioquic-http3/protocol-lab-package.json') -Raw | ConvertFrom-Json
if ($manifest.packageVersion -ne '0.3.3') { throw 'aioquic HTTP/3 target package version must be 0.3.3.' }
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath 'implementations/aioquic-http3' `
    -AllowDirtySource:$AllowDirtySource
