$ErrorActionPreference = "Stop"

$packageRoot = $PSScriptRoot
$requiredFiles = @(
    "protocol-lab-package.json",
    "protocol-lab.internal.json",
    "authority-lock.json",
    "scenarios/http3/websocket/rfc9220-extended-connect.yaml",
    "scenarios/http3/websocket/rfc9220-control-frames.yaml",
    "scenarios/http3/websocket/rfc9220-text-echo.yaml",
    "scenarios/http3/websocket/rfc9220-binary-echo.yaml",
    "scenarios/http3/websocket/rfc9220-close.yaml",
    "scenarios/http3/websocket/rfc9220-fragmented-binary-echo.yaml",
    "suites/aioquic-rfc9220-websocket-proof.yaml"
)

foreach ($relativePath in $requiredFiles) {
    $path = Join-Path $packageRoot $relativePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required aioquic RFC9220 scenario package file is missing: $relativePath"
    }
}

$authority = Get-Content (Join-Path $packageRoot 'authority-lock.json') -Raw | ConvertFrom-Json
if ($authority.authorityCommit -ne '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574') { throw 'RFC9220 authority commit mismatch.' }
$fragmented = Join-Path $packageRoot 'scenarios/http3/websocket/rfc9220-fragmented-binary-echo.yaml'
$hash = (Get-FileHash $fragmented -Algorithm SHA256).Hash.ToLowerInvariant()
if ($hash -ne '76bb1c269d42b5ba53742bf5c69e8f2728427406946a7cf2802023f482959725') { throw 'RFC9220 fragmented scenario authority hash mismatch.' }

Write-Host "aioquic RFC9220 WebSocket scenario package authority lock is valid."
