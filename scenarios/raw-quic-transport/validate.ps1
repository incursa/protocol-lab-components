$ErrorActionPreference = "Stop"

$packageRoot = $PSScriptRoot
$requiredFiles = @(
    "protocol-lab-package.json",
    "protocol-lab.internal.json",
    "scenarios/quic/transport/stream-throughput.yaml",
    "scenarios/quic/transport/stream-download-1mb.yaml",
    "scenarios/quic/transport/stream-throughput-64kb.yaml",
    "scenarios/quic/transport/stream-throughput-16mb.yaml",
    "scenarios/quic/transport/latency-echo-1kb.yaml",
    "scenarios/quic/transport/multiplex-100-streams.yaml",
    "scenarios/quic/transport/multiplex-100x1kb.yaml",
    "scenarios/quic/transport/multiplex-16x1mb.yaml",
    "scenarios/quic/transport/stream-limits-100-streams.yaml",
    "scenarios/quic/transport/payload-large-1mb.yaml",
    "scenarios/quic/transport/duplex-streams.yaml",
    "scenarios/quic/transport/duplex-streams-16x1mb.yaml",
    "scenarios/quic/transport/duplex-streams-peer-matrix.yaml",
    "scenarios/quic/transport/cancellation-reset-stream.yaml",
    "scenarios/quic/transport/handshake-cold.yaml",
    "scenarios/quic/transport/connection-churn.yaml",
    "scenarios/quic/transport/stream-churn.yaml",
    "scenarios/quic/transport/resumption-rejected.yaml",
    "scenarios/quic/transport/resumed-handshake.yaml",
    "scenarios/quic/transport/zero-rtt-accepted.yaml",
    "scenarios/quic/transport/zero-rtt-rejected.yaml",
    "suites/raw-quic-transport-v1-smoke.yaml",
    "suites/quic-transport-v1-comparison.yaml",
    "specifications/documents/rfc9000.json",
    "specifications/requirements/rfc9000/REQ-QUIC-RFC9000-0271.json",
    "specifications/requirements/rfc9000/REQ-QUIC-RFC9000-0897.json",
    "specifications/catalogs/quic-rfc9000-handshake-pilot.json",
    "specifications/scenario-mappings/quic.transport.handshake-cold.json",
    "specifications/coverage-profiles/quic-handshake-bootstrap-pilot.json"
)

foreach ($relativePath in $requiredFiles) {
    $path = Join-Path $packageRoot $relativePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required raw QUIC scenario package file is missing: $relativePath"
    }
}

Write-Host "Raw QUIC scenario package files are present."
