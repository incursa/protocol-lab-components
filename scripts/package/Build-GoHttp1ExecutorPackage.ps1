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

$componentName = 'go-http1-executor'
$componentRoot = Join-Path $Root "executors/$componentName"
$sourceRoot = Join-Path $componentRoot 'source'
$sourcePackageManifest = Join-Path $componentRoot 'protocol-lab-package.json'
$sourceInternalManifest = Join-Path $componentRoot 'protocol-lab.internal.json'
$sourceExecutorManifest = Join-Path $componentRoot 'test-executors/go-http1-executor.yaml'
$sourceToolchain = Join-Path $componentRoot 'toolchain.json'
$sourceLicense = Join-Path $componentRoot 'third-party/oha-LICENSE.txt'

foreach ($path in @($sourcePackageManifest, $sourceInternalManifest, $sourceExecutorManifest, $sourceToolchain, $sourceLicense, (Join-Path $sourceRoot 'go.mod'))) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required HTTP/1 executor package input was not found: $path"
    }
}

if (-not $SkipSmoke) {
    & go -C $sourceRoot test -count=1 .
    if ($LASTEXITCODE -ne 0) {
        throw "go test failed with exit code $LASTEXITCODE."
    }
}

$rid = switch ($RuntimeIdentifier) {
    'win-x64' {
        [ordered]@{
            os = 'windows'; arch = 'x64'; goOs = 'windows'; goArch = 'amd64'
            executorName = 'go-http1-executor.exe'; ohaName = 'oha.exe'
            assetName = 'oha-windows-amd64-pgo.exe'
            url = 'https://github.com/hatoo/oha/releases/download/v1.15.0/oha-windows-amd64-pgo.exe'
            sha256 = 'b1dac6c1272abbb4b2c52723d869f0bfdd7807e742955e46e5cb515a15114f6f'
        }
    }
    'linux-x64' {
        [ordered]@{
            os = 'linux'; arch = 'x64'; goOs = 'linux'; goArch = 'amd64'
            executorName = 'go-http1-executor'; ohaName = 'oha'
            assetName = 'oha-linux-amd64-pgo'
            url = 'https://github.com/hatoo/oha/releases/download/v1.15.0/oha-linux-amd64-pgo'
            sha256 = '54008b400a990998824c4f91952c18565433452b3b71c2f4b47d9aebfaa34d9c'
        }
    }
}

$packageManifest = Get-Content -LiteralPath $sourcePackageManifest -Raw | ConvertFrom-Json
$packageId = [string]$packageManifest.packageId
$packageVersion = [string]$packageManifest.packageVersion
$stagingRoot = Join-Path $OutputRoot "$componentName/$RuntimeIdentifier"
$packageRoot = Join-Path $stagingRoot 'package'
$packageBin = Join-Path $packageRoot "bin/$RuntimeIdentifier"
$packageTools = Join-Path $packageRoot "tools/$RuntimeIdentifier"
$packageExecutors = Join-Path $packageRoot 'test-executors'
$packageLicenses = Join-Path $packageRoot 'third-party'
$executorPath = Join-Path $packageBin $rid.executorName
$ohaPath = Join-Path $packageTools $rid.ohaName

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $packageBin, $packageTools, $packageExecutors, $packageLicenses | Out-Null

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
    throw "Compiled HTTP/1 executor was not produced at $executorPath."
}

Invoke-WebRequest -Uri $rid.url -OutFile $ohaPath
$actualOhaHash = (Get-FileHash -LiteralPath $ohaPath -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actualOhaHash -ne $rid.sha256) {
    throw "oha asset '$($rid.assetName)' SHA-256 mismatch: expected $($rid.sha256), observed $actualOhaHash."
}

Copy-Item -LiteralPath $sourcePackageManifest -Destination (Join-Path $packageRoot 'protocol-lab-package.json') -Force
Copy-Item -LiteralPath $sourceExecutorManifest -Destination (Join-Path $packageExecutors 'go-http1-executor.yaml') -Force
Copy-Item -LiteralPath $sourceToolchain -Destination (Join-Path $packageRoot 'toolchain.json') -Force
Copy-Item -LiteralPath $sourceLicense -Destination (Join-Path $packageLicenses 'oha-LICENSE.txt') -Force

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
        name = 'oha'
        version = '1.15.0'
        asset = $rid.assetName
        downloadUrl = $rid.url
        sha256 = $rid.sha256
        license = 'MIT'
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
