[CmdletBinding()]
param(
    [string]$TargetBaseUrl = $env:PLAB_TARGET_BASE_URL,
    [string]$OutputDirectory = $env:PLAB_ARTIFACT_DIR
)

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($TargetBaseUrl)) {
    throw "TargetBaseUrl or PLAB_TARGET_BASE_URL is required."
}

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $PSScriptRoot 'artifacts'
}

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null

$sourceRoot = Join-Path $PSScriptRoot 'source'
Push-Location $sourceRoot
try {
    & go run . --target-url $TargetBaseUrl --output-dir $OutputDirectory
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
finally {
    Pop-Location
}
