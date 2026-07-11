$ErrorActionPreference = "Stop"

$packageRoot = $PSScriptRoot
$requiredFiles = @(
    "protocol-lab-package.json",
    "protocol-lab.internal.json",
    "scenarios/quic/transport/stream-throughput.yaml",
    "scenarios/quic/transport/latency-echo-1kb.yaml",
    "scenarios/quic/transport/multiplex-100-streams.yaml",
    "scenarios/quic/transport/stream-limits-100-streams.yaml",
    "scenarios/quic/transport/payload-large-1mb.yaml",
    "scenarios/quic/transport/duplex-streams.yaml",
    "scenarios/quic/transport/cancellation-reset-stream.yaml",
    "scenarios/quic/transport/cold-handshake.yaml",
    "scenarios/quic/transport/stream-churn.yaml",
    "scenarios/quic/transport/resumption-resumed.yaml",
    "scenarios/quic/transport/resumption-rejected.yaml",
    "scenarios/quic/transport/0-rtt-accepted.yaml",
    "scenarios/quic/transport/0-rtt-rejected.yaml",
    "suites/raw-quic-transport-v1-smoke.yaml"
)

foreach ($relativePath in $requiredFiles) {
    $path = Join-Path $packageRoot $relativePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required raw QUIC scenario package file is missing: $relativePath"
    }
}

Write-Host "Raw QUIC scenario package files are present."
