[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
$manifest = Get-Content (Join-Path $Root 'scenarios/tls13-handshake-performance/protocol-lab-package.json') -Raw | ConvertFrom-Json
if ($manifest.packageVersion -ne '0.2.2') { throw 'TLS scenario package version must be 0.2.2.' }
if ((@($manifest.providedSuites.suiteId) -join ',') -ne 'tls-performance-smoke,tls-contract-breadth-smoke,tls-security-diagnostics') { throw 'TLS scenario package providedSuites mismatch.' }
& (Join-Path $Root 'scenarios/tls13-handshake-performance/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -ComponentPath 'scenarios/tls13-handshake-performance' `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -AllowDirtySource:$AllowDirtySource
