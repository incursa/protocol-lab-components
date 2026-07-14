[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
$manifest = Get-Content (Join-Path $Root 'scenarios/aioquic-rfc9220-websocket/protocol-lab-package.json') -Raw | ConvertFrom-Json
if ($manifest.packageVersion -ne '0.2.2') { throw 'RFC9220 scenario package version must be 0.2.2.' }
if ($manifest.packageId -ne 'org.protocol-lab.components.scenario.http3-websocket-performance') { throw 'RFC9220 scenario package identity mismatch.' }
if ((@($manifest.providedLoadProfiles.loadProfileId) -join ',') -ne 'websocket-smoke,diagnostic') { throw 'RFC9220 scenario package load-profile declarations mismatch.' }
if ((@($manifest.providedSuites.suiteId) -join ',') -ne 'aioquic-rfc9220-websocket-proof,aioquic-rfc9220-websocket-fragmentation-diagnostic') { throw 'RFC9220 scenario package suite declarations mismatch.' }
& (Join-Path $Root 'scenarios/aioquic-rfc9220-websocket/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath 'scenarios/aioquic-rfc9220-websocket' `
    -AllowDirtySource:$AllowDirtySource
