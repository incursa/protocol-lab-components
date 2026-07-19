[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$ArtifactRoot = (Join-Path $Root 'artifacts/component-aware-release-tests')
)

$ErrorActionPreference = 'Stop'
$graphPath = Join-Path $Root 'release/component-graph.v1.json'
& (Join-Path $PSScriptRoot 'Test-ProtocolLabComponentReleaseGraph.ps1') -Root $Root -GraphPath $graphPath
& (Join-Path $PSScriptRoot 'Test-ProtocolLabReleaseIntents.ps1') -Root $Root -GraphPath $graphPath

function Get-ClosureDigest([string]$ComponentId) {
    return ((& (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentClosure.ps1') -Root $Root -GraphPath $graphPath -ComponentId $ComponentId | ConvertFrom-Json).componentClosureDigest)
}

$before = Get-ClosureDigest 'http2-performance-scenarios'
$unrelatedPath = Join-Path $Root 'docs/.component-aware-release-stability-test.txt'
Set-Content -LiteralPath $unrelatedPath -Value 'unrelated documentation test input' -Encoding utf8NoBOM
try { $after = Get-ClosureDigest 'http2-performance-scenarios' }
finally { Remove-Item -LiteralPath $unrelatedPath -Force -ErrorAction SilentlyContinue }
if ($before -ne $after) { throw 'An unrelated documentation file changed the modeled component closure digest.' }

$selection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/http2-performance/scenarios/http2/core/plaintext.yaml' | ConvertFrom-Json
if (@($selection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'apache-http2,caddy-http2,go-http2-executor,http2-performance-scenarios,kestrel-http2,nginx-http2') {
    throw 'Declared reverse-dependency selection did not include the complete modeled HTTP/2 cohort.'
}
$http1Selection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/http1-performance/scenarios/http1/core/plaintext.yaml' | ConvertFrom-Json
if (@($http1Selection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'apache-http1,caddy-http1,go-http1-executor,http1-performance-scenarios,kestrel-http1,nginx-http1') {
    throw 'Declared reverse-dependency selection did not include the complete modeled HTTP/1 cohort.'
}
$dnsClassicSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/dns-classic-calibration/scenarios/dns/classic/query-a-udp.yaml' | ConvertFrom-Json
if (@($dnsClassicSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'bind9-classic-authority,dns-classic-calibration,go-dns-classic-authority,go-dns-tcp-executor,go-dns-udp-executor,technitium-classic-authority') {
    throw 'Declared reverse-dependency selection did not include the complete modeled classic DNS cohort.'
}
$unknown = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'unmodeled-release-input.txt' | ConvertFrom-Json
if (-not $unknown.fullBuildDryRunRequired) { throw 'Unknown changes must require conservative full-build dry-run.' }

Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $ArtifactRoot | Out-Null
$packageOutput = Join-Path $ArtifactRoot 'packages'
& (Join-Path $PSScriptRoot 'Build-Http2PerformanceScenarioPackage.ps1') -Root $Root -OutputRoot $packageOutput -AllowDirtySource
$packages = @(Get-ChildItem -LiteralPath $packageOutput -File -Filter '*.plabpkg')
if ($packages.Count -ne 1) { throw "Expected exactly one scenario package, found $($packages.Count)." }
$package = $packages[0]
$attestation = Get-Item -LiteralPath "$($package.FullName).build-attestation.json"
& (Join-Path $PSScriptRoot 'Test-ProtocolLabPackageBuildAttestation.ps1') -PackagePath $package.FullName -AttestationPath $attestation.FullName

$temporaryPayload = Join-Path $Root 'scenarios/http2-performance/.component-aware-release-immutable-test.txt'
Set-Content -LiteralPath $temporaryPayload -Value 'must change package closure' -Encoding utf8NoBOM
try {
    $collision = $null
    try { & (Join-Path $PSScriptRoot 'Build-Http2PerformanceScenarioPackage.ps1') -Root $Root -OutputRoot $packageOutput -AllowDirtySource }
    catch { $collision = $_ }
    if ($null -eq $collision -or $collision.Exception.Message -notmatch 'Immutable package collision') { throw 'Changed package bytes did not require a version advance.' }
}
finally { Remove-Item -LiteralPath $temporaryPayload -Force -ErrorAction SilentlyContinue }

$snapshot = Join-Path $ArtifactRoot 'catalog-2026.07.17-test.json'
& (Join-Path $PSScriptRoot 'New-ProtocolLabCatalogSnapshot.ps1') -CatalogVersion '2026.07.17-test' -PackagePath $package.FullName -AttestationPath $attestation.FullName -ReleaseTag 'packages/http2-performance-scenarios/v0.2.2' -OutputPath $snapshot
& (Join-Path $PSScriptRoot 'Test-ProtocolLabCatalogSnapshot.ps1') -SnapshotPath $snapshot

$invalidIntentRoot = Join-Path $ArtifactRoot 'invalid-intents'
New-Item -ItemType Directory -Force -Path $invalidIntentRoot | Out-Null
'{ "schemaVersion": "protocol-lab.release-intent.v1", "id": "invalid", "classification": "no-release", "status": "approved", "components": ["forbidden"], "reason": "test" }' | Set-Content -LiteralPath (Join-Path $invalidIntentRoot 'invalid.json') -Encoding utf8NoBOM
$intentError = $null
try { & (Join-Path $PSScriptRoot 'Test-ProtocolLabReleaseIntents.ps1') -Root $Root -GraphPath $graphPath -IntentRoot $invalidIntentRoot }
catch { $intentError = $_ }
if ($null -eq $intentError -or $intentError.Exception.Message -notmatch 'no-release intents') { throw 'Invalid no-release intent was accepted.' }

[ordered]@{
    schemaVersion = 'protocol-lab.component-aware-release-tests.v1'
    status = 'passed'
    unaffectedPackageClosureStable = $true
    reverseDependencySelection = $true
    immutableVersionEnforcement = $true
    releaseIntentEnforcement = $true
    catalogSnapshotReproducibility = $true
} | ConvertTo-Json
