[CmdletBinding()]
param(
    [int]$Port = 8080,
    [string]$Configuration = 'Release'
)

$ErrorActionPreference = 'Stop'

$project = Join-Path $PSScriptRoot 'src/KestrelHttp1.csproj'
$env:PLAB_HTTP_PORT = $Port.ToString([System.Globalization.CultureInfo]::InvariantCulture)

dotnet run --project $project --configuration $Configuration --no-launch-profile
