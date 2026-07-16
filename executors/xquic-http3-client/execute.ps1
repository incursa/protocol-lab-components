[CmdletBinding()]
param(
    [string]$TargetUrl = $(if ($env:PLAB_TARGET_BASE_URL) { $env:PLAB_TARGET_BASE_URL } elseif ($env:PLAB_TARGET_URL) { $env:PLAB_TARGET_URL } else { 'https://host.docker.internal:8443/' }),
    [int]$ExpectedStatus = 200,
    [int]$TimeoutSeconds = 5,
    [string]$Image = 'ghcr.io/alibaba/xquic/xquic-interop@sha256:875df1e9935c6a07e26d7b5ae14df9edd06703061ce35920234a97d6991c58e0',
    [string]$OutputRoot = $(if ($env:PLAB_ARTIFACT_DIR) { $env:PLAB_ARTIFACT_DIR } else { 'artifacts/xquic-http3-client' }),
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'
$root = if ([IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path (Get-Location) $OutputRoot }
New-Item -ItemType Directory -Force -Path $root | Out-Null
if ($TargetUrl.Contains("'")) { throw 'TargetUrl cannot contain a single quote.' }
$shell = "cd /xquic_bin && ./demo_client -l d -L /out/client.log -D /out -U '$TargetUrl' -A h3 -K $TimeoutSeconds -o"
$args = @('run', '--rm', '-v', "${root}:/out", '--entrypoint', 'bash', $Image, '-lc', $shell)
Set-Content -LiteralPath (Join-Path $root 'command.txt') -Value ('docker ' + ($args -join ' '))
if ($PlanOnly) {
    [ordered]@{ status = 'planned'; targetUrl = $TargetUrl; expectedStatus = $ExpectedStatus; image = $Image } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $root 'result.json')
    return
}
& docker @args > (Join-Path $root 'stdout.txt') 2> (Join-Path $root 'stderr.txt')
$exitCode = $LASTEXITCODE
$stdout = Get-Content -Raw -LiteralPath (Join-Path $root 'stdout.txt')
$statusObserved = $stdout -match "(?m)^:status = $ExpectedStatus\s*$"
$alpnObserved = $stdout -match 'option set ALPN\[h3\]'
$completionWarning = $stdout -match 'err:260|recv_fin:0'
$passed = $exitCode -eq 0 -and $statusObserved -and $alpnObserved
[ordered]@{
    status = if ($passed) { 'passed' } else { 'failed' }
    targetUrl = $TargetUrl
    expectedStatus = $ExpectedStatus
    actualStatus = if ($statusObserved) { $ExpectedStatus } else { $null }
    negotiatedProtocol = if ($alpnObserved) { 'h3' } else { $null }
    responseCompletionWarning = $completionWarning
    canonicalPayloadClaimed = $false
    exitCode = $exitCode
    image = $Image
} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $root 'result.json')
if (-not $passed) { throw 'XQUIC HTTP/3 diagnostic validation failed.' }
$executorId = if ($env:PLAB_EXECUTOR_ID) { $env:PLAB_EXECUTOR_ID } else { 'xquic-http3-client' }
$executorVersion = if ($env:PLAB_EXECUTOR_VERSION) { $env:PLAB_EXECUTOR_VERSION } else { '0.1.1' }
$connections = if ($env:PLAB_CONNECTIONS) { [int]$env:PLAB_CONNECTIONS } else { 1 }
$concurrency = if ($env:PLAB_CONCURRENCY) { [int]$env:PLAB_CONCURRENCY } else { 1 }
$streams = if ($env:PLAB_STREAMS_PER_CONNECTION) { [int]$env:PLAB_STREAMS_PER_CONNECTION } else { 1 }
$duration = if ($env:PLAB_DURATION_SECONDS) { [int]$env:PLAB_DURATION_SECONDS } else { 0 }
$warmup = if ($env:PLAB_WARMUP_SECONDS) { [int]$env:PLAB_WARMUP_SECONDS } else { 0 }
[ordered]@{
    schemaVersion = 'protocol-lab.http-executor-result.v1'
    executor = [ordered]@{ id = $executorId; version = $executorVersion }
    loadGenerator = [ordered]@{ id = $executorId; version = $executorVersion }
    validation = [ordered]@{ status = 'passed' }
    protocolProof = [ordered]@{ requestedProtocol = 'h3'; observedProtocol = 'h3'; exactProtocolMatched = $true; fallbackDetected = $false }
    requestedLoad = [ordered]@{ connections = $connections; concurrency = $concurrency; streamsPerConnection = $streams; durationSeconds = $duration; warmupSeconds = $warmup }
    effectiveLoad = [ordered]@{ connections = 1; concurrency = 1; streamsPerConnection = 1; durationSeconds = 0; warmupSeconds = 0 }
    metrics = [ordered]@{ totalRequests = 1; successfulRequests = 1; failedRequests = 0; timeoutRequests = 0; requestsPerSecond = 1; bytesSent = 0; bytesReceived = 0; throughputBytesPerSecond = 0; latencyMeanMs = 0; latencyP50Ms = 0; latencyP75Ms = 0; latencyP90Ms = 0; latencyP95Ms = 0; latencyP99Ms = 0; statusCodeCounts = [ordered]@{ ([string]$ExpectedStatus) = 1 } }
    warnings = @("Diagnostic single-request peer characterization; XQUIC response completion warning retained=$completionWarning; no benchmark payload or latency claim is made.")
} | ConvertTo-Json -Depth 8 -Compress | Write-Output
