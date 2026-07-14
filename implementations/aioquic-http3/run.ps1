[CmdletBinding()]
param(
    [ValidateSet('Client', 'Server')]
    [string]$Mode = 'Client',
    [string]$Image = 'incursa-protocol-lab-aioquic-http3:0.3.0',
    [string]$Url = 'https://host.docker.internal:8443/status',
    [int]$ExpectedStatus = 200,
    [string]$OutputPath = 'artifacts/aioquic-http3/body.bin',
    [string]$WwwRoot = 'www',
    [string]$CertPath = 'certs/leaf.pem',
    [string]$KeyPath = 'certs/leaf-key.pem',
    [int]$Port = 4433,
    [string]$DockerNetwork = '',
    [switch]$SkipBuild,
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'

$componentRoot = $PSScriptRoot
$outputFullPath = if ([System.IO.Path]::IsPathRooted($OutputPath)) { $OutputPath } else { Join-Path $componentRoot $OutputPath }
$artifactRoot = Split-Path -Parent $outputFullPath
New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null

$commandPath = Join-Path $artifactRoot 'command.txt'
$stdoutPath = Join-Path $artifactRoot 'stdout.txt'
$stderrPath = Join-Path $artifactRoot 'stderr.txt'
$resultPath = Join-Path $artifactRoot 'result.json'

Push-Location $componentRoot
try {
    $commands = [System.Collections.Generic.List[string]]::new()

    if (-not $SkipBuild) {
        $buildArgs = @('build', '--build-arg', 'AIOQUIC_VERSION=1.3.0', '-f', 'docker/aioquic.Dockerfile', '-t', $Image, '.')
        $commands.Add('docker ' + ($buildArgs -join ' '))
    }

    if ($Mode -eq 'Client') {
        $outputDirectory = Split-Path -Parent $outputFullPath
        $outputFileName = Split-Path -Leaf $outputFullPath
        $dockerArgs = @('run', '--rm')
        if (-not [string]::IsNullOrWhiteSpace($DockerNetwork)) {
            $dockerArgs += @('--network', $DockerNetwork)
        }
        $dockerArgs += @(
            '-v', "${outputDirectory}:/downloads",
            $Image,
            '/usr/local/bin/aioquic-http3-client',
            $Url,
            "/downloads/$outputFileName",
            '--expect-status',
            ([string]$ExpectedStatus)
        )
    }
    else {
        $wwwFullPath = if ([System.IO.Path]::IsPathRooted($WwwRoot)) { $WwwRoot } else { Join-Path $componentRoot $WwwRoot }
        $certFullPath = if ([System.IO.Path]::IsPathRooted($CertPath)) { $CertPath } else { Join-Path $componentRoot $CertPath }
        $keyFullPath = if ([System.IO.Path]::IsPathRooted($KeyPath)) { $KeyPath } else { Join-Path $componentRoot $KeyPath }
        $certDirectory = Split-Path -Parent $certFullPath
        $certFileName = Split-Path -Leaf $certFullPath
        $keyFileName = Split-Path -Leaf $keyFullPath
        $dockerArgs = @('run', '--rm', '-p', "${Port}:4433/udp")
        if (-not [string]::IsNullOrWhiteSpace($DockerNetwork)) {
            $dockerArgs += @('--network', $DockerNetwork)
        }
        $dockerArgs += @(
            '-v', "${wwwFullPath}:/www:ro",
            '-v', "${certDirectory}:/certs:ro",
            $Image,
            '/usr/local/bin/aioquic-http3-server',
            '/www',
            "/certs/$certFileName",
            "/certs/$keyFileName",
            '4433'
        )
    }

    $commands.Add('docker ' + ($dockerArgs -join ' '))
    Set-Content -LiteralPath $commandPath -Value $commands

    if ($PlanOnly) {
        [ordered]@{
            status = 'planned'
            mode = $Mode
            image = $Image
            commands = $commands.ToArray()
        } | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $resultPath
        Write-Host "Planned aioquic HTTP/3 $Mode command at $commandPath"
        return
    }

    if (-not $SkipBuild) {
        & docker @buildArgs
        if ($LASTEXITCODE -ne 0) {
            throw "aioquic HTTP/3 Docker build failed with exit code $LASTEXITCODE."
        }
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
        throw "aioquic HTTP/3 $Mode failed with exit code $exitCode. See $stderrPath"
    }
}
finally {
    Pop-Location
}
