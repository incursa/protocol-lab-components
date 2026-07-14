[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
$manifest = Get-Content (Join-Path $Root 'scenarios/aioquic-rfc9220-websocket/protocol-lab-package.json') -Raw | ConvertFrom-Json
if ($manifest.packageVersion -ne '0.2.2') { throw 'RFC9220 scenario package version must be 0.2.2.' }
if (@($manifest.providedLoadProfiles).Count -ne 1 -or $manifest.providedLoadProfiles[0].loadProfileId -ne 'websocket-smoke') { throw 'RFC9220 scenario package must provide exactly websocket-smoke.' }
& (Join-Path $Root 'scenarios/aioquic-rfc9220-websocket/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath 'scenarios/aioquic-rfc9220-websocket' `
    -AllowDirtySource:$AllowDirtySource
