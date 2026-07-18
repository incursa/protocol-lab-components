[CmdletBinding()]
param([Parameter(Mandatory)][string]$SnapshotPath)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'ProtocolLabComponentRelease.Common.ps1')
$snapshot = Get-Content -LiteralPath $SnapshotPath -Raw | ConvertFrom-Json -AsHashtable
if ($snapshot.schemaVersion -ne 'protocol-lab.catalog-snapshot.v1') { throw "Unsupported catalog snapshot schema '$($snapshot.schemaVersion)'." }
if ($snapshot.recommendedTag -ne "catalog/$($snapshot.catalogVersion)") { throw 'Catalog snapshot tag does not match catalog version.' }
$claimed = [string]$snapshot.snapshotDigest
$snapshot.Remove('snapshotDigest')
$actual = Get-ProtocolLabSha256Text (ConvertTo-ProtocolLabCanonicalJson $snapshot)
if ($actual -ne $claimed) { throw "Catalog snapshot digest mismatch: expected $claimed, actual $actual." }
$seen = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
foreach ($entry in @($snapshot.entries)) {
    foreach ($field in @('packageId', 'packageVersion', 'artifactSha256', 'releaseTag', 'sourceCommit', 'componentTreeDigest', 'buildRecipeDigest', 'componentClosureDigest')) {
        if ([string]::IsNullOrWhiteSpace([string]$entry[$field])) { throw "Catalog entry is missing '$field'." }
    }
    if (-not $seen.Add("$($entry.packageId)@$($entry.packageVersion)")) { throw "Catalog snapshot duplicates package identity $($entry.packageId)@$($entry.packageVersion)." }
}
Write-Host "Validated immutable catalog snapshot $SnapshotPath ($claimed)."
