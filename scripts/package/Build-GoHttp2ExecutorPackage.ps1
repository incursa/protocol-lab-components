[CmdletBinding()]
param(
    [ValidateSet('win-x64', 'linux-x64')]
    [string]$RuntimeIdentifier = 'win-x64',

    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),

    [switch]$SkipSmoke,

    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
$Root = [System.IO.Path]::GetFullPath($Root)
$OutputRoot = [System.IO.Path]::GetFullPath($OutputRoot)

$componentName = 'go-http2-executor'
$componentRoot = Join-Path $Root "executors/$componentName"
$sourceRoot = Join-Path $componentRoot 'source'
$sourcePackageManifest = Join-Path $componentRoot 'protocol-lab-package.json'
$sourceInternalManifest = Join-Path $componentRoot 'protocol-lab.internal.json'
$sourceExecutorManifest = Join-Path $componentRoot 'test-executors/go-http2-executor.yaml'
$sourceToolchain = Join-Path $componentRoot 'toolchain.json'
$sourceLicense = Join-Path $componentRoot 'third-party/golang-x-net-LICENSE.txt'

foreach ($path in @($sourcePackageManifest, $sourceInternalManifest, $sourceExecutorManifest, $sourceToolchain, $sourceLicense, (Join-Path $sourceRoot 'go.mod'), (Join-Path $sourceRoot 'go.sum'))) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required HTTP/2 executor package input was not found: $path"
    }
}

if (-not $SkipSmoke) {
    & go -C $sourceRoot test -count=1 .
    if ($LASTEXITCODE -ne 0) {
        throw "go test failed with exit code $LASTEXITCODE."
    }
}

$rid = switch ($RuntimeIdentifier) {
    'win-x64' { [ordered]@{ os = 'windows'; arch = 'x64'; goOs = 'windows'; goArch = 'amd64'; executorName = 'go-http2-executor.exe' } }
    'linux-x64' { [ordered]@{ os = 'linux'; arch = 'x64'; goOs = 'linux'; goArch = 'amd64'; executorName = 'go-http2-executor' } }
}

$packageManifest = Get-Content -LiteralPath $sourcePackageManifest -Raw | ConvertFrom-Json
$stagingRoot = Join-Path $OutputRoot "$componentName/$RuntimeIdentifier"
$packageRoot = Join-Path $stagingRoot 'package'
$packageBin = Join-Path $packageRoot "bin/$RuntimeIdentifier"
$packageExecutors = Join-Path $packageRoot 'test-executors'
$packageLicenses = Join-Path $packageRoot 'third-party'
$executorPath = Join-Path $packageBin $rid.executorName

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $packageBin, $packageExecutors, $packageLicenses | Out-Null

$oldGoOs = [Environment]::GetEnvironmentVariable('GOOS', 'Process')
$oldGoArch = [Environment]::GetEnvironmentVariable('GOARCH', 'Process')
try {
    [Environment]::SetEnvironmentVariable('GOOS', $rid.goOs, 'Process')
    [Environment]::SetEnvironmentVariable('GOARCH', $rid.goArch, 'Process')
    & go -C $sourceRoot build -buildvcs=false -trimpath -o $executorPath .
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE."
    }
}
finally {
    [Environment]::SetEnvironmentVariable('GOOS', $oldGoOs, 'Process')
    [Environment]::SetEnvironmentVariable('GOARCH', $oldGoArch, 'Process')
}

if (-not (Test-Path -LiteralPath $executorPath -PathType Leaf)) {
    throw "Compiled HTTP/2 executor was not produced at $executorPath."
}

Copy-Item -LiteralPath $sourcePackageManifest -Destination (Join-Path $packageRoot 'protocol-lab-package.json') -Force
Copy-Item -LiteralPath $sourceExecutorManifest -Destination (Join-Path $packageExecutors 'go-http2-executor.yaml') -Force
Copy-Item -LiteralPath $sourceToolchain -Destination (Join-Path $packageRoot 'toolchain.json') -Force
Copy-Item -LiteralPath $sourceLicense -Destination (Join-Path $packageLicenses 'golang-x-net-LICENSE.txt') -Force

$executionManifest = Get-Content -LiteralPath $sourceInternalManifest -Raw | ConvertFrom-Json
$executionManifest.environments = @(
    [ordered]@{
        os = $rid.os
        arch = $rid.arch
        entrypoint = [ordered]@{
            kind = 'process'
            path = "bin/$RuntimeIdentifier/$($rid.executorName)"
            arguments = @()
            workingDirectory = '.'
        }
    }
)
$executionManifest.dependencies.requiresPwsh = $false
$executionManifest.dependencies.requiresBash = $false
$executionManifest.dependencies.requiresGo = $false
$executionManifest.dependencies.requiredCapabilities = @()
$executionManifest.dependencies | Add-Member -NotePropertyName tools -NotePropertyValue @(
    [ordered]@{
        name = 'go-x-net-http2-h2c-load'
        version = '0.3.0'
        engineModule = 'golang.org/x/net/http2'
        engineModuleVersion = 'v0.57.0'
        license = 'BSD-3-Clause'
        embedded = $true
    }
) -Force
$executionManifest | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8NoBOM

$packageArguments = @{
    Root = $Root
    OutputRoot = $OutputRoot
    ComponentPath = $packageRoot
    SourceComponentPath = $componentRoot
    ArtifactSuffix = $RuntimeIdentifier
    BuildConfiguration = 'Release'
    RuntimeIdentifier = $RuntimeIdentifier
    PreparedPackageRoot = $true
    AllowDirtySource = $AllowDirtySource
}
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') @packageArguments
