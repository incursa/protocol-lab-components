[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$GraphPath = (Join-Path $Root 'release/component-graph.v1.json'),
    [string]$IntentRoot = (Join-Path $Root 'release/release-intents')
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'ProtocolLabComponentRelease.Common.ps1')
$graph = Get-ProtocolLabComponentReleaseGraph -GraphPath $GraphPath
$componentById = @{}
foreach ($component in @($graph.components)) { $componentById[[string]$component.id] = $component }
$errors = [System.Collections.Generic.List[string]]::new()
$intents = @(Get-ChildItem -LiteralPath $IntentRoot -File -Filter '*.json' | Sort-Object Name)
if ($intents.Count -eq 0) { $errors.Add("No release intent records found under $IntentRoot.") }

foreach ($file in $intents) {
    try { $intent = Get-Content -LiteralPath $file.FullName -Raw | ConvertFrom-Json -AsHashtable }
    catch { $errors.Add("$($file.Name): invalid JSON."); continue }
    if ($intent.schemaVersion -ne 'protocol-lab.release-intent.v1') { $errors.Add("$($file.Name): unsupported schema version '$($intent.schemaVersion)'.") }
    if ([string]::IsNullOrWhiteSpace([string]$intent.id)) { $errors.Add("$($file.Name): missing id.") }
    if ($intent.classification -notin @('release', 'no-release')) { $errors.Add("$($file.Name): classification must be release or no-release.") }
    if ([string]::IsNullOrWhiteSpace([string]$intent.reason)) { $errors.Add("$($file.Name): missing reason.") }
    if ($intent.classification -eq 'no-release' -and @($intent.components).Count -ne 0) { $errors.Add("$($file.Name): no-release intents may not authorize components.") }
    if ($intent.classification -eq 'release') {
        if ($intent.status -ne 'approved') { $errors.Add("$($file.Name): release intent must be approved.") }
        if (@($intent.components).Count -eq 0) { $errors.Add("$($file.Name): release intent has no components.") }
        foreach ($componentRelease in @($intent.components)) {
            $componentId = [string]$componentRelease.componentId
            if (-not $componentById.ContainsKey($componentId)) { $errors.Add("$($file.Name): unknown component '$componentId'."); continue }
            if ([string]$componentRelease.version -notmatch '^\d+\.\d+\.\d+([+-][0-9A-Za-z.-]+)?$') { $errors.Add("$($file.Name): component '$componentId' has invalid version '$($componentRelease.version)'.") }
            if ([string]::IsNullOrWhiteSpace([string]$componentRelease.releaseNotesPath) -or -not (Test-Path -LiteralPath (Join-Path $Root $componentRelease.releaseNotesPath))) { $errors.Add("$($file.Name): component '$componentId' must name an existing release note/changelog path.") }
        }
    }
}

if ($errors.Count -gt 0) { throw ($errors -join [Environment]::NewLine) }
[ordered]@{ schemaVersion = 'protocol-lab.release-intent-validation.v1'; status = 'passed'; intentCount = $intents.Count } | ConvertTo-Json
