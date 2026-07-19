[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$ComponentPath,

    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),

    [string]$BuildConfiguration = 'Release',

    [string]$RuntimeIdentifier = 'portable',

    [string]$SourceComponentPath,

    [string]$ArtifactSuffix = '',

    [string]$ComponentGraphPath = (Join-Path $Root 'release/component-graph.v1.json'),

    [switch]$IncludeReadme,

    [switch]$PreparedPackageRoot,

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

function ConvertTo-DeterministicJsonValue {
    param($Value)

    if ($null -eq $Value) {
        return $null
    }

    if ($Value -is [System.Collections.IDictionary]) {
        $ordered = [ordered]@{}
        foreach ($key in @($Value.Keys | Sort-Object)) {
            $ordered[[string]$key] = ConvertTo-DeterministicJsonValue -Value $Value[$key]
        }

        return $ordered
    }

    if ($Value -is [pscustomobject]) {
        $ordered = [ordered]@{}
        foreach ($property in @($Value.PSObject.Properties | Sort-Object Name)) {
            $ordered[[string]$property.Name] = ConvertTo-DeterministicJsonValue -Value $property.Value
        }

        return $ordered
    }

    if ($Value -is [System.Collections.IEnumerable] -and -not ($Value -is [string])) {
        return @($Value | ForEach-Object { ConvertTo-DeterministicJsonValue -Value $_ })
    }

    return $Value
}

function ConvertTo-DeterministicJson {
    param([Parameter(Mandatory)]$Value)

    return ((ConvertTo-DeterministicJsonValue -Value $Value) | ConvertTo-Json -Depth 32)
}

function New-DeterministicZipArchive {
    param(
        [Parameter(Mandatory)][string]$SourceRoot,
        [Parameter(Mandatory)][string]$DestinationPath
    )

    $fixedTimestamp = [DateTimeOffset]::new(1980, 1, 1, 0, 0, 0, [TimeSpan]::Zero)
    $archiveStream = [System.IO.File]::Open($DestinationPath, [System.IO.FileMode]::CreateNew, [System.IO.FileAccess]::ReadWrite, [System.IO.FileShare]::None)
    try {
        $archive = [System.IO.Compression.ZipArchive]::new($archiveStream, [System.IO.Compression.ZipArchiveMode]::Create, $true, [System.Text.UTF8Encoding]::new($false))
        try {
            $files = @(Get-ChildItem -LiteralPath $SourceRoot -Recurse -File -Force | Sort-Object { [System.IO.Path]::GetRelativePath($SourceRoot, $_.FullName).Replace('\', '/') })
            foreach ($file in $files) {
                $entryName = [System.IO.Path]::GetRelativePath($SourceRoot, $file.FullName).Replace('\', '/')
                $entry = $archive.CreateEntry($entryName, [System.IO.Compression.CompressionLevel]::Optimal)
                $entry.LastWriteTime = $fixedTimestamp
                $entryStream = $entry.Open()
                $sourceStream = [System.IO.File]::OpenRead($file.FullName)
                try {
                    $sourceStream.CopyTo($entryStream)
                }
                finally {
                    $sourceStream.Dispose()
                    $entryStream.Dispose()
                }
            }
        }
        finally {
            $archive.Dispose()
        }
    }
    finally {
        $archiveStream.Dispose()
    }
}

$componentRoot = if ([System.IO.Path]::IsPathRooted($ComponentPath)) {
    $ComponentPath
}
else {
    Join-Path $Root $ComponentPath
}

$componentRoot = (Resolve-Path $componentRoot).Path
$repositoryRoot = (Resolve-Path $Root).Path
$sourceComponentRoot = if ([string]::IsNullOrWhiteSpace($SourceComponentPath)) {
    $componentRoot
}
elseif ([System.IO.Path]::IsPathRooted($SourceComponentPath)) {
    (Resolve-Path $SourceComponentPath).Path
}
else {
    (Resolve-Path (Join-Path $Root $SourceComponentPath)).Path
}
$sourceRepository = Invoke-GitText -RepositoryRoot $repositoryRoot -Arguments @('config', '--get', 'remote.origin.url')
$sourceCommit = Invoke-GitText -RepositoryRoot $repositoryRoot -Arguments @('rev-parse', 'HEAD')
$sourceCommitTimestamp = Invoke-GitText -RepositoryRoot $repositoryRoot -Arguments @('show', '-s', '--format=%cI', 'HEAD')
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

$componentRelativePath = [System.IO.Path]::GetRelativePath($repositoryRoot, $sourceComponentRoot).Replace('\', '/')
$componentClosure = $null
if (Test-Path -LiteralPath $ComponentGraphPath -PathType Leaf) {
    . (Join-Path $PSScriptRoot 'ProtocolLabComponentRelease.Common.ps1')
    $componentGraph = Get-ProtocolLabComponentReleaseGraph -GraphPath $ComponentGraphPath
    $modeledComponent = @($componentGraph.components | Where-Object {
        $_.migrationState -eq 'modeled' -and $_.packageRoot -eq $componentRelativePath
    })
    if ($modeledComponent.Count -gt 1) {
        throw "Component graph has multiple modeled entries for '$componentRelativePath'."
    }
    if ($modeledComponent.Count -eq 1) {
        $componentClosure = Get-ProtocolLabComponentClosure -Graph $componentGraph -ComponentId ([string]$modeledComponent[0].id) -Root $repositoryRoot
        if ([string]$componentClosure.packageId -ne $packageId) {
            throw "Component graph package identity for '$componentRelativePath' does not match its manifest."
        }
    }
}

New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null
$normalizedArtifactSuffix = if ([string]::IsNullOrWhiteSpace($ArtifactSuffix)) { '' } elseif ($ArtifactSuffix.StartsWith('.')) { $ArtifactSuffix } else { ".$ArtifactSuffix" }
$stagingRoot = Join-Path $OutputRoot ("stage/" + (($packageId + $normalizedArtifactSuffix) -replace '[^A-Za-z0-9_.-]', '_'))
$artifactPath = Join-Path $OutputRoot "$packageId.$packageVersion$normalizedArtifactSuffix.plabpkg"
$candidateArtifactPath = "$artifactPath.building"
$attestationPath = "$artifactPath.build-attestation.json"

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $stagingRoot | Out-Null

$excludedNames = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
@(
    'package.protocol-lab.json',
    'artifacts',
    'packages'
) | ForEach-Object { [void]$excludedNames.Add($_) }
if (-not $PreparedPackageRoot) {
    [void]$excludedNames.Add('bin')
    [void]$excludedNames.Add('obj')
}
if (-not $IncludeReadme) {
    [void]$excludedNames.Add('README.md')
}

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

$stagedInternalManifestPath = Join-Path $stagingRoot 'protocol-lab.internal.json'
if (Test-Path -LiteralPath $stagedInternalManifestPath -PathType Leaf) {
    $stagedInternalManifest = Get-Content -LiteralPath $stagedInternalManifestPath -Raw | ConvertFrom-Json
    ConvertTo-DeterministicJson -Value $stagedInternalManifest | Set-Content -LiteralPath $stagedInternalManifestPath -Encoding utf8NoBOM
}

$embeddedProvenance = if ($null -ne $componentClosure) {
    # The component closure is the package-byte identity. Checkout provenance,
    # dirty state, host OS, and observation time remain attestation-only so an
    # unrelated monorepo commit cannot change this archive.
    [ordered]@{
        schemaVersion = 'protocol-lab.package-build-provenance.v2'
        component = [ordered]@{
            id = $componentClosure.componentId
            componentTreeDigest = $componentClosure.componentTreeDigest
            buildRecipeDigest = $componentClosure.buildRecipeDigest
            componentClosureDigest = $componentClosure.componentClosureDigest
        }
        build = [ordered]@{
            configuration = $BuildConfiguration
            runtimeIdentifier = $RuntimeIdentifier
        }
        package = [ordered]@{
            packageId = $packageId
            packageVersion = $packageVersion
        }
    }
}
else {
    [ordered]@{
        schemaVersion = 'protocol-lab.package-build-provenance.v1'
        generatedAtUtc = ([DateTimeOffset]::Parse($sourceCommitTimestamp)).ToUniversalTime().ToString('O')
        timestampBasis = 'source-commit'
        parityEligible = $workingTreeClean
        source = [ordered]@{
            repository = $sourceRepository
            commitSha = $sourceCommit
            workingTreeClean = $workingTreeClean
            dirtyState = if ($workingTreeClean) { 'clean' } else { 'dirty' }
            dirtyEntries = if ($workingTreeClean) { @() } else { @($sourceStatus -split "`n" | Where-Object { $_ }) }
            componentPath = $componentRelativePath
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
        }
    }
}
$embeddedProvenance | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $stagingRoot 'package-build-provenance.json') -Encoding utf8NoBOM

Remove-Item -LiteralPath $candidateArtifactPath -Force -ErrorAction SilentlyContinue
New-DeterministicZipArchive -SourceRoot $stagingRoot -DestinationPath $candidateArtifactPath

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
        componentPath = $componentRelativePath
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
    component = if ($null -eq $componentClosure) { $null } else {
        [ordered]@{
            id = $componentClosure.componentId
            componentTreeDigest = $componentClosure.componentTreeDigest
            buildRecipeDigest = $componentClosure.buildRecipeDigest
            componentClosureDigest = $componentClosure.componentClosureDigest
            contracts = @($componentClosure.closure.contracts)
            toolchains = @($componentClosure.closure.toolchains)
        }
    }
    claimBoundary = if ($workingTreeClean) {
        if ($null -ne $componentClosure) {
            'This attestation records checkout provenance for a deterministic immutable package whose archive identity is the declared component dependency closure.'
        }
        else {
            'This attestation identifies a deterministic immutable package archive for the recorded clean commit, configuration, runtime, and toolchain.'
        }
    }
    else {
        'Diagnostic-only dirty-source build. This artifact is not eligible for source/package parity or publication.'
    }
}

if (Test-Path -LiteralPath $attestationPath -PathType Leaf) {
    $existingAttestation = Get-Content -LiteralPath $attestationPath -Raw | ConvertFrom-Json
    if ([string]$existingAttestation.package.sha256 -ne $materializedHash -or
        [string]$existingAttestation.package.packageId -ne $packageId -or
        [string]$existingAttestation.package.packageVersion -ne $packageVersion) {
        throw "Existing build attestation does not describe the retained immutable package: $attestationPath"
    }
}
else {
    $attestation | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $attestationPath -Encoding utf8NoBOM
}

Write-Host "Created $artifactPath"
Write-Host "SHA-256 $materializedHash"
Write-Host "Build attestation $attestationPath"
