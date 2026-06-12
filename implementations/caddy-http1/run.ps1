[CmdletBinding()]
param(
    [int]$Port = 8080
)

$ErrorActionPreference = 'Stop'
$env:PLAB_HTTP_PORT = $Port.ToString([System.Globalization.CultureInfo]::InvariantCulture)
if (-not (Get-Command caddy -ErrorAction SilentlyContinue)) {
    throw "caddy executable was not found on PATH."
}

caddy run --config (Join-Path $PSScriptRoot 'Caddyfile') --adapter caddyfile
