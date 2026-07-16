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
$componentName = 'quiche-raw'
$componentRoot = Join-Path $Root "implementations/$componentName"
$sourceRoot = Join-Path $componentRoot 'source'
$sourcePackageManifest = Join-Path $componentRoot 'protocol-lab-package.json'
$sourceInternalManifest = Join-Path $componentRoot 'protocol-lab.internal.json'
$sourceImplementationManifest = Join-Path $componentRoot 'implementations/quiche-raw.yaml'
$sourceRunSh = Join-Path $componentRoot 'run.sh'
$sourceCert = Join-Path $Root 'implementations/aioquic-http3/certs/leaf.pem'
$sourceKey = Join-Path $Root 'implementations/aioquic-http3/certs/leaf-key.pem'
$sourceLicense = Join-Path $componentRoot 'third-party/quiche-COPYING.txt'
$buildImage = 'incursa-protocol-lab-quiche-raw-build:0.1.0'

foreach ($path in @($sourcePackageManifest, $sourceInternalManifest, $sourceImplementationManifest, $sourceRunSh, $sourceCert, $sourceKey, $sourceLicense, (Join-Path $sourceRoot 'Cargo.toml'), (Join-Path $sourceRoot 'Cargo.lock'), (Join-Path $sourceRoot 'Dockerfile'), (Join-Path $sourceRoot 'src/main.rs'))) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required quiche raw package input was not found: $path"
    }
}
& docker build --pull --tag $buildImage --file (Join-Path $sourceRoot 'Dockerfile') $Root
if ($LASTEXITCODE -ne 0) {
    throw "quiche raw image build failed with exit code $LASTEXITCODE."
}

$packageManifest = Get-Content -LiteralPath $sourcePackageManifest -Raw | ConvertFrom-Json
$packageId = [string]$packageManifest.packageId
$packageVersion = [string]$packageManifest.packageVersion
$stagingRoot = Join-Path $OutputRoot $componentName
$packageRoot = Join-Path $stagingRoot 'package'
$packageBinLinux = Join-Path $packageRoot 'bin/linux-x64'
$packageImplementations = Join-Path $packageRoot 'implementations'
$packageCerts = Join-Path $packageRoot 'certs'
$packageLicenses = Join-Path $packageRoot 'third-party'
$artifactPath = Join-Path $OutputRoot "$packageId.$packageVersion.plabpkg"

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $packageBinLinux, $packageImplementations, $packageCerts, $packageLicenses | Out-Null
$containerId = (& docker create $buildImage).Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($containerId)) {
    throw 'Could not create quiche raw build container.'
}
try {
    & docker cp "${containerId}:/usr/local/bin/quiche-raw" (Join-Path $packageBinLinux 'quiche-raw')
    if ($LASTEXITCODE -ne 0) {
        throw "Could not extract quiche raw build output from container $containerId."
    }
}
finally {
    & docker rm $containerId | Out-Null
}

if (-not $SkipSmoke) {
    $smokeRoot = $packageRoot.Replace('\', '/')
    & docker run --rm --mount "type=bind,source=$smokeRoot,target=/package" --workdir /package `
        'debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818' `
        /package/bin/linux-x64/quiche-raw --self-test
    if ($LASTEXITCODE -ne 0) {
        throw "quiche raw extracted binary smoke failed with exit code $LASTEXITCODE."
    }
}

Copy-Item -LiteralPath $sourceCert -Destination (Join-Path $packageCerts 'leaf.pem') -Force
Copy-Item -LiteralPath $sourceKey -Destination (Join-Path $packageCerts 'leaf-key.pem') -Force
Copy-Item -LiteralPath $sourceLicense -Destination (Join-Path $packageLicenses 'quiche-COPYING.txt') -Force
Copy-Item -LiteralPath $sourcePackageManifest -Destination (Join-Path $packageRoot 'protocol-lab-package.json') -Force
Copy-Item -LiteralPath $sourceInternalManifest -Destination (Join-Path $packageRoot 'protocol-lab.internal.json') -Force
Copy-Item -LiteralPath $sourceImplementationManifest -Destination (Join-Path $packageImplementations 'quiche-raw.yaml') -Force
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
    foreach ($entryName in @('README.md', 'bin/linux-x64/quiche-raw', 'certs/leaf.pem', 'certs/leaf-key.pem', 'third-party/quiche-COPYING.txt', 'implementations/quiche-raw.yaml', 'protocol-lab-package.json', 'protocol-lab.internal.json', 'run.sh')) {
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
