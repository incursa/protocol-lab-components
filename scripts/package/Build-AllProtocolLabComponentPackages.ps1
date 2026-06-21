[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),

    [switch]$Clean
)

$ErrorActionPreference = 'Stop'

Add-Type -AssemblyName System.IO.Compression.FileSystem

function Get-RelativePath {
    param(
        [Parameter(Mandatory)][string]$BasePath,
        [Parameter(Mandatory)][string]$Path
    )

    return [System.IO.Path]::GetRelativePath($BasePath, $Path).Replace('\', '/')
}

function Assert-PathIsUnderRoot {
    param(
        [Parameter(Mandatory)][string]$CandidatePath,
        [Parameter(Mandatory)][string]$ExpectedRoot
    )

    $resolvedRoot = [System.IO.Path]::GetFullPath($ExpectedRoot)
    $resolvedCandidate = [System.IO.Path]::GetFullPath($CandidatePath)
    if (-not $resolvedCandidate.StartsWith($resolvedRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to clean output path outside repository root: $resolvedCandidate"
    }
}

function Read-ZipJsonEntry {
    param(
        [Parameter(Mandatory)][System.IO.Compression.ZipArchive]$Archive,
        [Parameter(Mandatory)][string]$EntryName
    )

    $entry = $Archive.Entries | Where-Object { $_.FullName -eq $EntryName } | Select-Object -First 1
    if ($null -eq $entry) {
        return $null
    }

    $stream = $entry.Open()
    try {
        $reader = [System.IO.StreamReader]::new($stream)
        try {
            return ($reader.ReadToEnd() | ConvertFrom-Json)
        }
        finally {
            $reader.Dispose()
        }
    }
    finally {
        $stream.Dispose()
    }
}

function ConvertTo-StringArray {
    param([AllowNull()]$Value)

    if ($null -eq $Value) {
        return @()
    }

    return @($Value | ForEach-Object { [string]$_ })
}

function Get-ProvidedIds {
    param([Parameter(Mandatory)]$Manifest)

    $ids = [System.Collections.Generic.List[string]]::new()
    foreach ($provided in @($Manifest.providedImplementations)) {
        if ($provided.implementationId) {
            [void]$ids.Add([string]$provided.implementationId)
        }
    }

    foreach ($provided in @($Manifest.providedTestExecutors)) {
        if ($provided.testExecutorId) {
            [void]$ids.Add([string]$provided.testExecutorId)
        }
    }

    foreach ($provided in @($Manifest.providedScenarios)) {
        if ($provided.scenarioId) {
            [void]$ids.Add([string]$provided.scenarioId)
        }
    }

    foreach ($provided in @($Manifest.providedSuites)) {
        if ($provided.suiteId) {
            [void]$ids.Add([string]$provided.suiteId)
        }
    }

    return @($ids)
}

function Get-PackageArtifactInspection {
    param(
        [Parameter(Mandatory)][System.IO.FileInfo]$Artifact,
        [Parameter(Mandatory)][string]$Root,
        [Parameter(Mandatory)][string]$OutputRoot,
        [Parameter(Mandatory)]$Build
    )

    $hash = Get-FileHash -LiteralPath $Artifact.FullName -Algorithm SHA256
    $zip = [System.IO.Compression.ZipFile]::OpenRead($Artifact.FullName)
    try {
        $publicManifestEntry = $zip.Entries | Where-Object { $_.FullName -eq 'protocol-lab-package.json' } | Select-Object -First 1
        $internalManifestEntry = $zip.Entries | Where-Object { $_.FullName -eq 'protocol-lab.internal.json' } | Select-Object -First 1

        if ($null -eq $publicManifestEntry) {
            throw "$($Artifact.Name): missing root protocol-lab-package.json."
        }

        if ($null -eq $internalManifestEntry) {
            throw "$($Artifact.Name): missing root protocol-lab.internal.json."
        }

        $publicManifest = Read-ZipJsonEntry -Archive $zip -EntryName 'protocol-lab-package.json'
        $internalManifest = Read-ZipJsonEntry -Archive $zip -EntryName 'protocol-lab.internal.json'

        if ($publicManifest.schemaVersion -ne 'protocol-lab-package-v2') {
            throw "$($Artifact.Name): protocol-lab-package.json has schemaVersion '$($publicManifest.schemaVersion)'."
        }

        if ($internalManifest.schemaVersion -ne 'protocol-lab-internal-execution-v1') {
            throw "$($Artifact.Name): protocol-lab.internal.json has schemaVersion '$($internalManifest.schemaVersion)'."
        }

        $runtimeIdentifiers = @(
            $internalManifest.environments | ForEach-Object {
                if ($_.os -and $_.arch) {
                    "$($_.os)-$($_.arch)"
                }
            }
        )

        return [ordered]@{
            packageId = [string]$publicManifest.packageId
            packageVersion = [string]$publicManifest.packageVersion
            kind = [string]$publicManifest.kind
            displayName = [string]$publicManifest.displayName
            artifactName = $Artifact.Name
            artifactPath = Get-RelativePath -BasePath $Root -Path $Artifact.FullName
            artifactRootPath = Get-RelativePath -BasePath $Root -Path $OutputRoot
            sizeBytes = $Artifact.Length
            sha256 = $hash.Hash.ToLowerInvariant()
            runtimeIdentifiers = ConvertTo-StringArray -Value $runtimeIdentifiers
            entryManifests = ConvertTo-StringArray -Value $publicManifest.entryManifests
            providedComponentIds = Get-ProvidedIds -Manifest $publicManifest
            builderScript = $Build.script
            buildArguments = ConvertTo-StringArray -Value $Build.arguments
            sourceComponentPath = $Build.componentPath
            archiveInspection = [ordered]@{
                hasPublicManifest = $true
                hasInternalManifest = $true
                entryCount = $zip.Entries.Count
            }
        }
    }
    finally {
        $zip.Dispose()
    }
}

function Write-PackageIndexMarkdown {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)]$Index
    )

    $lines = [System.Collections.Generic.List[string]]::new()
    [void]$lines.Add('# ProtocolLab Component Package Index')
    [void]$lines.Add('')
    [void]$lines.Add(('Generated at: `{0}`' -f $Index.generatedAtUtc))
    [void]$lines.Add('')
    [void]$lines.Add('| Package | Version | Kind | Artifact | SHA-256 |')
    [void]$lines.Add('| --- | --- | --- | --- | --- |')

    foreach ($package in $Index.packages) {
        [void]$lines.Add(('| `{0}` | `{1}` | `{2}` | `{3}` | `{4}` |' -f $package.packageId, $package.packageVersion, $package.kind, $package.artifactName, $package.sha256))
    }

    $lines | Set-Content -LiteralPath $Path -Encoding utf8
}

function Write-ValidationSummaryMarkdown {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)]$Summary
    )

    $lines = [System.Collections.Generic.List[string]]::new()
    [void]$lines.Add('# ProtocolLab Component Package Validation Summary')
    [void]$lines.Add('')
    [void]$lines.Add(('Generated at: `{0}`' -f $Summary.generatedAtUtc))
    [void]$lines.Add('')
    [void]$lines.Add("| Check | Status |")
    [void]$lines.Add("| --- | --- |")
    [void]$lines.Add(('| Manifest validation | `{0}` |' -f $Summary.manifestValidation.status))
    [void]$lines.Add(('| Package archive inspection | `{0}` |' -f $Summary.archiveInspection.status))
    [void]$lines.Add(('| Built artifact count | `{0}` |' -f $Summary.archiveInspection.artifactCount))
    [void]$lines.Add('')
    [void]$lines.Add('## Builders')
    [void]$lines.Add('')
    [void]$lines.Add('| Component | Script | Arguments | Artifacts | Status |')
    [void]$lines.Add('| --- | --- | --- | --- | --- |')

    foreach ($builder in $Summary.builders) {
        $arguments = if ($builder.arguments.Count -gt 0) { $builder.arguments -join ' ' } else { '' }
        $artifacts = if ($builder.artifacts.Count -gt 0) { $builder.artifacts -join '<br>' } else { '' }
        [void]$lines.Add(('| `{0}` | `{1}` | `{2}` | {3} | `{4}` |' -f $builder.componentPath, $builder.script, $arguments, $artifacts, $builder.status))
    }

    $lines | Set-Content -LiteralPath $Path -Encoding utf8
}

$Root = (Resolve-Path $Root).Path
$OutputRoot = [System.IO.Path]::GetFullPath($OutputRoot)
Assert-PathIsUnderRoot -CandidatePath $OutputRoot -ExpectedRoot $Root

New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null

if ($Clean) {
    Get-ChildItem -LiteralPath $OutputRoot -File -Filter '*.plabpkg' -ErrorAction SilentlyContinue | Remove-Item -Force
    foreach ($generatedFile in @(
        'package-index.json',
        'package-index.md',
        'SHA256SUMS.txt',
        'package-validation-summary.json',
        'package-validation-summary.md'
    )) {
        Remove-Item -LiteralPath (Join-Path $OutputRoot $generatedFile) -Force -ErrorAction SilentlyContinue
    }

    foreach ($generatedDirectory in @('stage', 'tmp')) {
        Remove-Item -LiteralPath (Join-Path $OutputRoot $generatedDirectory) -Recurse -Force -ErrorAction SilentlyContinue
    }
}

$manifestValidationCommand = './scripts/package/Validate-ProtocolLabComponentManifests.ps1'
$manifestValidationOutput = & (Join-Path $PSScriptRoot 'Validate-ProtocolLabComponentManifests.ps1') -Root $Root 2>&1

$packageBuilds = @(
    [pscustomobject]@{ componentPath = 'implementations/kestrel-http1'; script = 'Build-KestrelHttp1Package.ps1'; arguments = @('win-x64') },
    [pscustomobject]@{ componentPath = 'implementations/kestrel-http1'; script = 'Build-KestrelHttp1Package.ps1'; arguments = @('linux-x64') },
    [pscustomobject]@{ componentPath = 'implementations/kestrel-http2'; script = 'Build-KestrelHttp2Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/kestrel-http3'; script = 'Build-KestrelHttp3Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/caddy-http1'; script = 'Build-CaddyHttp1Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/caddy-http3'; script = 'Build-CaddyHttp3Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/nginx-http1'; script = 'Build-NginxHttp1Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/nginx-http3'; script = 'Build-NginxHttp3Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/quic-go-http3'; script = 'Build-QuicGoHttp3Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/quic-go-raw'; script = 'Build-QuicGoRawPackage.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/aioquic-http3'; script = 'Build-AioquicHttp3Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/quiche-http3'; script = 'Build-QuicheHttp3Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'implementations/ngtcp2-http3'; script = 'Build-Ngtcp2Http3Package.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'executors/curl-http3-client'; script = 'Build-CurlHttp3ClientPackage.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'executors/go-http1-executor'; script = 'Build-GoHttp1ExecutorPackage.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'executors/h3spec-http3-qpack'; script = 'Build-H3SpecHttp3QpackPackage.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'scenarios/h3spec-http3-qpack'; script = 'Build-H3SpecHttp3QpackScenarioPackage.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'executors/aioquic-rfc9220-websocket'; script = 'Build-AioquicRfc9220WebSocketPackage.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'scenarios/aioquic-rfc9220-websocket'; script = 'Build-AioquicRfc9220WebSocketScenarioPackage.ps1'; arguments = @() },
    [pscustomobject]@{ componentPath = 'executors/quic-go-raw-load'; script = 'Build-QuicGoRawLoadPackage.ps1'; arguments = @('win-x64') },
    [pscustomobject]@{ componentPath = 'executors/quic-go-raw-load'; script = 'Build-QuicGoRawLoadPackage.ps1'; arguments = @('linux-x64') },
    [pscustomobject]@{ componentPath = 'scenarios/raw-quic-transport'; script = 'Build-RawQuicScenarioPackage.ps1'; arguments = @() }
)

$builderResults = [System.Collections.Generic.List[object]]::new()
$builtArtifacts = [System.Collections.Generic.List[System.IO.FileInfo]]::new()

foreach ($build in $packageBuilds) {
    $scriptPath = Join-Path $PSScriptRoot $build.script
    if (-not (Test-Path -LiteralPath $scriptPath -PathType Leaf)) {
        throw "Package build script not found: $scriptPath"
    }

    $startTime = (Get-Date).ToUniversalTime().AddSeconds(-2)
    $argumentList = @($build.arguments)
    $namedArguments = @{
        Root = $Root
        OutputRoot = $OutputRoot
    }

    & $scriptPath @argumentList @namedArguments
    if ($LASTEXITCODE -is [int] -and $LASTEXITCODE -ne 0) {
        throw "$($build.script) failed with exit code $LASTEXITCODE."
    }

    $artifacts = @(
        Get-ChildItem -LiteralPath $OutputRoot -File -Filter '*.plabpkg' |
            Where-Object { $_.LastWriteTimeUtc -ge $startTime } |
            Sort-Object FullName
    )

    if ($artifacts.Count -eq 0) {
        throw "$($build.script) did not produce or update a .plabpkg artifact."
    }

    foreach ($artifact in $artifacts) {
        [void]$builtArtifacts.Add($artifact)
    }

    [void]$builderResults.Add([pscustomobject]@{
        componentPath = $build.componentPath
        script = $build.script
        arguments = ConvertTo-StringArray -Value $build.arguments
        artifacts = @($artifacts | ForEach-Object { $_.Name })
        status = 'passed'
    })
}

$uniqueBuiltArtifacts = @(
    $builtArtifacts |
        Sort-Object FullName -Unique |
        Sort-Object Name
)

$packageInspections = [System.Collections.Generic.List[object]]::new()
foreach ($artifact in $uniqueBuiltArtifacts) {
    $matchingBuild = $builderResults |
        Where-Object { $_.artifacts -contains $artifact.Name } |
        Select-Object -First 1
    [void]$packageInspections.Add((Get-PackageArtifactInspection -Artifact $artifact -Root $Root -OutputRoot $OutputRoot -Build $matchingBuild))
}

$commit = $null
try {
    $commit = (& git -C $Root rev-parse HEAD).Trim()
}
catch {
    $commit = $null
}

$generatedAtUtc = (Get-Date).ToUniversalTime().ToString('o')
$index = [ordered]@{
    schemaVersion = 'protocol-lab-package-index-v1'
    generatedAtUtc = $generatedAtUtc
    repository = [ordered]@{
        root = $Root
        commit = $commit
    }
    artifactRootPath = Get-RelativePath -BasePath $Root -Path $OutputRoot
    packageCount = $packageInspections.Count
    packages = @($packageInspections | Sort-Object packageId, packageVersion, artifactName)
}

$summary = [ordered]@{
    schemaVersion = 'protocol-lab-package-validation-summary-v1'
    generatedAtUtc = $generatedAtUtc
    manifestValidation = [ordered]@{
        command = $manifestValidationCommand
        status = 'passed'
        output = @($manifestValidationOutput | ForEach-Object { [string]$_ })
    }
    archiveInspection = [ordered]@{
        status = 'passed'
        artifactCount = $packageInspections.Count
        requiredRootEntries = @('protocol-lab-package.json', 'protocol-lab.internal.json')
        artifacts = @(
            $packageInspections | ForEach-Object {
                [ordered]@{
                    artifactName = $_.artifactName
                    packageId = $_.packageId
                    packageVersion = $_.packageVersion
                    hasPublicManifest = $_.archiveInspection.hasPublicManifest
                    hasInternalManifest = $_.archiveInspection.hasInternalManifest
                    entryCount = $_.archiveInspection.entryCount
                    status = 'passed'
                }
            }
        )
    }
    builders = @($builderResults)
}

$indexJsonPath = Join-Path $OutputRoot 'package-index.json'
$indexMarkdownPath = Join-Path $OutputRoot 'package-index.md'
$sha256ManifestPath = Join-Path $OutputRoot 'SHA256SUMS.txt'
$summaryJsonPath = Join-Path $OutputRoot 'package-validation-summary.json'
$summaryMarkdownPath = Join-Path $OutputRoot 'package-validation-summary.md'

$index | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath $indexJsonPath -Encoding utf8
Write-PackageIndexMarkdown -Path $indexMarkdownPath -Index $index
@($index.packages | ForEach-Object { "$($_.sha256)  $($_.artifactName)" }) |
    Set-Content -LiteralPath $sha256ManifestPath -Encoding utf8
$summary | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath $summaryJsonPath -Encoding utf8
Write-ValidationSummaryMarkdown -Path $summaryMarkdownPath -Summary $summary

Write-Host "Built $($index.packageCount) ProtocolLab component package artifact(s)."
Write-Host "Package index: $indexJsonPath"
Write-Host "Package index markdown: $indexMarkdownPath"
Write-Host "SHA-256 manifest: $sha256ManifestPath"
Write-Host "Validation summary: $summaryJsonPath"
