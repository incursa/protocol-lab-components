[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$GraphPath = (Join-Path $Root 'release/component-graph.v1.json')
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'ProtocolLabComponentRelease.Common.ps1')
$graph = Get-ProtocolLabComponentReleaseGraph -GraphPath $GraphPath
$errors = [System.Collections.Generic.List[string]]::new()
$componentById = @{}
$packageIds = @{}

foreach ($shared in @($graph.sharedInputs)) {
    if ([string]::IsNullOrWhiteSpace([string]$shared.id)) { $errors.Add('Shared input has no id.'); continue }
    foreach ($path in @($shared.paths)) {
        if (-not (Test-Path -LiteralPath (Join-Path $Root $path))) { $errors.Add("Shared input '$($shared.id)' path does not exist: $path") }
    }
}

foreach ($component in @($graph.components)) {
    if ([string]::IsNullOrWhiteSpace([string]$component.id) -or $componentById.ContainsKey([string]$component.id)) {
        $errors.Add("Component id '$($component.id)' is missing or duplicated.")
        continue
    }
    $componentById[[string]$component.id] = $component
    if ($packageIds.ContainsKey([string]$component.packageId)) { $errors.Add("Package id '$($component.packageId)' is declared by both '$($componentById[$packageIds[[string]$component.packageId]].id)' and '$($component.id)'.") }
    $packageIds[[string]$component.packageId] = [string]$component.id

    $manifestPath = Join-Path $Root (Join-Path $component.packageRoot 'protocol-lab-package.json')
    if (-not (Test-Path -LiteralPath $manifestPath)) { $errors.Add("Component '$($component.id)' manifest does not exist: $manifestPath"); continue }
    $manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
    if ($manifest.packageId -ne $component.packageId) { $errors.Add("Component '$($component.id)' graph packageId does not match manifest.") }
    if (-not (Test-Path -LiteralPath (Join-Path $Root $component.build.script))) { $errors.Add("Component '$($component.id)' build script does not exist: $($component.build.script)") }
    foreach ($group in @('payload', 'buildRecipe', 'fixtures')) {
        foreach ($path in @($component.inputs[$group])) {
            if (-not (Test-Path -LiteralPath (Join-Path $Root $path))) { $errors.Add("Component '$($component.id)' $group input does not exist: $path") }
        }
    }
}

foreach ($component in @($graph.components)) {
    foreach ($dependency in @($component.dependsOn)) {
        $target = [string]$dependency.componentId
        if (-not $componentById.ContainsKey($target)) { $errors.Add("Component '$($component.id)' depends on unknown component '$target'."); continue }
        if (@($componentById[$target].reverseDependencies) -notcontains $component.id) { $errors.Add("Component '$target' must declare '$($component.id)' as a reverse dependency.") }
    }
    foreach ($reverse in @($component.reverseDependencies)) {
        if (-not $componentById.ContainsKey([string]$reverse)) { $errors.Add("Component '$($component.id)' declares unknown reverse dependency '$reverse'."); continue }
        if (@($componentById[[string]$reverse].dependsOn | ForEach-Object componentId) -notcontains $component.id) { $errors.Add("Reverse dependency '$reverse' must declare a dependency on '$($component.id)'.") }
    }
}

if ($errors.Count -gt 0) { throw ($errors -join [Environment]::NewLine) }
$allManifestCount = @(Get-ChildItem -LiteralPath $Root -Recurse -File -Filter 'protocol-lab-package.json' | Where-Object { $_.FullName -notmatch '[\\/](artifacts|packages)[\\/]' }).Count
[ordered]@{
    schemaVersion = 'protocol-lab.component-graph-validation.v1'
    status = 'passed'
    modeledComponentCount = @($graph.components).Count
    unmodeledManifestCount = $allManifestCount - @($graph.components).Count
    policy = $graph.unmodeledPackagePolicy
} | ConvertTo-Json -Depth 8
