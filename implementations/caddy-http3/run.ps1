[CmdletBinding()]
param(
    [string]$Image = 'incursa-protocol-lab-caddy-http3:0.1.2',
    [int]$Port = 5445,
    [switch]$SkipBuild,
    [switch]$PlanOnly,
    [string]$OutputRoot = 'artifacts/caddy-http3'
)

$ErrorActionPreference = 'Stop'

$componentRoot = $PSScriptRoot
$artifactRoot = if ([System.IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $componentRoot $OutputRoot }
New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null

$commandPath = Join-Path $artifactRoot 'command.txt'
$resultPath = Join-Path $artifactRoot 'result.json'
$stdoutPath = Join-Path $artifactRoot 'stdout.txt'
$stderrPath = Join-Path $artifactRoot 'stderr.txt'

Push-Location $componentRoot
try {
    $commands = [System.Collections.Generic.List[string]]::new()

    if (-not $SkipBuild) {
        $buildArgs = @('build', '--pull', '-f', 'docker/Caddy.Dockerfile', '-t', $Image, 'docker')
        $commands.Add('docker ' + ($buildArgs -join ' '))
    }

    $dockerArgs = @(
        'run',
        '--rm',
        '-p',
        "${Port}:8443/tcp",
        '-p',
        "${Port}:8443/udp",
        $Image,
        'caddy',
        'run',
        '--config',
        '/etc/caddy/Caddyfile',
        '--adapter',
        'caddyfile'
    )
    $commands.Add('docker ' + ($dockerArgs -join ' '))
    Set-Content -LiteralPath $commandPath -Value $commands

    if ($PlanOnly) {
        [ordered]@{
            status = 'planned'
            image = $Image
            port = $Port
            commands = $commands.ToArray()
        } | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $resultPath
        Write-Host "Planned Caddy HTTP/3 command at $commandPath"
        return
    }

    if (-not $SkipBuild) {
        & docker @buildArgs
        if ($LASTEXITCODE -ne 0) {
            throw "Caddy HTTP/3 Docker build failed with exit code $LASTEXITCODE."
        }
    }

    & docker @dockerArgs > $stdoutPath 2> $stderrPath
    $exitCode = $LASTEXITCODE
    [ordered]@{
        status = if ($exitCode -eq 0) { 'stopped' } else { 'failed' }
        image = $Image
        port = $Port
        exitCode = $exitCode
        stdoutPath = $stdoutPath
        stderrPath = $stderrPath
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath

    if ($exitCode -ne 0) {
        throw "Caddy HTTP/3 server failed with exit code $exitCode. See $stderrPath"
    }
}
finally {
    Pop-Location
}
