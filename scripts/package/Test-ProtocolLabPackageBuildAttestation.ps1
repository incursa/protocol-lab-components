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

Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [System.IO.Compression.ZipFile]::OpenRead($actualPackagePath)
try {
    $entry = $archive.Entries | Where-Object FullName -eq 'package-build-provenance.json' | Select-Object -First 1
    if ($null -eq $entry) {
        throw 'Package does not contain package-build-provenance.json.'
    }

    $reader = [System.IO.StreamReader]::new($entry.Open())
    try { $embedded = $reader.ReadToEnd() | ConvertFrom-Json } finally { $reader.Dispose() }
    if ($embedded.schemaVersion -eq 'protocol-lab.package-build-provenance.v1') {
        if ([string]$embedded.source.repository -ne [string]$attestation.source.repository -or
            [string]$embedded.source.commitSha -ne [string]$attestation.source.commitSha -or
            [string]$embedded.source.dirtyState -ne [string]$attestation.source.dirtyState -or
            [string]$embedded.build.configuration -ne [string]$attestation.build.configuration -or
            [string]$embedded.build.runtimeIdentifier -ne [string]$attestation.build.runtimeIdentifier -or
            [string]$embedded.package.packageId -ne [string]$attestation.package.packageId -or
            [string]$embedded.package.packageVersion -ne [string]$attestation.package.packageVersion) {
            throw 'Embedded package build provenance does not match the external build attestation.'
        }
    }
    elseif ($embedded.schemaVersion -eq 'protocol-lab.package-build-provenance.v2') {
        foreach ($field in @('id', 'componentTreeDigest', 'buildRecipeDigest', 'componentClosureDigest')) {
            if ([string]::IsNullOrWhiteSpace([string]$embedded.component.$field) -or
                [string]$embedded.component.$field -ne [string]$attestation.component.$field) {
                throw "Embedded component closure provenance '$field' does not match the external build attestation."
            }
        }
        if ([string]$embedded.build.configuration -ne [string]$attestation.build.configuration -or
            [string]$embedded.build.runtimeIdentifier -ne [string]$attestation.build.runtimeIdentifier -or
            [string]$embedded.package.packageId -ne [string]$attestation.package.packageId -or
            [string]$embedded.package.packageVersion -ne [string]$attestation.package.packageVersion) {
            throw 'Embedded component package identity does not match the external build attestation.'
        }
    }
    else {
        throw "Unsupported embedded package build provenance schema '$($embedded.schemaVersion)'."
    }
}
finally {
    $archive.Dispose()
}

if ($RequireParityEligible -and $attestation.parityEligible -ne $true) {
    throw "Package build attestation is not parity eligible. Source dirty state: $($attestation.source.dirtyState)."
}

Write-Host "Validated package build attestation for $($attestation.package.packageId) $($attestation.package.packageVersion) ($actualHash)."
