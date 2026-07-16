[CmdletBinding()]
param(
    [string]$TargetUrl = 'https://host.docker.internal:8443/',
    [int]$ExpectedStatus = 200,
    [int]$TimeoutSeconds = 5,
    [string]$Image = 'ghcr.io/alibaba/xquic/xquic-interop@sha256:875df1e9935c6a07e26d7b5ae14df9edd06703061ce35920234a97d6991c58e0',
    [string]$OutputRoot = 'artifacts/xquic-http3-client',
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
