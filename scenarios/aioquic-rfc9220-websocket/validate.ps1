$ErrorActionPreference = "Stop"

$packageRoot = $PSScriptRoot
$requiredFiles = @(
    "protocol-lab-package.json",
    "protocol-lab.internal.json",
    "scenarios/http3/websocket/rfc9220-extended-connect.yaml",
    "scenarios/http3/websocket/rfc9220-control-frames.yaml",
    "scenarios/http3/websocket/rfc9220-text-echo.yaml",
    "scenarios/http3/websocket/rfc9220-binary-echo.yaml",
    "scenarios/http3/websocket/rfc9220-close.yaml",
    "suites/aioquic-rfc9220-websocket-proof.yaml"
)

foreach ($relativePath in $requiredFiles) {
    $path = Join-Path $packageRoot $relativePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required aioquic RFC9220 scenario package file is missing: $relativePath"
    }
}

Write-Host "aioquic RFC9220 WebSocket scenario package files are present."
