[CmdletBinding()]
param(
    [string]$ScenarioId = $env:PLAB_SCENARIO_ID,
    [string]$LoadProfileId = $env:PLAB_LOAD_PROFILE_ID,
    [string]$TargetUrl = $(if ($env:PLAB_TARGET_BASE_URL) { $env:PLAB_TARGET_BASE_URL } else { 'https://host.docker.internal:4435/websocket-proof' }),
    [string]$Image = 'incursa-protocol-lab-aioquic-rfc9220-websocket:0.3.0',
    [string]$TargetImageId = $env:PLAB_TARGET_IMAGE_ID,
    [string]$ScenarioPackageSha256 = $env:PLAB_SCENARIO_PACKAGE_SHA256,
    [string]$ExecutorPackageSha256 = $env:PLAB_EXECUTOR_PACKAGE_SHA256,
    [string]$TargetPackageSha256 = $env:PLAB_TARGET_PACKAGE_SHA256,
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
function Write-TerminalResult([string]$status, [string]$reason) {
    $value = [ordered]@{ schemaVersion = 'protocol-lab.rfc9220-executor-result.v1'; executorId = 'aioquic-rfc9220-websocket'; executorVersion = '0.3.0'; loadGeneratorId = 'aioquic-rfc9220-websocket-load'; loadGeneratorVersion = '0.3.0'; parserId = 'protocol-lab-rfc9220-json'; scenarioId = $ScenarioId; loadProfileId = $LoadProfileId; status = $status; passed = $false; reason = $reason }
    $json = $value | ConvertTo-Json -Depth 6 -Compress
    $json | Set-Content $resultPath -Encoding utf8NoBOM
    Write-Output $json
}
if ($ScenarioId -in $unsupported) { Write-TerminalResult 'unsupported' 'scenario identity belongs to another WebSocket binding or an unimplemented diagnostic'; exit 3 }
if ([string]::IsNullOrWhiteSpace($ScenarioId) -or $ScenarioId -notin $supported) { Write-TerminalResult 'unknown' 'unknown scenario identity'; exit 2 }

$fragmented = $ScenarioId -eq 'http3.websocket.rfc9220.fragmented-binary-echo'
$expectedProfile = if ($fragmented) { 'diagnostic' } else { 'websocket-smoke' }
if ([string]::IsNullOrWhiteSpace($LoadProfileId)) { $LoadProfileId = $expectedProfile }
if ($LoadProfileId -ne $expectedProfile) { Write-TerminalResult 'unsupported' "scenario requires exact load profile $expectedProfile"; exit 3 }
$concurrency = if ($fragmented) { 8 } else { 1 }
$warmup = 1
$duration = if ($fragmented) { 10 } else { 5 }
$cooldown = if ($fragmented) { 1 } else { 0 }
$timeout = if ($fragmented) { 10 } else { 5 }
if ($PlanOnly) { Write-TerminalResult 'planned' "exact profile ${LoadProfileId}: connections=1 concurrency=$concurrency warmup=$warmup duration=$duration cooldown=$cooldown timeout=$timeout; package and image digests are required at execution"; return }
foreach ($pair in @(@('scenario package digest', $ScenarioPackageSha256), @('executor package digest', $ExecutorPackageSha256), @('target package digest', $TargetPackageSha256))) {
    if ($pair[1] -notmatch '^[0-9a-fA-F]{64}$') { throw "$($pair[0]) is required for package-backed execution." }
}
if ($TargetImageId -notmatch '^sha256:[0-9a-fA-F]{64}$') { throw 'Immutable target image ID is required for package-backed execution.' }

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
$buildStdoutPath = Join-Path $resolvedOutputRoot 'image-build.stdout.log'
$buildStderrPath = Join-Path $resolvedOutputRoot 'image-build.stderr.log'

Push-Location $componentRoot
try {
    if (-not $SkipBuild) {
        $buildArgs = @('build', '--build-arg', 'AIOQUIC_VERSION=1.3.0', '-f', 'docker/aioquic-rfc9220-websocket.Dockerfile', '-t', $Image, '.')
        & docker @buildArgs > $buildStdoutPath 2> $buildStderrPath
        if ($LASTEXITCODE -ne 0) { throw "aioquic RFC9220 Docker build failed with exit code $LASTEXITCODE" }
    }
    $executorImageId = (& docker image inspect --format '{{.Id}}' $Image).Trim()
    if ($LASTEXITCODE -ne 0 -or $executorImageId -notmatch '^sha256:[0-9a-fA-F]{64}$') { throw 'Immutable executor image ID could not be resolved.' }
    $dockerArgs = @('run', '--rm', '--add-host=host.docker.internal:host-gateway')
    if ($DockerNetwork) { $dockerArgs += @('--network', $DockerNetwork) }
    $dockerArgs += @('-v', "${resolvedOutputRoot}:/proof", '-e', 'QLOGDIR=/proof/qlog', '-e', 'SSLKEYLOGFILE=/proof/sslkeylog/keys.log', $Image, '/usr/local/bin/aioquic-http3-websocket-client', $TargetUrl, '/proof/client-result.json', '--scenario-id', $ScenarioId, '--load-profile-id', $LoadProfileId, '--concurrency', [string]$concurrency, '--warmup', [string]$warmup, '--duration', [string]$duration, '--cooldown', [string]$cooldown, '--timeout', [string]$timeout, '--scenario-package-sha256', $ScenarioPackageSha256.ToLowerInvariant(), '--executor-package-sha256', $ExecutorPackageSha256.ToLowerInvariant(), '--target-package-sha256', $TargetPackageSha256.ToLowerInvariant(), '--executor-image-id', $executorImageId, '--target-image-id', $TargetImageId)
    ('docker ' + ($dockerArgs -join ' ')) | Set-Content -LiteralPath $commandPath -Encoding utf8NoBOM
    & docker @dockerArgs > $stdoutPath 2> $stderrPath
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0 -or -not (Test-Path $clientResultPath)) { throw "aioquic RFC9220 scenario $ScenarioId failed with exit code $exitCode" }
    $client = Get-Content $clientResultPath -Raw | ConvertFrom-Json
    if (-not $client.passed -or $client.status -ne 'passed' -or $client.scenarioId -ne $ScenarioId -or $client.loadProfileId -ne $LoadProfileId -or $client.executorId -ne 'aioquic-rfc9220-websocket' -or $client.executorVersion -ne '0.3.0' -or $client.loadGeneratorId -ne 'aioquic-rfc9220-websocket-load' -or $client.parserId -ne 'protocol-lab-rfc9220-json') { throw 'client result identity or validation status mismatch' }
    Copy-Item -LiteralPath $clientResultPath -Destination $resultPath -Force
    Get-Content $resultPath -Raw
}
finally { Pop-Location }
