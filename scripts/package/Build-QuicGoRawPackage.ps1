[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$SkipSmoke
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression.FileSystem

$Root = (Resolve-Path -LiteralPath $Root).Path
$OutputRoot = [System.IO.Path]::GetFullPath($OutputRoot)

$componentName = 'quic-go-raw'
$componentRoot = Join-Path $Root "implementations/$componentName"
$sourceRoot = Join-Path $componentRoot 'source'
$sourcePackageManifest = Join-Path $componentRoot 'protocol-lab-package.json'
$sourceInternalManifest = Join-Path $componentRoot 'protocol-lab.internal.json'
$sourceImplementationManifest = Join-Path $componentRoot 'implementations/quic-go-raw.yaml'
$sourceRunPs1 = Join-Path $componentRoot 'run.ps1'
$sourceRunSh = Join-Path $componentRoot 'run.sh'

foreach ($path in @(
    $sourcePackageManifest,
    $sourceInternalManifest,
    $sourceImplementationManifest,
    $sourceRunPs1,
    $sourceRunSh,
    (Join-Path $sourceRoot 'go.mod')
)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required quic-go raw package input was not found: $path"
    }
}

if (-not $SkipSmoke) {
    & go -C $sourceRoot test ./cmd/quic-go-raw
    if ($LASTEXITCODE -ne 0) {
        throw "go test failed with exit code $LASTEXITCODE."
    }
}

$packageManifest = Get-Content -LiteralPath $sourcePackageManifest -Raw | ConvertFrom-Json
$packageId = [string]$packageManifest.packageId
$packageVersion = [string]$packageManifest.packageVersion
$stagingRoot = Join-Path $OutputRoot $componentName
$packageRoot = Join-Path $stagingRoot 'package'
$packageBinLinux = Join-Path $packageRoot 'bin/linux-x64'
$packageBinWindows = Join-Path $packageRoot 'bin/windows-x64'
$packageImplementations = Join-Path $packageRoot 'implementations'
$artifactName = "$packageId.$packageVersion.plabpkg"
$artifactPath = Join-Path $OutputRoot $artifactName
$linuxBinaryPath = Join-Path $packageBinLinux 'quic-go-raw'
$windowsBinaryPath = Join-Path $packageBinWindows 'quic-go-raw.exe'

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $packageBinLinux, $packageBinWindows, $packageImplementations | Out-Null

function Invoke-GoBuild {
    param(
        [Parameter(Mandatory)][string]$GoOs,
        [Parameter(Mandatory)][string]$GoArch,
        [Parameter(Mandatory)][string]$OutputPath
    )

    $oldGoOs = [Environment]::GetEnvironmentVariable('GOOS', 'Process')
    $oldGoArch = [Environment]::GetEnvironmentVariable('GOARCH', 'Process')
    $oldCgo = [Environment]::GetEnvironmentVariable('CGO_ENABLED', 'Process')
    try {
        [Environment]::SetEnvironmentVariable('GOOS', $GoOs, 'Process')
        [Environment]::SetEnvironmentVariable('GOARCH', $GoArch, 'Process')
        [Environment]::SetEnvironmentVariable('CGO_ENABLED', '0', 'Process')
        & go -C $sourceRoot build -buildvcs=false -trimpath -ldflags '-s -w -X main.quicGoVersion=v0.60.0' -o $OutputPath ./cmd/quic-go-raw
        if ($LASTEXITCODE -ne 0) {
            throw "go build failed for $GoOs/$GoArch with exit code $LASTEXITCODE."
        }
    }
    finally {
        [Environment]::SetEnvironmentVariable('GOOS', $oldGoOs, 'Process')
        [Environment]::SetEnvironmentVariable('GOARCH', $oldGoArch, 'Process')
        [Environment]::SetEnvironmentVariable('CGO_ENABLED', $oldCgo, 'Process')
    }
}

Invoke-GoBuild -GoOs 'linux' -GoArch 'amd64' -OutputPath $linuxBinaryPath
Invoke-GoBuild -GoOs 'windows' -GoArch 'amd64' -OutputPath $windowsBinaryPath

Copy-Item -LiteralPath $sourcePackageManifest -Destination (Join-Path $packageRoot 'protocol-lab-package.json') -Force
Copy-Item -LiteralPath $sourceInternalManifest -Destination (Join-Path $packageRoot 'protocol-lab.internal.json') -Force
Copy-Item -LiteralPath $sourceImplementationManifest -Destination (Join-Path $packageImplementations 'quic-go-raw.yaml') -Force
Copy-Item -LiteralPath (Join-Path $componentRoot 'README.md') -Destination (Join-Path $packageRoot 'README.md') -Force
Copy-Item -LiteralPath $sourceRunPs1 -Destination (Join-Path $packageRoot 'run.ps1') -Force
Copy-Item -LiteralPath $sourceRunSh -Destination (Join-Path $packageRoot 'run.sh') -Force

$executionManifest = Get-Content -LiteralPath $sourceInternalManifest -Raw | ConvertFrom-Json
$executionManifest.environments = @(
    [ordered]@{
        os = 'linux'
        arch = 'x64'
        entrypoint = [ordered]@{
            kind = 'process'
            path = 'bin/linux-x64/quic-go-raw'
            arguments = @()
            workingDirectory = '.'
        }
    }
    [ordered]@{
        os = 'windows'
        arch = 'x64'
        entrypoint = [ordered]@{
            kind = 'process'
            path = 'bin/windows-x64/quic-go-raw.exe'
            arguments = @()
            workingDirectory = '.'
        }
    }
)
$executionManifest.commands.buildTemplate = 'pwsh ../../scripts/package/Build-QuicGoRawPackage.ps1'
$executionManifest.commands.serverTemplate = 'pwsh ./run.ps1'
$executionManifest.commands.planOnly = 'pwsh ./run.ps1 -PlanOnly'
$executionManifest | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8

& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath $packageRoot `
    -SourceComponentPath $componentRoot `
    -BuildConfiguration Release `
    -RuntimeIdentifier 'linux-x64+windows-x64' `
    -IncludeReadme `
    -PreparedPackageRoot

$archive = $null
try {
    $archive = [System.IO.Compression.ZipFile]::OpenRead($artifactPath)
    foreach ($entryName in @(
        'README.md',
        'bin/linux-x64/quic-go-raw',
        'bin/windows-x64/quic-go-raw.exe',
        'implementations/quic-go-raw.yaml',
        'protocol-lab-package.json',
        'protocol-lab.internal.json',
        'run.ps1',
        'run.sh'
    )) {
        if (-not $archive.GetEntry($entryName)) {
            throw "Package archive '$artifactPath' is missing expected entry '$entryName'."
        }
    }
}
finally {
    if ($null -ne $archive) {
        $archive.Dispose()
    }
}
