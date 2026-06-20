[CmdletBinding()]
param(
    [ValidateSet('Client', 'Server')]
    [string]$Mode = 'Client',
    [string]$Image = 'incursa-protocol-lab-quiche-http3:0.1.3',
    [string]$Url = 'https://host.docker.internal:4433/bytes/1024',
    [string]$ConnectTo = 'host.docker.internal:4433',
    [string]$OutputRoot = 'artifacts/quiche-http3',
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
    $shellCommand = "mkdir -p /downloads /logs/qlog /logs/sslkeylog && quiche-client --http-version HTTP/3 --no-verify --connect-to $ConnectTo --dump-responses /downloads --dump-json --max-json-payload 0 $Url"
    $dockerArgs = @('run', '--rm', '-v', "${artifactRoot}:/downloads", '-e', 'QLOGDIR=/logs/qlog', '-e', 'SSLKEYLOGFILE=/logs/sslkeylog/keys.log')
}
else {
    $shellCommand = "mkdir -p /tmp/www/bytes /tmp/certs /logs/qlog /logs/sslkeylog && perl -e 'binmode STDOUT; for (`$i = 0; `$i < 1024; `$i++) { print chr(`$i % 251) }' > /tmp/www/bytes/1024 && perl -e 'binmode STDOUT; for (`$i = 0; `$i < 65536; `$i++) { print chr(`$i % 251) }' > /tmp/www/bytes/65536 && perl -e 'binmode STDOUT; for (`$i = 0; `$i < 1048576; `$i++) { print chr(`$i % 251) }' > /tmp/www/bytes/1048576 && printf '%s\n' '{`"protocol`":`"h3`",`"server`":`"quiche`",`"implementation`":`"quiche-http3`",`"utc`":`"static`",`"processId`":1}' > /tmp/www/status && openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 3650 -keyout /tmp/certs/priv.key -out /tmp/certs/cert.pem >/tmp/certs/openssl.out 2>/tmp/certs/openssl.err && quiche-server --listen 0.0.0.0:4433 --cert /tmp/certs/cert.pem --key /tmp/certs/priv.key --root /tmp/www --http-version HTTP/3"
    $dockerArgs = @('run', '--rm', '-p', "${Port}:4433/udp", '-e', 'QLOGDIR=/logs/qlog', '-e', 'SSLKEYLOGFILE=/logs/sslkeylog/keys.log')
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
    Write-Host "Planned quiche HTTP/3 $Mode command at $commandPath"
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
    throw "quiche HTTP/3 $Mode failed with exit code $exitCode. See $stderrPath"
}
