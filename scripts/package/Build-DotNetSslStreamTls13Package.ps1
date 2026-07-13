[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
& dotnet build (Join-Path $Root 'implementations/dotnet-sslstream-tls13/src/DotNetSslStreamTls13.csproj') --configuration Release
if ($LASTEXITCODE -ne 0) { throw "dotnet-sslstream-tls13 build failed with exit code $LASTEXITCODE." }
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -Root $Root `
    -OutputRoot $OutputRoot `
    -ComponentPath 'implementations/dotnet-sslstream-tls13' `
    -AllowDirtySource:$AllowDirtySource
