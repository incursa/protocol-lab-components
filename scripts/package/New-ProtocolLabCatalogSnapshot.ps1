[CmdletBinding()]
param(
    [Parameter(Mandatory)][ValidatePattern('^\d{4}\.\d{2}\.\d{2}([-.][0-9A-Za-z.-]+)?$')][string]$CatalogVersion,
    [Parameter(Mandatory)][string[]]$PackagePath,
    [Parameter(Mandatory)][string[]]$AttestationPath,
    [Parameter(Mandatory)][string[]]$ReleaseTag,
    [Parameter(Mandatory)][string]$OutputPath
)

$ErrorActionPreference = 'Stop'
if ($PackagePath.Count -ne $AttestationPath.Count -or $PackagePath.Count -ne $ReleaseTag.Count) {
    throw 'PackagePath, AttestationPath, and ReleaseTag must have the same number of entries.'
}

. (Join-Path $PSScriptRoot 'ProtocolLabComponentRelease.Common.ps1')
$entries = [System.Collections.Generic.List[object]]::new()
Add-Type -AssemblyName System.IO.Compression.FileSystem
for ($index = 0; $index -lt $PackagePath.Count; $index++) {
    & (Join-Path $PSScriptRoot 'Test-ProtocolLabPackageBuildAttestation.ps1') -PackagePath $PackagePath[$index] -AttestationPath $AttestationPath[$index]
    $attestation = Get-Content -LiteralPath $AttestationPath[$index] -Raw | ConvertFrom-Json
    if ($null -eq $attestation.component -or [string]::IsNullOrWhiteSpace([string]$attestation.component.componentClosureDigest)) {
        throw "Catalog snapshots require component-closure provenance: $($PackagePath[$index])"
    }
    $entries.Add([ordered]@{
        packageId = [string]$attestation.package.packageId
        packageVersion = [string]$attestation.package.packageVersion
        artifactSha256 = [string]$attestation.package.sha256
        releaseTag = [string]$ReleaseTag[$index]
        sourceCommit = [string]$attestation.source.commitSha
        componentTreeDigest = [string]$attestation.component.componentTreeDigest
        buildRecipeDigest = [string]$attestation.component.buildRecipeDigest
        componentClosureDigest = [string]$attestation.component.componentClosureDigest
        contracts = @($attestation.component.contracts)
        toolchains = @($attestation.component.toolchains)
    })
}

$sortedEntries = @($entries | Sort-Object packageId, packageVersion, artifactSha256)
$content = [ordered]@{
    schemaVersion = 'protocol-lab.catalog-snapshot.v1'
    catalogVersion = $CatalogVersion
    recommendedTag = "catalog/$CatalogVersion"
    entries = $sortedEntries
}
$content.snapshotDigest = Get-ProtocolLabSha256Text (ConvertTo-ProtocolLabCanonicalJson $content)
$directory = Split-Path -Parent $OutputPath
if ($directory) { New-Item -ItemType Directory -Force -Path $directory | Out-Null }
if (Test-Path -LiteralPath $OutputPath) { throw "Catalog snapshot paths are immutable and must not be overwritten: $OutputPath" }
$content | ConvertTo-Json -Depth 32 | Set-Content -LiteralPath $OutputPath -Encoding utf8NoBOM
Write-Host "Created catalog snapshot $OutputPath ($($content.snapshotDigest))."
