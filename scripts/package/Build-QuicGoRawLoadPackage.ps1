[CmdletBinding()]
param(
    [ValidateSet('win-x64', 'linux-x64')]
    [string]$RuntimeIdentifier = 'win-x64',

    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),

    [switch]$SkipSmoke
)

$ErrorActionPreference = 'Stop'

function ConvertFrom-RuntimeIdentifier {
    param([Parameter(Mandatory)][string]$RuntimeIdentifier)

    $parts = $RuntimeIdentifier.Split('-', 2, [System.StringSplitOptions]::RemoveEmptyEntries)
    if ($parts.Length -ne 2) {
        throw "Unsupported runtime identifier '$RuntimeIdentifier'."
    }

    $goOs = switch ($parts[0].ToLowerInvariant()) {
        'win' { 'windows' }
        'windows' { 'windows' }
        'linux' { 'linux' }
        default { throw "Unsupported runtime identifier OS '$($parts[0])'." }
    }

    $goArch = switch ($parts[1].ToLowerInvariant()) {
        'x64' { 'amd64' }
        default { throw "Unsupported runtime identifier architecture '$($parts[1])'." }
    }

    return [ordered]@{
        os = $goOs
        arch = $parts[1].ToLowerInvariant()
        goOs = $goOs
        goArch = $goArch
        exeSuffix = if ($goOs -eq 'windows') { '.exe' } else { '' }
    }
}

$componentName = 'quic-go-raw-load'
$componentRoot = Join-Path $Root "executors/$componentName"
$sourceRoot = Join-Path $componentRoot 'source'
$sourcePackageManifest = Join-Path $componentRoot 'protocol-lab-package.json'
$sourceInternalManifest = Join-Path $componentRoot 'protocol-lab.internal.json'
$sourceExecutorManifest = Join-Path $componentRoot 'test-executors/quic-go-raw-load.yaml'
$sourceToolchain = Join-Path $componentRoot 'toolchain.json'
$sourceExecutePs1 = Join-Path $componentRoot 'execute.ps1'
$sourceExecuteSh = Join-Path $componentRoot 'execute.sh'

foreach ($path in @($sourcePackageManifest, $sourceInternalManifest, $sourceExecutorManifest, $sourceToolchain, $sourceExecutePs1, $sourceExecuteSh, (Join-Path $sourceRoot 'go.mod'))) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required quic-go raw load package input was not found: $path"
    }
}

if (-not $SkipSmoke) {
    & go -C $sourceRoot test ./cmd/quic-go-raw-load
    if ($LASTEXITCODE -ne 0) {
        throw "go test failed with exit code $LASTEXITCODE."
    }
}

$rid = ConvertFrom-RuntimeIdentifier -RuntimeIdentifier $RuntimeIdentifier
$packageManifest = Get-Content -LiteralPath $sourcePackageManifest -Raw | ConvertFrom-Json
$packageId = [string]$packageManifest.packageId
$packageVersion = [string]$packageManifest.packageVersion
$stagingRoot = Join-Path $OutputRoot "$componentName/$RuntimeIdentifier"
$packageRoot = Join-Path $stagingRoot 'package'
$packageBin = Join-Path $packageRoot "bin/$RuntimeIdentifier"
$packageExecutors = Join-Path $packageRoot 'test-executors'
$artifactName = "$packageId.$packageVersion.$RuntimeIdentifier.plabpkg"
$artifactPath = Join-Path $OutputRoot $artifactName
$binaryName = "quic-go-raw-load$($rid.exeSuffix)"
$binaryPath = Join-Path $packageBin $binaryName

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $packageBin, $packageExecutors | Out-Null

$oldGoOs = [Environment]::GetEnvironmentVariable('GOOS', 'Process')
$oldGoArch = [Environment]::GetEnvironmentVariable('GOARCH', 'Process')
try {
    [Environment]::SetEnvironmentVariable('GOOS', $rid.goOs, 'Process')
    [Environment]::SetEnvironmentVariable('GOARCH', $rid.goArch, 'Process')
    & go -C $sourceRoot build -trimpath -o $binaryPath ./cmd/quic-go-raw-load
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE."
    }
}
finally {
    [Environment]::SetEnvironmentVariable('GOOS', $oldGoOs, 'Process')
    [Environment]::SetEnvironmentVariable('GOARCH', $oldGoArch, 'Process')
}

Copy-Item -LiteralPath $sourcePackageManifest -Destination (Join-Path $packageRoot 'protocol-lab-package.json') -Force
Copy-Item -LiteralPath $sourceExecutorManifest -Destination (Join-Path $packageExecutors 'quic-go-raw-load.yaml') -Force
Copy-Item -LiteralPath $sourceToolchain -Destination (Join-Path $packageRoot 'toolchain.json') -Force
Copy-Item -LiteralPath $sourceExecutePs1 -Destination (Join-Path $packageRoot 'execute.ps1') -Force
Copy-Item -LiteralPath $sourceExecuteSh -Destination (Join-Path $packageRoot 'execute.sh') -Force

$executionManifest = Get-Content -LiteralPath $sourceInternalManifest -Raw | ConvertFrom-Json
$executionManifest.environments = @(
    [ordered]@{
        os = $rid.os
        arch = $rid.arch
        entrypoint = [ordered]@{
            kind = 'process'
            path = "bin/$RuntimeIdentifier/$binaryName"
            arguments = @()
            workingDirectory = '.'
        }
    }
)
$executionManifest.dependencies.requiresPwsh = $false
$executionManifest.dependencies.requiresBash = $false
$executionManifest.dependencies.requiresGo = $false
$executionManifest.dependencies.requiredCapabilities = @()
$executionManifest | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8

Remove-Item -LiteralPath $artifactPath -Force -ErrorAction SilentlyContinue
Compress-Archive -Path (Join-Path $packageRoot '*') -DestinationPath $artifactPath -Force

Write-Host "Created $artifactPath"
