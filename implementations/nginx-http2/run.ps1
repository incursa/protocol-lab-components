[CmdletBinding()]
param(
    [string]$Image = 'incursa-protocol-lab-nginx-http2:0.1.1',
    [int]$Port = 8084,
    [switch]$SkipBuild,
    [switch]$PlanOnly,
    [switch]$ProofOnly,
    [string]$OutputRoot = 'artifacts/nginx-http2'
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
    $configArgs = @('run', '--rm', $Image, 'nginx', '-t')
    $dockerArgs = @('run', '--rm', '-p', "${Port}:8080/tcp", $Image)
    $commands.Add('docker ' + ($proofArgs -join ' '))
    $commands.Add('docker ' + ($configArgs -join ' '))
    $commands.Add('docker ' + ($dockerArgs -join ' '))
    Set-Content -LiteralPath $commandPath -Value $commands

    if ($PlanOnly) {
        [ordered]@{ status = 'planned'; image = $Image; port = $Port; protocolVariant = 'h2c-prior-knowledge'; commands = $commands.ToArray() } |
            ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $resultPath -Encoding utf8NoBOM
        Write-Host "Planned nginx HTTP/2 commands at $commandPath"
        return
    }

    if (-not $SkipBuild) {
        & docker @buildArgs
        if ($LASTEXITCODE -ne 0) { throw "nginx HTTP/2 Docker build failed with exit code $LASTEXITCODE." }
    }

    $versionText = (& docker @proofArgs 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { throw "nginx HTTP/2 module proof failed with exit code $LASTEXITCODE." }
    Set-Content -LiteralPath $versionPath -Value $versionText -Encoding utf8NoBOM
    if ($versionText -notmatch '--with-http_v2_module') { throw "Selected image '$Image' does not advertise --with-http_v2_module." }

    & docker @configArgs
    if ($LASTEXITCODE -ne 0) { throw "nginx HTTP/2 configuration validation failed with exit code $LASTEXITCODE." }

    if ($ProofOnly) {
        [ordered]@{ status = 'proved'; image = $Image; requiredModule = '--with-http_v2_module'; protocolVariant = 'h2c-prior-knowledge' } |
            ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath -Encoding utf8NoBOM
        Write-Host "Proved nginx HTTP/2 package inputs at $versionPath"
        return
    }

    & docker @dockerArgs > $stdoutPath 2> $stderrPath
    $exitCode = $LASTEXITCODE
    [ordered]@{ status = if ($exitCode -eq 0) { 'stopped' } else { 'failed' }; image = $Image; port = $Port; protocolVariant = 'h2c-prior-knowledge'; exitCode = $exitCode; nginxVersionPath = $versionPath; stdoutPath = $stdoutPath; stderrPath = $stderrPath } |
        ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath -Encoding utf8NoBOM
    if ($exitCode -ne 0) { throw "nginx HTTP/2 server failed with exit code $exitCode. See $stderrPath" }
}
finally {
    Pop-Location
}
