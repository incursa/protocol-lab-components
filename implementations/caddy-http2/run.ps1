[CmdletBinding()]
param(
    [string]$Image = 'incursa-protocol-lab-caddy-http2:0.1.0',
    [int]$Port = 8083,
    [switch]$SkipBuild,
    [switch]$PlanOnly,
    [switch]$ProofOnly,
    [string]$OutputRoot = 'artifacts/caddy-http2'
)

$ErrorActionPreference = 'Stop'
$componentRoot = $PSScriptRoot
$artifactRoot = if ([System.IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $componentRoot $OutputRoot }
New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null

$commandPath = Join-Path $artifactRoot 'command.txt'
$versionPath = Join-Path $artifactRoot 'caddy-version.txt'
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

    $proofArgs = @('run', '--rm', $Image, 'caddy', 'version')
    $configArgs = @('run', '--rm', $Image, 'caddy', 'validate', '--config', '/etc/caddy/Caddyfile', '--adapter', 'caddyfile')
    $dockerArgs = @('run', '--rm', '-p', "${Port}:8080/tcp", $Image)
    $commands.Add('docker ' + ($proofArgs -join ' '))
    $commands.Add('docker ' + ($configArgs -join ' '))
    $commands.Add('docker ' + ($dockerArgs -join ' '))
    Set-Content -LiteralPath $commandPath -Value $commands

    if ($PlanOnly) {
        [ordered]@{ status = 'planned'; image = $Image; port = $Port; protocolVariant = 'h2c-prior-knowledge'; commands = $commands.ToArray() } |
            ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $resultPath -Encoding utf8NoBOM
        Write-Host "Planned Caddy HTTP/2 commands at $commandPath"
        return
    }

    if (-not $SkipBuild) {
        & docker @buildArgs
        if ($LASTEXITCODE -ne 0) { throw "Caddy HTTP/2 Docker build failed with exit code $LASTEXITCODE." }
    }

    $versionText = (& docker @proofArgs 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { throw "Caddy HTTP/2 version proof failed with exit code $LASTEXITCODE." }
    Set-Content -LiteralPath $versionPath -Value $versionText -Encoding utf8NoBOM
    if ($versionText -notmatch '^v2\.11\.2(?:\s|$)') { throw "Selected image '$Image' did not prove Caddy v2.11.2: $versionText" }

    & docker @configArgs
    if ($LASTEXITCODE -ne 0) { throw "Caddy HTTP/2 configuration validation failed with exit code $LASTEXITCODE." }

    if ($ProofOnly) {
        [ordered]@{ status = 'proved'; image = $Image; caddyVersion = $versionText; protocolVariant = 'h2c-prior-knowledge' } |
            ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath -Encoding utf8NoBOM
        Write-Host "Proved Caddy HTTP/2 package inputs at $versionPath"
        return
    }

    & docker @dockerArgs > $stdoutPath 2> $stderrPath
    $exitCode = $LASTEXITCODE
    [ordered]@{ status = if ($exitCode -eq 0) { 'stopped' } else { 'failed' }; image = $Image; port = $Port; protocolVariant = 'h2c-prior-knowledge'; exitCode = $exitCode; stdoutPath = $stdoutPath; stderrPath = $stderrPath } |
        ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath -Encoding utf8NoBOM
    if ($exitCode -ne 0) { throw "Caddy HTTP/2 server failed with exit code $exitCode. See $stderrPath" }
}
finally {
    Pop-Location
}
