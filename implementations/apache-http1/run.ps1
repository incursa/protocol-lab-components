[CmdletBinding()]
param(
    [int]$Port = 0,
    [string]$ContainerName = $env:PLAB_CONTAINER_NAME
)

$ErrorActionPreference = 'Stop'
$image = 'ubuntu/apache2@sha256:6563a8f98ce5469715962cf217335ec73842e56abb3720094a15f2b6747b87bc'
if ($Port -eq 0) {
    $configuredPort = $env:PLAB_HTTP_PORT
    $parsedPort = 0
    $Port = if ([int]::TryParse($configuredPort, [ref]$parsedPort)) { $parsedPort } else { 8080 }
}
if ($Port -lt 1 -or $Port -gt 65535) { throw "Port must be between 1 and 65535." }
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw "docker executable was not found on PATH." }
if ([string]::IsNullOrWhiteSpace($ContainerName)) { $ContainerName = "protocol-lab-apache-http1-$PID" }

$runRoot = Join-Path ([System.IO.Path]::GetTempPath()) "protocol-lab-apache-http1-$PID"
$fixtureRoot = Join-Path $runRoot 'fixtures'
New-Item -ItemType Directory -Force -Path $fixtureRoot | Out-Null
try {
    foreach ($encoded in Get-ChildItem -LiteralPath (Join-Path $PSScriptRoot 'fixtures') -Filter '*.b64' -File) {
        $destination = Join-Path $fixtureRoot $encoded.BaseName
        $bytes = [Convert]::FromBase64String((Get-Content -LiteralPath $encoded.FullName -Raw).Trim())
        [System.IO.File]::WriteAllBytes($destination, $bytes)
    }

    $arguments = @(
        'run', '--rm', '--init', '--pull', 'missing',
        '--publish', "127.0.0.1:${Port}:8080",
        '--mount', "type=bind,source=$fixtureRoot,target=/var/www/protocol-lab,readonly",
        '--mount', "type=bind,source=$(Join-Path $PSScriptRoot 'apache-http1.conf'),target=/etc/apache2/conf-enabled/zzz-protocol-lab-http1.conf,readonly"
    )
    $arguments += @('--name', $ContainerName)
    $arguments += @('--entrypoint', '/bin/sh', $image, '-ec', 'a2enmod headers >/dev/null && exec apache2-foreground')
    & docker @arguments
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    & docker rm --force $ContainerName 2>$null | Out-Null
    Remove-Item -LiteralPath $runRoot -Recurse -Force -ErrorAction SilentlyContinue
}
