[CmdletBinding()]
param(
    [string]$Image = 'incursa-protocol-lab-nginx-http3:0.1.4',
    [int]$Port = 5446,
    [switch]$SkipBuild,
    [switch]$PlanOnly,
    [switch]$ProofOnly,
    [string]$OutputRoot = 'artifacts/nginx-http3'
)

$ErrorActionPreference = 'Stop'

$componentRoot = $PSScriptRoot
$artifactRoot = if ([System.IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $componentRoot $OutputRoot }
New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null

$commandPath = Join-Path $artifactRoot 'command.txt'
$versionPath = Join-Path $artifactRoot 'nginx-version.txt'
$resultPath = Join-Path $artifactRoot 'result.json'
$stdoutPath = Join-Path $artifactRoot 'stdout.txt'
$stderrPath = Join-Path $artifactRoot 'stderr.txt'

Push-Location $componentRoot
try {
    $commands = [System.Collections.Generic.List[string]]::new()

    if (-not $SkipBuild) {
        $buildArgs = @('build', '--pull', '-f', 'docker/nginx.Dockerfile', '-t', $Image, 'docker')
        $commands.Add('docker ' + ($buildArgs -join ' '))
    }

    $proofArgs = @('run', '--rm', '--entrypoint', 'nginx', $Image, '-V')
    $commands.Add('docker ' + ($proofArgs -join ' '))

    $configArgs = @('run', '--rm', $Image, 'nginx', '-t')
    $commands.Add('docker ' + ($configArgs -join ' '))

    $dockerArgs = @(
        'run',
        '--rm',
        '-p',
        "${Port}:8443/tcp",
        '-p',
        "${Port}:8443/udp",
        '-e',
        "PLAB_HTTP_PORT=$Port",
        $Image,
        'nginx',
        '-g',
        'daemon off;'
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
        Write-Host "Planned nginx HTTP/3 command at $commandPath"
        return
    }

    if (-not $SkipBuild) {
        & docker @buildArgs
        if ($LASTEXITCODE -ne 0) {
            throw "nginx HTTP/3 Docker build failed with exit code $LASTEXITCODE."
        }
    }

    $versionStdout = [System.IO.Path]::GetTempFileName()
    $versionStderr = [System.IO.Path]::GetTempFileName()
    try {
        $versionProcess = Start-Process -FilePath 'docker' -ArgumentList $proofArgs -NoNewWindow -Wait -PassThru -RedirectStandardOutput $versionStdout -RedirectStandardError $versionStderr
        $versionText = ((Get-Content -LiteralPath $versionStdout -Raw) + (Get-Content -LiteralPath $versionStderr -Raw))
    }
    finally {
        Remove-Item -LiteralPath $versionStdout, $versionStderr -Force -ErrorAction SilentlyContinue
    }

    Set-Content -LiteralPath $versionPath -Value $versionText
    if ($versionProcess.ExitCode -ne 0) {
        throw "nginx -V proof failed with exit code $($versionProcess.ExitCode)."
    }

    if ($versionText -notmatch '--with[-_]http_v3_module') {
        throw "Selected nginx image '$Image' does not advertise HTTP/3 support. Expected nginx -V output to include --with-http_v3_module or --with_http_v3_module."
    }

    & docker @configArgs
    if ($LASTEXITCODE -ne 0) {
        throw "nginx HTTP/3 Docker configuration validation failed with exit code $LASTEXITCODE."
    }

    if ($ProofOnly) {
        [ordered]@{
            status = 'proved'
            image = $Image
            nginxVersionPath = $versionPath
            requiredModule = '--with-http_v3_module'
        } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath
        Write-Host "Proved nginx HTTP/3 module support at $versionPath"
        return
    }

    & docker @dockerArgs > $stdoutPath 2> $stderrPath
    $exitCode = $LASTEXITCODE
    [ordered]@{
        status = if ($exitCode -eq 0) { 'stopped' } else { 'failed' }
        image = $Image
        port = $Port
        exitCode = $exitCode
        nginxVersionPath = $versionPath
        stdoutPath = $stdoutPath
        stderrPath = $stderrPath
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath

    if ($exitCode -ne 0) {
        throw "nginx HTTP/3 server failed with exit code $exitCode. See $stderrPath"
    }
}
finally {
    Pop-Location
}
