[CmdletBinding()]
param(
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference='Stop'
& (Join-Path $PSScriptRoot 'Test-TlsEndpointToolPackages.ps1') -Root $Root
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath 'implementations/openssl-s-server' `
    -RuntimeIdentifier linux-x64 `
    -IncludeReadme `
    -AllowDirtySource:$AllowDirtySource
