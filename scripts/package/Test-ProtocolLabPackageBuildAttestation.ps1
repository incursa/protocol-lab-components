[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$PackagePath,
    [Parameter(Mandatory)][string]$AttestationPath,
    [switch]$RequireParityEligible
)

$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $PackagePath -PathType Leaf)) {
    throw "Package artifact not found: $PackagePath"
}

if (-not (Test-Path -LiteralPath $AttestationPath -PathType Leaf)) {
    throw "Package build attestation not found: $AttestationPath"
}

$attestation = Get-Content -LiteralPath $AttestationPath -Raw | ConvertFrom-Json
if ($attestation.schemaVersion -ne 'protocol-lab.package-build-attestation.v1') {
    throw "Unsupported package build attestation schema '$($attestation.schemaVersion)'."
}

$requiredValues = [ordered]@{
    'source.repository' = [string]$attestation.source.repository
    'source.commitSha' = [string]$attestation.source.commitSha
    'source.dirtyState' = [string]$attestation.source.dirtyState
    'build.configuration' = [string]$attestation.build.configuration
    'build.runtimeIdentifier' = [string]$attestation.build.runtimeIdentifier
    'package.packageId' = [string]$attestation.package.packageId
    'package.packageVersion' = [string]$attestation.package.packageVersion
    'package.sha256' = [string]$attestation.package.sha256
    'package.materializationPath' = [string]$attestation.package.materializationPath
}

foreach ($entry in $requiredValues.GetEnumerator()) {
    if ([string]::IsNullOrWhiteSpace($entry.Value)) {
        throw "Package build attestation is missing required value '$($entry.Key)'."
    }
}

if ([string]$attestation.source.commitSha -notmatch '^[0-9a-fA-F]{40,64}$') {
    throw "Package source commit SHA is malformed."
}

if ([string]$attestation.package.sha256 -notmatch '^[0-9a-fA-F]{64}$') {
    throw "Package SHA-256 is malformed."
}

$actualPackagePath = [System.IO.Path]::GetFullPath((Resolve-Path -LiteralPath $PackagePath).Path)
$recordedPackagePath = [System.IO.Path]::GetFullPath([string]$attestation.package.materializationPath)
if (-not [string]::Equals($actualPackagePath, $recordedPackagePath, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Package materialization path does not match the attestation."
}

$actualHash = (Get-FileHash -LiteralPath $actualPackagePath -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actualHash -ne ([string]$attestation.package.sha256).ToLowerInvariant()) {
    throw "Package SHA-256 does not match the build attestation."
}

if ($RequireParityEligible -and $attestation.parityEligible -ne $true) {
    throw "Package build attestation is not parity eligible. Source dirty state: $($attestation.source.dirtyState)."
}

Write-Host "Validated package build attestation for $($attestation.package.packageId) $($attestation.package.packageVersion) ($actualHash)."
