[CmdletBinding()]
param(
    [string]$TargetUrl = $(if ($env:PLAB_TARGET_BASE_URL) { $env:PLAB_TARGET_BASE_URL } elseif ($env:PLAB_TARGET_URL) { $env:PLAB_TARGET_URL } else { 'https://host.docker.internal:8443/status' }),
    [int]$ExpectedStatus = 200,
    [int]$TimeoutSeconds = 15,
    [string]$Image = "ghcr.io/macbre/curl-http3",
    [string]$OutputRoot = $(if ($env:PLAB_ARTIFACT_DIR) { $env:PLAB_ARTIFACT_DIR } else { 'artifacts/curl-http3-client' }),
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'

$resolvedOutputRoot = if ([IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path (Get-Location) $OutputRoot }
New-Item -ItemType Directory -Force -Path $resolvedOutputRoot | Out-Null

$bodyPath = Join-Path $resolvedOutputRoot 'body.bin'
$stdoutPath = Join-Path $resolvedOutputRoot 'stdout.txt'
$stderrPath = Join-Path $resolvedOutputRoot 'stderr.txt'
$commandPath = Join-Path $resolvedOutputRoot 'command.txt'
$resultPath = Join-Path $resolvedOutputRoot 'result.json'
$containerTargetUrl = $TargetUrl -replace '://localhost(?=[:/])', '://host.docker.internal' -replace '://127\.0\.0\.1(?=[:/])', '://host.docker.internal'

$dockerArguments = @(
    'run', '--rm'
)
if ($IsLinux) {
    $dockerArguments += '--add-host=host.docker.internal:host-gateway'
}
$dockerArguments += @(
    '-v', "${resolvedOutputRoot}:/out",
    $Image,
    'curl',
    '--http3-only',
    '--max-time', ([string]$TimeoutSeconds),
    '--insecure',
    '--silent',
    '--show-error',
    '--output', '/out/body.bin',
    '--write-out', '%{http_code}',
    $containerTargetUrl
)

Set-Content -LiteralPath $commandPath -Value ("docker " + ($dockerArguments -join ' '))

if ($PlanOnly) {
    [ordered]@{
        status = 'planned'
        targetUrl = $TargetUrl
        expectedStatus = $ExpectedStatus
        image = $Image
        command = Get-Content -LiteralPath $commandPath -Raw
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath
    Write-Host "Planned curl HTTP/3 executor command at $commandPath"
    return
}

& docker @dockerArguments > $stdoutPath 2> $stderrPath
$exitCode = $LASTEXITCODE
$statusText = (Get-Content -LiteralPath $stdoutPath -Raw).Trim()
$actualStatus = 0
if (-not [int]::TryParse($statusText, [ref]$actualStatus)) {
    $actualStatus = -1
}

$result = [ordered]@{
    status = if ($exitCode -eq 0 -and $actualStatus -eq $ExpectedStatus) { 'passed' } else { 'failed' }
    targetUrl = $TargetUrl
    expectedStatus = $ExpectedStatus
    actualStatus = $actualStatus
    exitCode = $exitCode
    image = $Image
    bodyPath = $bodyPath
    stdoutPath = $stdoutPath
    stderrPath = $stderrPath
}
$result | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath

if ($exitCode -ne 0) {
    throw "curl HTTP/3 executor failed with exit code $exitCode. See $stderrPath"
}

if ($actualStatus -ne $ExpectedStatus) {
    throw "curl HTTP/3 executor observed status $actualStatus; expected $ExpectedStatus."
}

$executorId = if ($env:PLAB_EXECUTOR_ID) { $env:PLAB_EXECUTOR_ID } else { 'curl-http3-client' }
$executorVersion = if ($env:PLAB_EXECUTOR_VERSION) { $env:PLAB_EXECUTOR_VERSION } else { '0.1.7' }
$connections = if ($env:PLAB_CONNECTIONS) { [int]$env:PLAB_CONNECTIONS } else { 1 }
$concurrency = if ($env:PLAB_CONCURRENCY) { [int]$env:PLAB_CONCURRENCY } else { 1 }
$streams = if ($env:PLAB_STREAMS_PER_CONNECTION) { [int]$env:PLAB_STREAMS_PER_CONNECTION } else { 1 }
$duration = if ($env:PLAB_DURATION_SECONDS) { [int]$env:PLAB_DURATION_SECONDS } else { 0 }
$warmup = if ($env:PLAB_WARMUP_SECONDS) { [int]$env:PLAB_WARMUP_SECONDS } else { 0 }
$bodyBytes = if (Test-Path -LiteralPath $bodyPath) { (Get-Item -LiteralPath $bodyPath).Length } else { 0 }
[ordered]@{
    schemaVersion = 'protocol-lab.http-executor-result.v1'
    executor = [ordered]@{ id = $executorId; version = $executorVersion }
    loadGenerator = [ordered]@{ id = $executorId; version = $executorVersion }
    validation = [ordered]@{ status = 'passed' }
    protocolProof = [ordered]@{ requestedProtocol = 'h3'; observedProtocol = 'h3'; exactProtocolMatched = $true; fallbackDetected = $false }
    requestedLoad = [ordered]@{ connections = $connections; concurrency = $concurrency; streamsPerConnection = $streams; durationSeconds = $duration; warmupSeconds = $warmup }
    effectiveLoad = [ordered]@{ connections = 1; concurrency = 1; streamsPerConnection = 1; durationSeconds = 0; warmupSeconds = 0 }
    metrics = [ordered]@{ totalRequests = 1; successfulRequests = 1; failedRequests = 0; timeoutRequests = 0; requestsPerSecond = 1; bytesSent = 0; bytesReceived = $bodyBytes; throughputBytesPerSecond = 0; latencyMeanMs = 0; latencyP50Ms = 0; latencyP75Ms = 0; latencyP90Ms = 0; latencyP95Ms = 0; latencyP99Ms = 0; statusCodeCounts = [ordered]@{ ([string]$actualStatus) = 1 } }
    warnings = @('Diagnostic single-request peer characterization; no benchmark payload or latency claim is made.')
} | ConvertTo-Json -Depth 8 -Compress | Write-Output
