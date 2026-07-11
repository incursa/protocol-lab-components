[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$ComponentPath,

    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),

    [string]$BuildConfiguration = 'Release',

    [string]$RuntimeIdentifier = 'portable',

    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'

function Invoke-GitText {
    param(
        [Parameter(Mandatory)][string]$RepositoryRoot,
        [Parameter(Mandatory)][string[]]$Arguments,
        [switch]$AllowEmpty
    )

    $output = & git -C $RepositoryRoot @Arguments 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Git command failed while capturing package source provenance: git $($Arguments -join ' ')"
    }

    $text = ($output -join "`n").Trim()
    if (-not $AllowEmpty -and [string]::IsNullOrWhiteSpace($text)) {
        throw "Git command returned no source provenance: git $($Arguments -join ' ')"
    }

    return $text
}

function Get-OptionalToolVersion {
    param([Parameter(Mandatory)][string]$Name, [Parameter(Mandatory)][string[]]$Arguments)

    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($null -eq $command) {
        return $null
    }

    $output = & $command.Source @Arguments 2>$null
    if ($LASTEXITCODE -ne 0) {
        return $null
    }

    return (($output -join "`n").Trim())
}

$componentRoot = if ([System.IO.Path]::IsPathRooted($ComponentPath)) {
    $ComponentPath
}
else {
    Join-Path $Root $ComponentPath
}

$componentRoot = (Resolve-Path $componentRoot).Path
$repositoryRoot = (Resolve-Path $Root).Path
$sourceRepository = Invoke-GitText -RepositoryRoot $repositoryRoot -Arguments @('config', '--get', 'remote.origin.url')
$sourceCommit = Invoke-GitText -RepositoryRoot $repositoryRoot -Arguments @('rev-parse', 'HEAD')
$sourceStatus = Invoke-GitText -RepositoryRoot $repositoryRoot -Arguments @('status', '--porcelain=v1', '--untracked-files=normal') -AllowEmpty
$workingTreeClean = [string]::IsNullOrWhiteSpace($sourceStatus)
if (-not $workingTreeClean -and -not $AllowDirtySource) {
    throw "Package builds require a clean source worktree. Commit or remove local changes, or use -AllowDirtySource for diagnostic-only output."
}

$packageManifestPath = Join-Path $componentRoot 'protocol-lab-package.json'
if (-not (Test-Path -LiteralPath $packageManifestPath)) {
    throw "Component package root must contain protocol-lab-package.json: $componentRoot"
}

$packageManifest = Get-Content -LiteralPath $packageManifestPath -Raw | ConvertFrom-Json
$packageId = [string]$packageManifest.packageId
$packageVersion = [string]$packageManifest.packageVersion
if ([string]::IsNullOrWhiteSpace($packageId) -or [string]::IsNullOrWhiteSpace($packageVersion)) {
    throw "protocol-lab-package.json must declare packageId and packageVersion."
}

New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null
$stagingRoot = Join-Path $OutputRoot ("stage/" + ($packageId -replace '[^A-Za-z0-9_.-]', '_'))
$artifactPath = Join-Path $OutputRoot "$packageId.$packageVersion.plabpkg"
$candidateArtifactPath = "$artifactPath.building"
$attestationPath = "$artifactPath.build-attestation.json"

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $stagingRoot | Out-Null

$excludedNames = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
@(
    'README.md',
    'package.protocol-lab.json',
    'bin',
    'obj',
    'artifacts',
    'packages'
) | ForEach-Object { [void]$excludedNames.Add($_) }

Get-ChildItem -LiteralPath $componentRoot -Recurse -File -Force | Where-Object {
    $relativePath = [System.IO.Path]::GetRelativePath($componentRoot, $_.FullName)
    $pathParts = $relativePath -split '[\\/]'
    -not ($pathParts | Where-Object { $excludedNames.Contains($_) })
} | ForEach-Object {
    $relativePath = [System.IO.Path]::GetRelativePath($componentRoot, $_.FullName)
    $destinationPath = Join-Path $stagingRoot $relativePath
    $destinationDirectory = Split-Path -Parent $destinationPath
    New-Item -ItemType Directory -Force -Path $destinationDirectory | Out-Null
    Copy-Item -LiteralPath $_.FullName -Destination $destinationPath -Force
}

Remove-Item -LiteralPath $candidateArtifactPath -Force -ErrorAction SilentlyContinue
Compress-Archive -Path (Join-Path $stagingRoot '*') -DestinationPath $candidateArtifactPath -Force

$candidateHash = (Get-FileHash -LiteralPath $candidateArtifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
if (Test-Path -LiteralPath $artifactPath -PathType Leaf) {
    $existingHash = (Get-FileHash -LiteralPath $artifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($existingHash -ne $candidateHash) {
        Remove-Item -LiteralPath $candidateArtifactPath -Force
        throw "Immutable package collision for '$packageId' '$packageVersion': existing SHA-256 $existingHash differs from candidate $candidateHash. Increment packageVersion or choose an empty output root."
    }

    Remove-Item -LiteralPath $candidateArtifactPath -Force
}
else {
    Move-Item -LiteralPath $candidateArtifactPath -Destination $artifactPath
}

$materializedHash = (Get-FileHash -LiteralPath $artifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
$attestation = [ordered]@{
    schemaVersion = 'protocol-lab.package-build-attestation.v1'
    generatedAtUtc = [DateTimeOffset]::UtcNow.ToString('O')
    parityEligible = $workingTreeClean
    source = [ordered]@{
        repository = $sourceRepository
        commitSha = $sourceCommit
        workingTreeClean = $workingTreeClean
        dirtyState = if ($workingTreeClean) { 'clean' } else { 'dirty' }
        dirtyEntries = if ($workingTreeClean) { @() } else { @($sourceStatus -split "`n" | Where-Object { $_ }) }
        componentPath = [System.IO.Path]::GetRelativePath($repositoryRoot, $componentRoot).Replace('\', '/')
    }
    build = [ordered]@{
        configuration = $BuildConfiguration
        runtimeIdentifier = $RuntimeIdentifier
        operatingSystem = [System.Runtime.InteropServices.RuntimeInformation]::OSDescription
        processArchitecture = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString()
        powershell = $PSVersionTable.PSVersion.ToString()
        dotnet = Get-OptionalToolVersion -Name 'dotnet' -Arguments @('--version')
        go = Get-OptionalToolVersion -Name 'go' -Arguments @('version')
    }
    package = [ordered]@{
        packageId = $packageId
        packageVersion = $packageVersion
        sha256 = $materializedHash
        materializationPath = [System.IO.Path]::GetFullPath($artifactPath)
        buildAttestationPath = [System.IO.Path]::GetFullPath($attestationPath)
        immutableIdentity = "$packageId@$packageVersion#$materializedHash"
    }
    claimBoundary = if ($workingTreeClean) {
        'This attestation identifies one retained immutable package artifact built from the recorded clean commit. It does not claim byte-reproducible rebuilds.'
    }
    else {
        'Diagnostic-only dirty-source build. This artifact is not eligible for source/package parity or publication.'
    }
}

if (Test-Path -LiteralPath $attestationPath -PathType Leaf) {
    $existingAttestation = Get-Content -LiteralPath $attestationPath -Raw | ConvertFrom-Json
    if ([string]$existingAttestation.package.sha256 -ne $materializedHash -or
        [string]$existingAttestation.source.commitSha -ne $sourceCommit -or
        [string]$existingAttestation.package.packageId -ne $packageId -or
        [string]$existingAttestation.package.packageVersion -ne $packageVersion) {
        throw "Existing build attestation does not describe the retained immutable package and source commit: $attestationPath"
    }
}
else {
    $attestation | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $attestationPath -Encoding utf8NoBOM
}

Write-Host "Created $artifactPath"
Write-Host "SHA-256 $materializedHash"
Write-Host "Build attestation $attestationPath"
