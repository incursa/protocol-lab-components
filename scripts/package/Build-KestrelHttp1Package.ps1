[CmdletBinding()]
param(
    [ValidateSet('win-x64', 'linux-x64')]
    [string]$RuntimeIdentifier = 'win-x64',

    [string]$Configuration = 'Release',

    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages')
)

$ErrorActionPreference = 'Stop'

$componentName = 'kestrel-http1'
$componentRoot = Join-Path $Root "implementations/$componentName"
$project = Join-Path $componentRoot 'src/KestrelHttp1.csproj'
$sourcePackageManifest = Join-Path $componentRoot 'protocol-lab-package.json'
$sourceInternalManifest = Join-Path $componentRoot 'protocol-lab.internal.json'
$sourceImplementationManifest = Join-Path $componentRoot 'implementations/kestrel-http1.yaml'

if (-not (Test-Path -LiteralPath $project)) {
    throw "Project not found: $project"
}

$packageManifest = Get-Content -LiteralPath $sourcePackageManifest -Raw | ConvertFrom-Json
$packageId = [string]$packageManifest.packageId
$packageVersion = [string]$packageManifest.packageVersion
$stagingRoot = Join-Path $OutputRoot "$componentName/$RuntimeIdentifier"
$publishRoot = Join-Path $stagingRoot 'publish'
$packageRoot = Join-Path $stagingRoot 'package'
$packageBin = Join-Path $packageRoot 'bin'
$packageImplementations = Join-Path $packageRoot 'implementations'
$artifactName = "$packageId.$packageVersion.$RuntimeIdentifier.plabpkg"
$artifactPath = Join-Path $OutputRoot $artifactName

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $publishRoot, $packageBin, $packageImplementations | Out-Null

dotnet publish $project `
    --configuration $Configuration `
    --runtime $RuntimeIdentifier `
    --self-contained true `
    -p:PublishSingleFile=true `
    -p:EnableCompressionInSingleFile=true `
    -p:PublishTrimmed=false `
    --output $publishRoot

$executableName = if ($RuntimeIdentifier.StartsWith('win-', [System.StringComparison]::OrdinalIgnoreCase)) {
    'kestrel-http1.exe'
}
else {
    'kestrel-http1'
}

$publishedExecutable = Join-Path $publishRoot $executableName
if (-not (Test-Path -LiteralPath $publishedExecutable)) {
    throw "Published executable not found: $publishedExecutable"
}

Copy-Item -LiteralPath $publishedExecutable -Destination (Join-Path $packageBin $executableName)
Copy-Item -LiteralPath $sourceImplementationManifest -Destination (Join-Path $packageImplementations 'kestrel-http1.yaml')

$executionManifest = Get-Content -LiteralPath $sourceInternalManifest -Raw | ConvertFrom-Json
$executionManifest.environments = @(
    $executionManifest.environments | Where-Object {
        $_.os -eq $(if ($RuntimeIdentifier.StartsWith('win-', [System.StringComparison]::OrdinalIgnoreCase)) { 'windows' } else { 'linux' }) -and
        $_.arch -eq 'x64'
    }
)

if ($executionManifest.environments.Count -ne 1) {
    throw "Expected one package environment for runtime '$RuntimeIdentifier'."
}

$packageManifest | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath (Join-Path $packageRoot 'protocol-lab-package.json') -Encoding utf8
$executionManifest | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8

& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath $packageRoot `
    -SourceComponentPath $componentRoot `
    -ArtifactSuffix $RuntimeIdentifier `
    -BuildConfiguration $Configuration `
    -RuntimeIdentifier $RuntimeIdentifier `
    -PreparedPackageRoot
