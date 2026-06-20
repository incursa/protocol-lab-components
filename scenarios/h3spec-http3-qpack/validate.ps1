$ErrorActionPreference = "Stop"

$packageRoot = $PSScriptRoot
$requiredFiles = @(
    "protocol-lab-package.json",
    "protocol-lab.internal.json",
    "scenarios/http3/core/status.yaml",
    "scenarios/http3/headers/response-headers-50x32.yaml",
    "scenarios/http3/protocol/qpack-repeated-headers.yaml",
    "suites/h3spec-http3-qpack-focused.yaml"
)

foreach ($relativePath in $requiredFiles) {
    $path = Join-Path $packageRoot $relativePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required h3spec scenario package file is missing: $relativePath"
    }
}

Write-Host "h3spec HTTP/3 and QPACK scenario package files are present."
