[CmdletBinding()]
param(
    [string]$Image = 'incursa-protocol-lab-xquic-http3:0.1.1',
    [int]$Port = 4433,
    [string]$OutputRoot = 'artifacts/xquic-http3',
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'
$artifactRoot = if ([IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $PSScriptRoot $OutputRoot }
New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
$shell = "set -euo pipefail; cd /xquic_bin; mkdir -p /tmp/www /tmp/certs /logs; printf 'xquic-http3-peer\n' >/tmp/www/index.html; openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 3650 -keyout /tmp/certs/priv.key -out /tmp/certs/cert.pem >/logs/openssl.stdout 2>/logs/openssl.stderr; cp /tmp/certs/priv.key server.key; cp /tmp/certs/cert.pem server.crt; exec ./demo_server -l d -L /logs/server.log -p 4433 -D /tmp/www -i -M"
$args = @('run', '--rm', '-p', "${Port}:4433/udp", '--entrypoint', 'bash', $Image, '-lc', $shell)
Set-Content -LiteralPath (Join-Path $artifactRoot 'command.txt') -Value ('docker ' + ($args -join ' '))
if ($PlanOnly) {
    [ordered]@{ status = 'planned'; image = $Image; port = $Port } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $artifactRoot 'result.json')
    return
}
& docker @args > (Join-Path $artifactRoot 'stdout.txt') 2> (Join-Path $artifactRoot 'stderr.txt')
$exitCode = $LASTEXITCODE
[ordered]@{ status = if ($exitCode -eq 0) { 'passed' } else { 'failed' }; image = $Image; exitCode = $exitCode } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $artifactRoot 'result.json')
if ($exitCode -ne 0) { throw "XQUIC HTTP/3 server failed with exit code $exitCode." }
