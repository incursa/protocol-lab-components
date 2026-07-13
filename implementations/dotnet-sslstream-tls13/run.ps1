[CmdletBinding()]
param(
    [int]$Port = 8443,
    [string]$Configuration = 'Release'
)

$ErrorActionPreference = 'Stop'
$env:PLAB_TLS_PORT = $Port.ToString([System.Globalization.CultureInfo]::InvariantCulture)
$env:PLAB_TLS_CERTIFICATE_PATH = Join-Path $PSScriptRoot 'certs/leaf.pem'
$env:PLAB_TLS_PRIVATE_KEY_PATH = Join-Path $PSScriptRoot 'certs/leaf-key.pem'
dotnet run --project (Join-Path $PSScriptRoot 'src/DotNetSslStreamTls13.csproj') --configuration $Configuration
