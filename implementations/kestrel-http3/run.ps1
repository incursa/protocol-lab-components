[CmdletBinding()]
param(
    [int]$Port = 8443
)

$ErrorActionPreference = 'Stop'
$env:PLAB_HTTP_PORT = [string]$Port

dotnet run --project (Join-Path $PSScriptRoot 'src/KestrelHttp3.csproj')
