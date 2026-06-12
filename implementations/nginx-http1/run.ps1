[CmdletBinding()]
param(
    [int]$Port = 8080
)

$ErrorActionPreference = 'Stop'
if (-not (Get-Command nginx -ErrorAction SilentlyContinue)) {
    throw "nginx executable was not found on PATH."
}

$runRoot = Join-Path ([System.IO.Path]::GetTempPath()) "protocol-lab-nginx-http1-$PID"
New-Item -ItemType Directory -Force -Path $runRoot | Out-Null
$config = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'nginx.conf.template') -Raw
$config = $config.Replace('${PLAB_HTTP_PORT}', $Port.ToString([System.Globalization.CultureInfo]::InvariantCulture))
$configPath = Join-Path $runRoot 'nginx.conf'
Set-Content -LiteralPath $configPath -Value $config -Encoding utf8

nginx -p $runRoot -c $configPath -g 'daemon off;'
