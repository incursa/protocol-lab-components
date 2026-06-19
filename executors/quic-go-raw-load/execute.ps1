[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]] $Arguments
)

$ErrorActionPreference = "Stop"

$packageRoot = $PSScriptRoot
$binary = Join-Path $packageRoot "bin/win-x64/quic-go-raw-load.exe"

if (Test-Path -LiteralPath $binary -PathType Leaf) {
    & $binary @Arguments
    exit $LASTEXITCODE
}

$sourceRoot = Join-Path $packageRoot "source"
if (-not (Test-Path -LiteralPath (Join-Path $sourceRoot "go.mod") -PathType Leaf)) {
    throw "quic-go raw load binary and source payload were not found."
}

& go -C $sourceRoot run ./cmd/quic-go-raw-load @Arguments
exit $LASTEXITCODE
