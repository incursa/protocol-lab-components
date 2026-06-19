[CmdletBinding()]
param(
    [string]$TargetUrl = "https://host.docker.internal:8443/status",
    [int]$ExpectedStatus = 200,
    [int]$TimeoutSeconds = 15,
    [string]$Image = "ghcr.io/macbre/curl-http3",
    [string]$OutputRoot = "artifacts/curl-http3-client",
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'

$resolvedOutputRoot = Join-Path (Get-Location) $OutputRoot
New-Item -ItemType Directory -Force -Path $resolvedOutputRoot | Out-Null

$bodyPath = Join-Path $resolvedOutputRoot 'body.bin'
$stdoutPath = Join-Path $resolvedOutputRoot 'stdout.txt'
$stderrPath = Join-Path $resolvedOutputRoot 'stderr.txt'
$commandPath = Join-Path $resolvedOutputRoot 'command.txt'
$resultPath = Join-Path $resolvedOutputRoot 'result.json'

$dockerArguments = @(
    'run', '--rm',
    '-v', "${resolvedOutputRoot}:/out",
    $Image,
    '--http3-only',
    '--max-time', ([string]$TimeoutSeconds),
    '--insecure',
    '--silent',
    '--show-error',
    '--output', '/out/body.bin',
    '--write-out', '%{http_code}',
    $TargetUrl
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

Write-Host "curl HTTP/3 executor passed for $TargetUrl"
