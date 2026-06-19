[CmdletBinding()]
param(
    [ValidateSet('Client', 'Server')]
    [string]$Mode = 'Client',
    [string]$Image = 'ghcr.io/ngtcp2/ngtcp2-interop:latest',
    [string]$Url = 'https://host.docker.internal:4433/small.txt',
    [string]$HostName = 'host.docker.internal',
    [int]$PeerPort = 4433,
    [string]$OutputRoot = 'artifacts/ngtcp2-http3',
    [string]$WwwRoot = 'www',
    [string]$CertPath = 'certs/cert.pem',
    [string]$KeyPath = 'certs/priv.key',
    [int]$Port = 4433,
    [string]$DockerNetwork = '',
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'

$componentRoot = $PSScriptRoot
$artifactRoot = if ([System.IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $componentRoot $OutputRoot }
New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null

$commandPath = Join-Path $artifactRoot 'command.txt'
$stdoutPath = Join-Path $artifactRoot 'stdout.txt'
$stderrPath = Join-Path $artifactRoot 'stderr.txt'
$resultPath = Join-Path $artifactRoot 'result.json'

if ($Mode -eq 'Client') {
    $shellCommand = "mkdir -p /downloads /logs/qlog /logs/sslkeylog && wsslclient --download=/downloads --exit-on-all-streams-close --timeout=15s --handshake-timeout=10s --no-http-dump --qlog-dir=/logs/qlog $HostName $PeerPort $Url"
    $dockerArgs = @('run', '--rm', '-v', "${artifactRoot}:/downloads", '-v', "${artifactRoot}:/logs", '-e', 'QLOGDIR=/logs/qlog', '-e', 'SSLKEYLOGFILE=/logs/sslkeylog/keys.log')
}
else {
    $wwwFullPath = if ([System.IO.Path]::IsPathRooted($WwwRoot)) { $WwwRoot } else { Join-Path $componentRoot $WwwRoot }
    $certFullPath = if ([System.IO.Path]::IsPathRooted($CertPath)) { $CertPath } else { Join-Path $componentRoot $CertPath }
    $keyFullPath = if ([System.IO.Path]::IsPathRooted($KeyPath)) { $KeyPath } else { Join-Path $componentRoot $KeyPath }
    $certDirectory = Split-Path -Parent $certFullPath
    $certFileName = Split-Path -Leaf $certFullPath
    $keyFileName = Split-Path -Leaf $keyFullPath
    $shellCommand = "mkdir -p /logs/qlog /logs/sslkeylog && /usr/local/bin/wsslserver --htdocs=/www --qlog-dir=/logs/qlog --no-http-dump --timeout=15s --handshake-timeout=10s 0.0.0.0 4433 /certs/$keyFileName /certs/$certFileName"
    $dockerArgs = @('run', '--rm', '-p', "${Port}:4433/udp", '-v', "${wwwFullPath}:/www:ro", '-v', "${certDirectory}:/certs:ro", '-v', "${artifactRoot}:/logs", '-e', 'QLOGDIR=/logs/qlog', '-e', 'SSLKEYLOGFILE=/logs/sslkeylog/keys.log')
}

if (-not [string]::IsNullOrWhiteSpace($DockerNetwork)) {
    $dockerArgs += @('--network', $DockerNetwork)
}

$dockerArgs += @('--entrypoint', '/bin/sh', $Image, '-lc', $shellCommand)
Set-Content -LiteralPath $commandPath -Value ('docker ' + ($dockerArgs -join ' '))

if ($PlanOnly) {
    [ordered]@{
        status = 'planned'
        mode = $Mode
        image = $Image
        command = Get-Content -LiteralPath $commandPath -Raw
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath
    Write-Host "Planned ngtcp2/nghttp3 HTTP/3 $Mode command at $commandPath"
    return
}

& docker @dockerArgs > $stdoutPath 2> $stderrPath
$exitCode = $LASTEXITCODE
[ordered]@{
    status = if ($exitCode -eq 0) { 'passed' } else { 'failed' }
    mode = $Mode
    image = $Image
    exitCode = $exitCode
    stdoutPath = $stdoutPath
    stderrPath = $stderrPath
} | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath

if ($exitCode -ne 0) {
    throw "ngtcp2/nghttp3 HTTP/3 $Mode failed with exit code $exitCode. See $stderrPath"
}
