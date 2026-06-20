[CmdletBinding()]
param(
    [ValidateSet('Client', 'Server')]
    [string]$Mode = 'Client',
    [string]$Image = 'incursa-protocol-lab-ngtcp2-http3:0.1.2',
    [string]$Url = 'https://host.docker.internal:4433/bytes/1024',
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
    $shellCommand = "mkdir -p /tmp/www/bytes /tmp/certs /logs/qlog /logs/sslkeylog && perl -e 'binmode STDOUT; for (`$i = 0; `$i < 1024; `$i++) { print chr(`$i % 251) }' > /tmp/www/bytes/1024 && perl -e 'binmode STDOUT; for (`$i = 0; `$i < 65536; `$i++) { print chr(`$i % 251) }' > /tmp/www/bytes/65536 && perl -e 'binmode STDOUT; for (`$i = 0; `$i < 1048576; `$i++) { print chr(`$i % 251) }' > /tmp/www/bytes/1048576 && printf '%s\n' 'application/octet-stream 1024 65536 1048576' > /tmp/mime.types && printf '%s\n' '{`"protocol`":`"h3`",`"server`":`"ngtcp2-nghttp3`",`"implementation`":`"ngtcp2-http3`",`"utc`":`"static`",`"processId`":1}' > /tmp/www/status && openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 3650 -keyout /tmp/certs/priv.key -out /tmp/certs/cert.pem >/tmp/certs/openssl.out 2>/tmp/certs/openssl.err && /usr/local/bin/wsslserver --htdocs=/tmp/www --mime-types-file=/tmp/mime.types --qlog-dir=/logs/qlog --no-http-dump --timeout=15s --handshake-timeout=10s 0.0.0.0 4433 /tmp/certs/priv.key /tmp/certs/cert.pem"
    $dockerArgs = @('run', '--rm', '-p', "${Port}:4433/udp", '-v', "${artifactRoot}:/logs", '-e', 'QLOGDIR=/logs/qlog', '-e', 'SSLKEYLOGFILE=/logs/sslkeylog/keys.log')
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
