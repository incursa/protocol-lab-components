[CmdletBinding()]
param(
    [int]$Port = 8082,
    [string]$Configuration = 'Release'
)

$ErrorActionPreference = 'Stop'
$env:PLAB_HTTP_PORT = $Port.ToString([System.Globalization.CultureInfo]::InvariantCulture)
dotnet run --project (Join-Path $PSScriptRoot 'src/KestrelHttp2.csproj') --configuration $Configuration --no-launch-profile
