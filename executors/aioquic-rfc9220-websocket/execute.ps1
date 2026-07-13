[CmdletBinding()]
param(
    [string]$ScenarioId = $env:PLAB_SCENARIO_ID,
    [string]$TargetUrl = $(if ($env:PLAB_TARGET_BASE_URL) { $env:PLAB_TARGET_BASE_URL } else { 'https://host.docker.internal:4435/websocket-proof' }),
    [double]$TimeoutSeconds = 20,
    [string]$Image = 'incursa-protocol-lab-aioquic-rfc9220-websocket:0.2.0',
    [string]$OutputRoot = $(if ($env:PLAB_ARTIFACT_DIR) { $env:PLAB_ARTIFACT_DIR } else { 'artifacts/aioquic-rfc9220-websocket' }),
    [string]$DockerNetwork = '',
    [switch]$SkipBuild,
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'
$supported = @(
    'http3.websocket.rfc9220.extended-connect',
    'http3.websocket.rfc9220.control-frames',
    'http3.websocket.rfc9220.text-echo',
    'http3.websocket.rfc9220.binary-echo',
    'http3.websocket.rfc9220.close',
    'http3.websocket.rfc9220.fragmented-binary-echo'
)
$unsupported = @(
    'websocket.echo',
    'http1.websocket.rfc6455.cleartext.upgrade', 'http1.websocket.rfc6455.cleartext.control-frames', 'http1.websocket.rfc6455.cleartext.text-echo', 'http1.websocket.rfc6455.cleartext.binary-echo', 'http1.websocket.rfc6455.cleartext.close',
    'http1.websocket.rfc6455.tls.upgrade', 'http1.websocket.rfc6455.tls.control-frames', 'http1.websocket.rfc6455.tls.text-echo', 'http1.websocket.rfc6455.tls.binary-echo', 'http1.websocket.rfc6455.tls.close', 'http1.websocket.rfc6455.tls.subprotocol-text-echo', 'http1.websocket.rfc6455.tls.permessage-deflate-binary-echo',
    'http2.websocket.rfc8441.extended-connect', 'http2.websocket.rfc8441.control-frames', 'http2.websocket.rfc8441.text-echo', 'http2.websocket.rfc8441.binary-echo', 'http2.websocket.rfc8441.close', 'http2.websocket.rfc8441.multi-message-text-echo'
)

$componentRoot = $PSScriptRoot
$resolvedOutputRoot = if ([IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $componentRoot $OutputRoot }
New-Item -ItemType Directory -Force -Path $resolvedOutputRoot | Out-Null
$resultPath = Join-Path $resolvedOutputRoot 'result.json'
if ($ScenarioId -in $unsupported) {
    [ordered]@{ schemaVersion = 'protocol-lab.aioquic-rfc9220-result.v2'; scenarioId = $ScenarioId; status = 'unsupported'; reason = 'scenario identity belongs to another WebSocket binding or an unimplemented diagnostic' } | ConvertTo-Json -Depth 4 | Set-Content $resultPath -Encoding utf8NoBOM
    exit 3
}
if ([string]::IsNullOrWhiteSpace($ScenarioId) -or $ScenarioId -notin $supported) {
    [ordered]@{ schemaVersion = 'protocol-lab.aioquic-rfc9220-result.v2'; scenarioId = $ScenarioId; status = 'unknown'; reason = 'unknown scenario identity' } | ConvertTo-Json -Depth 4 | Set-Content $resultPath -Encoding utf8NoBOM
    exit 2
}

$builder = [UriBuilder]$TargetUrl
if ([string]::IsNullOrWhiteSpace($builder.Path) -or $builder.Path -eq '/') { $builder.Path = '/websocket-proof' }
if ($builder.Host -in @('localhost', '127.0.0.1', '::1')) { $builder.Host = 'host.docker.internal' }
$TargetUrl = $builder.Uri.AbsoluteUri
$qlogRoot = Join-Path $resolvedOutputRoot 'qlog'
$sslKeyLogRoot = Join-Path $resolvedOutputRoot 'sslkeylog'
New-Item -ItemType Directory -Force -Path $qlogRoot, $sslKeyLogRoot | Out-Null
$clientResultPath = Join-Path $resolvedOutputRoot 'client-result.json'
$stdoutPath = Join-Path $resolvedOutputRoot 'load.stdout.log'
$stderrPath = Join-Path $resolvedOutputRoot 'load.stderr.log'
$commandPath = Join-Path $resolvedOutputRoot 'command.txt'
$buildStdoutPath = Join-Path $resolvedOutputRoot 'build.stdout.log'
$buildStderrPath = Join-Path $resolvedOutputRoot 'build.stderr.log'

Push-Location $componentRoot
try {
    $commands = [Collections.Generic.List[string]]::new()
    if (-not $SkipBuild) {
        $buildArgs = @('build', '--build-arg', 'AIOQUIC_VERSION=1.3.0', '-f', 'docker/aioquic-rfc9220-websocket.Dockerfile', '-t', $Image, '.')
        $commands.Add('docker ' + ($buildArgs -join ' '))
    }
    $dockerArgs = @('run', '--rm', '--add-host=host.docker.internal:host-gateway')
    if ($DockerNetwork) { $dockerArgs += @('--network', $DockerNetwork) }
    $dockerArgs += @('-v', "${resolvedOutputRoot}:/proof", '-e', 'QLOGDIR=/proof/qlog', '-e', 'SSLKEYLOGFILE=/proof/sslkeylog/keys.log', $Image, '/usr/local/bin/aioquic-http3-websocket-client', $TargetUrl, '/proof/client-result.json', '--scenario-id', $ScenarioId, '--timeout', [string]$TimeoutSeconds)
    $commands.Add('docker ' + ($dockerArgs -join ' '))
    Set-Content -LiteralPath $commandPath -Value $commands -Encoding utf8NoBOM
    if ($PlanOnly) {
        [ordered]@{ schemaVersion = 'protocol-lab.aioquic-rfc9220-result.v2'; scenarioId = $ScenarioId; status = 'planned'; targetUrl = $TargetUrl; image = $Image; executor = @{ id = 'aioquic-rfc9220-websocket'; version = '0.2.0' }; commands = $commands.ToArray() } | ConvertTo-Json -Depth 6 | Set-Content $resultPath -Encoding utf8NoBOM
        return
    }
    if (-not $SkipBuild) {
        & docker @buildArgs > $buildStdoutPath 2> $buildStderrPath
        if ($LASTEXITCODE -ne 0) { throw "aioquic RFC9220 Docker build failed with exit code $LASTEXITCODE" }
    }
    & docker @dockerArgs > $stdoutPath 2> $stderrPath
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0 -or -not (Test-Path $clientResultPath)) { throw "aioquic RFC9220 scenario $ScenarioId failed with exit code $exitCode" }
    $client = Get-Content $clientResultPath -Raw | ConvertFrom-Json
    if ($client.status -ne 'passed' -or $client.scenarioId -ne $ScenarioId) { throw 'client result identity or validation status mismatch' }
    [ordered]@{
        schemaVersion = 'protocol-lab.aioquic-rfc9220-result.v2'
        scenarioId = $ScenarioId
        status = 'passed'
        authorityCommit = '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'
        executor = @{ id = 'aioquic-rfc9220-websocket'; version = '0.2.0' }
        validation = @{ status = 'passed' }
        protocolProof = $client.protocolProof
        metrics = $client.metrics
        warnings = @('Local package-backed RFC 9220 evidence is diagnostic and non-publishable.')
    } | ConvertTo-Json -Depth 20 | Set-Content $resultPath -Encoding utf8NoBOM
    [ordered]@{ id = 'aioquic-rfc9220-websocket'; version = '0.2.0'; aioquicVersion = '1.3.0'; role = 'test-executor' } | ConvertTo-Json | Set-Content (Join-Path $resolvedOutputRoot 'executor-identity.json') -Encoding utf8NoBOM
}
finally { Pop-Location }
