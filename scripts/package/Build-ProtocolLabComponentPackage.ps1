[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$ComponentPath,

    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages')
)

$ErrorActionPreference = 'Stop'

$componentRoot = if ([System.IO.Path]::IsPathRooted($ComponentPath)) {
    $ComponentPath
}
else {
    Join-Path $Root $ComponentPath
}

$componentRoot = (Resolve-Path $componentRoot).Path
$packageManifestPath = Join-Path $componentRoot 'protocol-lab-package.json'
if (-not (Test-Path -LiteralPath $packageManifestPath)) {
    throw "Component package root must contain protocol-lab-package.json: $componentRoot"
}

$packageManifest = Get-Content -LiteralPath $packageManifestPath -Raw | ConvertFrom-Json
$packageId = [string]$packageManifest.packageId
$packageVersion = [string]$packageManifest.packageVersion
if ([string]::IsNullOrWhiteSpace($packageId) -or [string]::IsNullOrWhiteSpace($packageVersion)) {
    throw "protocol-lab-package.json must declare packageId and packageVersion."
}

New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null
$stagingRoot = Join-Path $OutputRoot ("stage/" + ($packageId -replace '[^A-Za-z0-9_.-]', '_'))
$artifactPath = Join-Path $OutputRoot "$packageId.$packageVersion.plabpkg"

Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $stagingRoot | Out-Null

$excludedNames = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
@(
    'README.md',
    'package.protocol-lab.json',
    'bin',
    'obj',
    'artifacts',
    'packages'
) | ForEach-Object { [void]$excludedNames.Add($_) }

Get-ChildItem -LiteralPath $componentRoot -Recurse -File -Force | Where-Object {
    $relativePath = [System.IO.Path]::GetRelativePath($componentRoot, $_.FullName)
    $pathParts = $relativePath -split '[\\/]'
    -not ($pathParts | Where-Object { $excludedNames.Contains($_) })
} | ForEach-Object {
    $relativePath = [System.IO.Path]::GetRelativePath($componentRoot, $_.FullName)
    $destinationPath = Join-Path $stagingRoot $relativePath
    $destinationDirectory = Split-Path -Parent $destinationPath
    New-Item -ItemType Directory -Force -Path $destinationDirectory | Out-Null
    Copy-Item -LiteralPath $_.FullName -Destination $destinationPath -Force
}

Remove-Item -LiteralPath $artifactPath -Force -ErrorAction SilentlyContinue
Compress-Archive -Path (Join-Path $stagingRoot '*') -DestinationPath $artifactPath -Force

Write-Host "Created $artifactPath"
