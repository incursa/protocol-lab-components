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
$componentName = 's2n-quic-raw'
$componentRoot = Join-Path $Root "implementations/$componentName"
$sourceRoot = Join-Path $componentRoot 'source'
$sourcePackageManifest = Join-Path $componentRoot 'protocol-lab-package.json'
$sourceInternalManifest = Join-Path $componentRoot 'protocol-lab.internal.json'
$sourceImplementationManifest = Join-Path $componentRoot 'implementations/s2n-quic-raw.yaml'
$sourceRunSh = Join-Path $componentRoot 'run.sh'
$buildImage = 'rust@sha256:af306cfa71d987911a781c37b59d7d67d934f49684058f96cf72079c3626bfe0'

foreach ($path in @($sourcePackageManifest, $sourceInternalManifest, $sourceImplementationManifest, $sourceRunSh, (Join-Path $sourceRoot 'Cargo.toml'), (Join-Path $sourceRoot 'Cargo.lock'))) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required s2n-quic raw package input was not found: $path"
    }
}

$dockerSource = $sourceRoot.Replace('\', '/')
if (-not $SkipSmoke) {
    & docker run --rm --mount "type=bind,source=$dockerSource,target=/src" --workdir /src $buildImage cargo test --locked
    if ($LASTEXITCODE -ne 0) {
        throw "s2n-quic raw cargo test failed with exit code $LASTEXITCODE."
    }
}

& docker run --rm --mount "type=bind,source=$dockerSource,target=/src" --workdir /src $buildImage cargo build --release --locked
if ($LASTEXITCODE -ne 0) {
    throw "s2n-quic raw cargo build failed with exit code $LASTEXITCODE."
}

$packageManifest = Get-Content -LiteralPath $sourcePackageManifest -Raw | ConvertFrom-Json
$packageId = [string]$packageManifest.packageId
$packageVersion = [string]$packageManifest.packageVersion
$stagingRoot = Join-Path $OutputRoot $componentName
$packageRoot = Join-Path $stagingRoot 'package'
$packageBinLinux = Join-Path $packageRoot 'bin/linux-x64'
$packageImplementations = Join-Path $packageRoot 'implementations'
$artifactPath = Join-Path $OutputRoot "$packageId.$packageVersion.plabpkg"

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $packageBinLinux, $packageImplementations | Out-Null
Copy-Item -LiteralPath (Join-Path $sourceRoot 'target/release/protocol-lab-s2n-quic-raw') -Destination (Join-Path $packageBinLinux 's2n-quic-raw') -Force
Copy-Item -LiteralPath $sourcePackageManifest -Destination (Join-Path $packageRoot 'protocol-lab-package.json') -Force
Copy-Item -LiteralPath $sourceInternalManifest -Destination (Join-Path $packageRoot 'protocol-lab.internal.json') -Force
Copy-Item -LiteralPath $sourceImplementationManifest -Destination (Join-Path $packageImplementations 's2n-quic-raw.yaml') -Force
Copy-Item -LiteralPath (Join-Path $componentRoot 'README.md') -Destination (Join-Path $packageRoot 'README.md') -Force
Copy-Item -LiteralPath $sourceRunSh -Destination (Join-Path $packageRoot 'run.sh') -Force

& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath $packageRoot `
    -SourceComponentPath $componentRoot `
    -BuildConfiguration Release `
    -RuntimeIdentifier 'linux-x64' `
    -IncludeReadme `
    -PreparedPackageRoot

$archive = $null
try {
    $archive = [System.IO.Compression.ZipFile]::OpenRead($artifactPath)
    foreach ($entryName in @('README.md', 'bin/linux-x64/s2n-quic-raw', 'implementations/s2n-quic-raw.yaml', 'protocol-lab-package.json', 'protocol-lab.internal.json', 'run.sh')) {
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
