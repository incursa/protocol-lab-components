[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$SkipSmoke
)

$ErrorActionPreference = 'Stop'

$componentName = 'quic-go-raw'
$componentRoot = Join-Path $Root "implementations/$componentName"
$sourceRoot = Join-Path $componentRoot 'source'
$sourcePackageManifest = Join-Path $componentRoot 'protocol-lab-package.json'
$sourceInternalManifest = Join-Path $componentRoot 'protocol-lab.internal.json'
$sourceImplementationManifest = Join-Path $componentRoot 'implementations/quic-go-raw.yaml'

foreach ($path in @($sourcePackageManifest, $sourceInternalManifest, $sourceImplementationManifest, (Join-Path $sourceRoot 'go.mod'))) {
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
$stagingRoot = Join-Path $OutputRoot "$componentName/linux-x64"
$packageRoot = Join-Path $stagingRoot 'package'
$packageBin = Join-Path $packageRoot 'bin/linux-x64'
$packageImplementations = Join-Path $packageRoot 'implementations'
$artifactName = "$packageId.$packageVersion.plabpkg"
$artifactPath = Join-Path $OutputRoot $artifactName
$binaryPath = Join-Path $packageBin 'quic-go-raw'

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $packageBin, $packageImplementations | Out-Null

$oldGoOs = [Environment]::GetEnvironmentVariable('GOOS', 'Process')
$oldGoArch = [Environment]::GetEnvironmentVariable('GOARCH', 'Process')
$oldCgo = [Environment]::GetEnvironmentVariable('CGO_ENABLED', 'Process')
try {
    [Environment]::SetEnvironmentVariable('GOOS', 'linux', 'Process')
    [Environment]::SetEnvironmentVariable('GOARCH', 'amd64', 'Process')
    [Environment]::SetEnvironmentVariable('CGO_ENABLED', '0', 'Process')
    & go -C $sourceRoot build -trimpath -ldflags '-s -w -X main.quicGoVersion=v0.60.0' -o $binaryPath ./cmd/quic-go-raw
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE."
    }
}
finally {
    [Environment]::SetEnvironmentVariable('GOOS', $oldGoOs, 'Process')
    [Environment]::SetEnvironmentVariable('GOARCH', $oldGoArch, 'Process')
    [Environment]::SetEnvironmentVariable('CGO_ENABLED', $oldCgo, 'Process')
}

Copy-Item -LiteralPath $sourcePackageManifest -Destination (Join-Path $packageRoot 'protocol-lab-package.json') -Force
Copy-Item -LiteralPath $sourceInternalManifest -Destination (Join-Path $packageRoot 'protocol-lab.internal.json') -Force
Copy-Item -LiteralPath $sourceImplementationManifest -Destination (Join-Path $packageImplementations 'quic-go-raw.yaml') -Force
Copy-Item -LiteralPath (Join-Path $componentRoot 'README.md') -Destination (Join-Path $packageRoot 'README.md') -Force

Remove-Item -LiteralPath $artifactPath -Force -ErrorAction SilentlyContinue
Compress-Archive -Path (Join-Path $packageRoot '*') -DestinationPath $artifactPath -Force

Write-Host "Created $artifactPath"
