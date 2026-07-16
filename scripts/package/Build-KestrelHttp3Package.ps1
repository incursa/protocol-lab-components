[CmdletBinding()]
param(
    [string]$Configuration = 'Release',

    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),

    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
$componentRoot = Join-Path $Root 'implementations/kestrel-http3'
$stagingRoot = Join-Path $OutputRoot 'tmp/kestrel-http3-linux-x64'
$publishRoot = Join-Path $stagingRoot 'publish'
$packageRoot = Join-Path $stagingRoot 'package'
$packageBin = Join-Path $packageRoot 'bin'
$packageImplementations = Join-Path $packageRoot 'implementations'

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $publishRoot, $packageBin, $packageImplementations | Out-Null

dotnet publish (Join-Path $componentRoot 'src/KestrelHttp3.csproj') `
    --configuration $Configuration `
    --runtime linux-x64 `
    --self-contained true `
    -p:PublishSingleFile=true `
    -p:EnableCompressionInSingleFile=true `
    -p:PublishTrimmed=false `
    -p:DebugType=None `
    -p:DebugSymbols=false `
    --output $publishRoot `
    --nologo
if ($LASTEXITCODE -ne 0) {
    throw "Kestrel HTTP/3 Linux x64 publish failed with exit code $LASTEXITCODE."
}

$published = Join-Path $publishRoot 'KestrelHttp3'
if (-not (Test-Path -LiteralPath $published -PathType Leaf)) {
    throw "Published Kestrel HTTP/3 executable was missing: $published"
}

Copy-Item -LiteralPath $published -Destination (Join-Path $packageBin 'kestrel-http3')
Copy-Item -LiteralPath (Join-Path $componentRoot 'protocol-lab-package.json') -Destination $packageRoot
Copy-Item -LiteralPath (Join-Path $componentRoot 'protocol-lab.internal.json') -Destination $packageRoot
Copy-Item -LiteralPath (Join-Path $componentRoot 'implementations/kestrel-http3.yaml') -Destination $packageImplementations

& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath $packageRoot `
    -SourceComponentPath $componentRoot `
    -BuildConfiguration $Configuration `
    -RuntimeIdentifier 'linux-x64' `
    -PreparedPackageRoot `
    -AllowDirtySource:$AllowDirtySource
