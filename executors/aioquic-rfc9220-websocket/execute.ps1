[CmdletBinding()]
param(
    [string]$TargetUrl = "https://host.docker.internal:4435/websocket-proof",
    [double]$TimeoutSeconds = 20,
    [string]$Image = "incursa-protocol-lab-aioquic-rfc9220-websocket:0.1.0",
    [string]$OutputRoot = "artifacts/aioquic-rfc9220-websocket",
    [string]$DockerNetwork = "",
    [switch]$SkipBuild,
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'

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
            commands = $commands.ToArray()
        } | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $resultPath
        Write-Host "Planned aioquic RFC9220 WebSocket executor command at $commandPath"
        return
    }

    if (-not $SkipBuild) {
        & docker @buildArgs
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

    [ordered]@{
        status = if ($exitCode -eq 0 -and $null -ne $clientResult -and $clientResult.status -eq 'passed') { 'passed' } else { 'failed' }
        targetUrl = $TargetUrl
        image = $Image
        exitCode = $exitCode
        clientResultPath = $clientResultPath
        stdoutPath = $stdoutPath
        stderrPath = $stderrPath
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath

    if ($exitCode -ne 0) {
        throw "aioquic RFC9220 WebSocket executor failed with exit code $exitCode. See $stderrPath"
    }

    if ($null -eq $clientResult -or $clientResult.status -ne 'passed') {
        throw "aioquic RFC9220 WebSocket executor did not produce a passed client result at $clientResultPath."
    }

    Write-Host "aioquic RFC9220 WebSocket executor passed for $TargetUrl"
}
finally {
    Pop-Location
}
