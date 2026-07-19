[CmdletBinding()]
param(
    [Parameter(Mandatory)][AllowEmptyCollection()][string[]]$ChangedPath,
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$GraphPath = (Join-Path $Root 'release/component-graph.v1.json')
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'ProtocolLabComponentRelease.Common.ps1')
$graph = Get-ProtocolLabComponentReleaseGraph -GraphPath $GraphPath
$components = @($graph.components)
$componentById = @{}
foreach ($component in $components) { $componentById[[string]$component.id] = $component }
$selected = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::Ordinal)
$classification = [System.Collections.Generic.List[string]]::new()
$unknown = [System.Collections.Generic.List[string]]::new()
$templatePaths = @($graph.templates | ForEach-Object paths | ForEach-Object { $_ })

foreach ($inputPath in @($ChangedPath)) {
    $path = ([string]$inputPath).Replace('\', '/').TrimStart('./')
    if ([string]::IsNullOrWhiteSpace($path)) { continue }
    $matched = $false
    if ($path -eq 'release/component-graph.v1.json' -or $path.StartsWith('release/release-intents/', [System.StringComparison]::OrdinalIgnoreCase)) {
        $classification.Add("$path:release-metadata")
        continue
    }
    if ($path -eq 'README.md' -or $path.StartsWith('docs/', [System.StringComparison]::OrdinalIgnoreCase)) {
        $classification.Add("$path:docs-only")
        continue
    }
    foreach ($templatePath in $templatePaths) {
        if (Test-ProtocolLabPathMatchesDeclaration -ChangedPath $path -DeclaredPath ([string]$templatePath)) {
            $classification.Add("$path:template")
            $matched = $true
            break
        }
    }
    if ($matched) { continue }
    foreach ($shared in @($graph.sharedInputs)) {
        foreach ($sharedPath in @($shared.paths)) {
            if (Test-ProtocolLabPathMatchesDeclaration -ChangedPath $path -DeclaredPath ([string]$sharedPath)) {
                foreach ($component in $components) {
                    if (@($component.inputs.shared) -contains $shared.id) { [void]$selected.Add([string]$component.id) }
                }
                $classification.Add("$path:shared:$($shared.id)")
                $matched = $true
                break
            }
        }
        if ($matched) { break }
    }
    if ($matched) { continue }
    foreach ($component in $components) {
        $componentMatched = $false
        foreach ($group in @('payload', 'buildRecipe', 'fixtures')) {
            foreach ($declaredPath in @($component.inputs[$group])) {
                if (Test-ProtocolLabPathMatchesDeclaration -ChangedPath $path -DeclaredPath ([string]$declaredPath)) {
                    [void]$selected.Add([string]$component.id)
                    $classification.Add("${path}:$($component.id):$group")
                    $matched = $true
                    $componentMatched = $true
                    break
                }
            }
            if ($componentMatched) { break }
        }
    }
    if (-not $matched) { $unknown.Add($path) }
}

# Explicit reverse dependencies are followed transitively. No package is added by
# directory ancestry or naming convention.
$queue = [System.Collections.Generic.Queue[string]]::new()
foreach ($id in $selected) { $queue.Enqueue($id) }
while ($queue.Count -gt 0) {
    $id = $queue.Dequeue()
    foreach ($reverse in @($componentById[$id].reverseDependencies)) {
        if ($selected.Add([string]$reverse)) { $queue.Enqueue([string]$reverse) }
    }
}

$ordered = @($selected | Sort-Object)
$builds = @($ordered | ForEach-Object {
    $component = $componentById[$_]
    [ordered]@{ componentId = $component.id; packageId = $component.packageId; script = $component.build.script; arguments = @($component.build.arguments); smoke = $component.smoke }
})
[ordered]@{
    schemaVersion = 'protocol-lab.component-release-selection.v1'
    changedPaths = @($ChangedPath | ForEach-Object { $_.Replace('\', '/') })
    classifications = @($classification | Sort-Object)
    validateAllCheapManifests = $true
    selectedComponents = $builds
    fullBuildDryRunRequired = $unknown.Count -gt 0
    unknownPaths = @($unknown | Sort-Object)
    publication = [ordered]@{ requested = $false; reason = 'Selection is dry-run only. Publication requires a separate approved release intent and registry preflight.' }
} | ConvertTo-Json -Depth 16
