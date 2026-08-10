[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
)

$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path -LiteralPath $Root).Path
$scenarioRoot = Join-Path $Root 'scenarios'
$validators = @(
    Get-ChildItem -LiteralPath $scenarioRoot -Filter validate.ps1 -Recurse -File |
        Sort-Object FullName
)

if ($validators.Count -eq 0) {
    throw 'No scenario package validators were found.'
}

foreach ($validator in $validators) {
    Write-Host "Validating $($validator.Directory.Name)"
    & $validator.FullName
}

Write-Host "Validated $($validators.Count) scenario package(s)."
