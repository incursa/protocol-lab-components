[CmdletBinding()]
param(
    [string]$TargetUrl = "https://host.docker.internal:4435/websocket-proof",
    [double]$TimeoutSeconds = 20,
    [string]$Image = "incursa-protocol-lab-aioquic-rfc9220-websocket:0.1.7",
    [string]$OutputRoot = "artifacts/aioquic-rfc9220-websocket",
    [string]$DockerNetwork = "",
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$RemainingArguments = @(),
    [switch]$SkipBuild,
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'

function Resolve-ProofTargetUrl {
    param([Parameter(Mandatory)][string]$Url)

    $builder = [UriBuilder]$Url
    if ([string]::IsNullOrWhiteSpace($builder.Path) -or $builder.Path -eq '/') {
        $builder.Path = '/websocket-proof'
    }

    if ($builder.Host -in @('localhost', '127.0.0.1', '::1')) {
        $builder.Host = 'host.docker.internal'
    }

    return $builder.Uri.AbsoluteUri
}

foreach ($argument in $RemainingArguments) {
    if ($argument -match '^https?://') {
        $TargetUrl = $argument
    }
    elseif (-not [string]::IsNullOrWhiteSpace($argument)) {
        throw "Unknown argument: $argument"
    }
}

$TargetUrl = Resolve-ProofTargetUrl -Url $TargetUrl

$componentRoot = $PSScriptRoot
$resolvedOutputRoot = if ([System.IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $componentRoot $OutputRoot }
$qlogRoot = Join-Path $resolvedOutputRoot 'qlog'
$sslKeyLogRoot = Join-Path $resolvedOutputRoot 'sslkeylog'
New-Item -ItemType Directory -Force -Path $resolvedOutputRoot, $qlogRoot, $sslKeyLogRoot | Out-Null

$clientResultPath = Join-Path $resolvedOutputRoot 'client-result.json'
$stdoutPath = Join-Path $resolvedOutputRoot 'stdout.txt'
$stderrPath = Join-Path $resolvedOutputRoot 'stderr.txt'
$commandPath = Join-Path $resolvedOutputRoot 'command.txt'
$resultPath = Join-Path $resolvedOutputRoot 'result.json'
$buildStdoutPath = Join-Path $resolvedOutputRoot 'build.stdout.txt'
$buildStderrPath = Join-Path $resolvedOutputRoot 'build.stderr.txt'

Push-Location $componentRoot
try {
    $commands = [System.Collections.Generic.List[string]]::new()
    if (-not $SkipBuild) {
        $buildArgs = @(
            'build',
            '--build-arg',
            'AIOQUIC_VERSION=1.3.0',
            '-f',
            'docker/aioquic-rfc9220-websocket.Dockerfile',
            '-t',
            $Image,
            '.'
        )
        $commands.Add('docker ' + ($buildArgs -join ' '))
    }

    $dockerArgs = @(
        'run',
        '--rm',
        '--add-host=host.docker.internal:host-gateway'
    )
    if (-not [string]::IsNullOrWhiteSpace($DockerNetwork)) {
        $dockerArgs += @('--network', $DockerNetwork)
    }

    $dockerArgs += @(
        '-v',
        "${resolvedOutputRoot}:/proof",
        '-e',
        'QLOGDIR=/proof/qlog',
        '-e',
        'SSLKEYLOGFILE=/proof/sslkeylog/keys.log',
        $Image,
        '/usr/local/bin/aioquic-http3-websocket-client',
        $TargetUrl,
        '/proof/client-result.json',
        '--timeout',
        ([string]$TimeoutSeconds)
    )
    $commands.Add('docker ' + ($dockerArgs -join ' '))
    Set-Content -LiteralPath $commandPath -Value $commands

    if ($PlanOnly) {
        [ordered]@{
            status = 'planned'
            targetUrl = $TargetUrl
            image = $Image
            tool = 'aioquic-rfc9220-websocket'
            metrics = [ordered]@{
                totalRequests = 0
                successfulRequests = 0
                failedRequests = 0
            }
            commands = $commands.ToArray()
        } | ConvertTo-Json -Depth 5 | Tee-Object -FilePath $resultPath
        [Console]::Error.WriteLine("Planned aioquic RFC9220 WebSocket executor command at $commandPath")
        return
    }

    if (-not $SkipBuild) {
        & docker @buildArgs > $buildStdoutPath 2> $buildStderrPath
        if ($LASTEXITCODE -ne 0) {
            throw "aioquic RFC9220 WebSocket Docker build failed with exit code $LASTEXITCODE."
        }
    }

    & docker @dockerArgs > $stdoutPath 2> $stderrPath
    $exitCode = $LASTEXITCODE
    $clientResult = $null
    if (Test-Path -LiteralPath $clientResultPath) {
        $clientResult = Get-Content -LiteralPath $clientResultPath -Raw | ConvertFrom-Json
    }

    $result = [ordered]@{
        status = if ($exitCode -eq 0 -and $null -ne $clientResult -and $clientResult.status -eq 'passed') { 'passed' } else { 'failed' }
        targetUrl = $TargetUrl
        image = $Image
        tool = 'aioquic-rfc9220-websocket'
        exitCode = $exitCode
        evidenceClass = if ($null -ne $clientResult) { $clientResult.evidenceClass } else { $null }
        statusCode = if ($null -ne $clientResult) { $clientResult.statusCode } else { $null }
        proofScope = if ($null -ne $clientResult) { @($clientResult.proofScope) } else { @() }
        metrics = [ordered]@{
            totalRequests = 1
            successfulRequests = if ($exitCode -eq 0 -and $null -ne $clientResult -and $clientResult.status -eq 'passed') { 1 } else { 0 }
            failedRequests = if ($exitCode -eq 0 -and $null -ne $clientResult -and $clientResult.status -eq 'passed') { 0 } else { 1 }
        }
        clientResultPath = $clientResultPath
        stdoutPath = $stdoutPath
        stderrPath = $stderrPath
    }
    $result | ConvertTo-Json -Depth 6 | Tee-Object -FilePath $resultPath

    if ($exitCode -ne 0) {
        throw "aioquic RFC9220 WebSocket executor failed with exit code $exitCode. See $stderrPath"
    }

    if ($null -eq $clientResult -or $clientResult.status -ne 'passed') {
        throw "aioquic RFC9220 WebSocket executor did not produce a passed client result at $clientResultPath."
    }

    [Console]::Error.WriteLine("aioquic RFC9220 WebSocket executor passed for $TargetUrl")
}
finally {
    Pop-Location
}
