[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
$componentRoot = Join-Path $Root 'implementations/apache-http1'
$packageRoot = Join-Path $OutputRoot 'apache-http1/package'
Remove-Item -LiteralPath (Split-Path -Parent $packageRoot) -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $packageRoot | Out-Null
Copy-Item -Path (Join-Path $componentRoot '*') -Destination $packageRoot -Recurse -Force
New-Item -ItemType Directory -Force -Path (Join-Path $packageRoot 'third-party') | Out-Null
Copy-Item -LiteralPath (Join-Path $Root 'LICENSE') -Destination (Join-Path $packageRoot 'third-party/apache-http-server-LICENSE.txt') -Force

& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath $packageRoot `
    -SourceComponentPath $componentRoot `
    -BuildConfiguration 'Release' `
    -RuntimeIdentifier 'portable-docker' `
    -PreparedPackageRoot `
    -IncludeReadme `
    -AllowDirtySource:$AllowDirtySource
