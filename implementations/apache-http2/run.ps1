[CmdletBinding()]
param(
    [string]$Variant = $env:PLAB_PROTOCOL_VARIANT,
    [int]$Port = 0,
    [string]$ContainerName = $env:PLAB_CONTAINER_NAME
)

$ErrorActionPreference = 'Stop'
$image = 'ubuntu/apache2@sha256:6563a8f98ce5469715962cf217335ec73842e56abb3720094a15f2b6747b87bc'
$normalizedVariant = switch ($Variant) {
    { [string]::IsNullOrWhiteSpace($_) } { 'h2c'; break }
    'h2c' { 'h2c'; break }
    'http2-h2c-prior-knowledge' { 'h2c'; break }
    'tls-alpn' { 'tls-alpn'; break }
    'http2-tls-alpn' { 'tls-alpn'; break }
    default { throw "Unsupported Apache HTTP/2 execution variant '$Variant'." }
}
if ($Port -eq 0) {
    $configuredPort = if ($normalizedVariant -eq 'h2c') { $env:PLAB_HTTP_PORT } else { $env:PLAB_HTTPS_PORT }
    $parsedPort = 0
    $Port = if ([int]::TryParse($configuredPort, [ref]$parsedPort)) { $parsedPort } elseif ($normalizedVariant -eq 'h2c') { 8082 } else { 8443 }
}
if ($Port -lt 1 -or $Port -gt 65535) { throw "Port must be between 1 and 65535." }
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw "docker executable was not found on PATH." }

$containerPort = if ($normalizedVariant -eq 'h2c') { 8082 } else { 8443 }
$configName = if ($normalizedVariant -eq 'h2c') { 'apache-http2-h2c.conf' } else { 'apache-http2-tls.conf' }
$moduleCommand = if ($normalizedVariant -eq 'h2c') { 'a2enmod headers http2 >/dev/null && exec apache2-foreground' } else { 'a2enmod headers http2 ssl >/dev/null && exec apache2-foreground' }
$runRoot = Join-Path ([System.IO.Path]::GetTempPath()) "protocol-lab-apache-http2-$PID"
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
        '--publish', "127.0.0.1:${Port}:${containerPort}",
        '--mount', "type=bind,source=$fixtureRoot,target=/var/www/protocol-lab,readonly",
        '--mount', "type=bind,source=$(Join-Path $PSScriptRoot $configName),target=/etc/apache2/conf-enabled/zzz-protocol-lab-http2.conf,readonly"
    )
    if ($normalizedVariant -eq 'tls-alpn') {
        $arguments += @('--mount', "type=bind,source=$(Join-Path $PSScriptRoot 'certs'),target=/run/protocol-lab-certs,readonly")
    }
    if (-not [string]::IsNullOrWhiteSpace($ContainerName)) { $arguments += @('--name', $ContainerName) }
    $arguments += @('--entrypoint', '/bin/sh', $image, '-ec', $moduleCommand)
    & docker @arguments
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Remove-Item -LiteralPath $runRoot -Recurse -Force -ErrorAction SilentlyContinue
}
