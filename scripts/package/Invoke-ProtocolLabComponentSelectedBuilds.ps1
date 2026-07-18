[CmdletBinding()]
param(
    [Parameter(Mandatory)][AllowEmptyCollection()][string[]]$ChangedPath,
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$GraphPath = (Join-Path $Root 'release/component-graph.v1.json'),
    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'
& (Join-Path $PSScriptRoot 'Validate-ProtocolLabComponentManifests.ps1')
& (Join-Path $PSScriptRoot 'Test-ProtocolLabComponentReleaseGraph.ps1') -Root $Root -GraphPath $GraphPath
& (Join-Path $PSScriptRoot 'Test-ProtocolLabReleaseIntents.ps1') -Root $Root -GraphPath $GraphPath
$selection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $GraphPath -ChangedPath $ChangedPath | ConvertFrom-Json

if ($selection.fullBuildDryRunRequired) {
    Write-Warning "Unknown release inputs require legacy full-build dry-run under policy '$((Get-Content -LiteralPath $GraphPath -Raw | ConvertFrom-Json).unmodeledPackagePolicy)': $($selection.unknownPaths -join ', ')"
}
if (-not $SkipBuild) {
    foreach ($component in @($selection.selectedComponents)) {
        $scriptPath = Join-Path $Root $component.script
        Write-Host "Building selected component $($component.componentId) with $($component.script)."
        & $scriptPath @($component.arguments) -Root $Root
        if ($LASTEXITCODE -ne 0) { throw "Selected component build failed for '$($component.componentId)'." }
    }
}
$selection | ConvertTo-Json -Depth 16
