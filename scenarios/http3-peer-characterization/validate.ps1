[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path

foreach ($path in @(
    'protocol-lab-package.json',
    'protocol-lab.internal.json',
    'scenarios/http3/external/peer-characterization.yaml',
    'suites/http3-peer-characterization.yaml'
)) {
    $candidate = Join-Path $root $path
    if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) {
        throw "Missing expected package file: $path"
    }
}

Write-Host 'HTTP/3 peer characterization scenario package files are present.'
