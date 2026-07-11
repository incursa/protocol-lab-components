[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$ComponentPath,
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$BuildConfiguration = 'Release',
    [string]$RuntimeIdentifier = 'portable',
    [string]$OutputRoot = (Join-Path $Root 'artifacts/reproducibility')
)

$ErrorActionPreference = 'Stop'
$builder = Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1'
$first = Join-Path $OutputRoot 'first'
$second = Join-Path $OutputRoot 'second'

foreach ($directory in @($first, $second)) {
    if (Test-Path -LiteralPath $directory) {
        Remove-Item -LiteralPath $directory -Recurse -Force
    }
}

& $builder -Root $Root -ComponentPath $ComponentPath -OutputRoot $first -BuildConfiguration $BuildConfiguration -RuntimeIdentifier $RuntimeIdentifier
& $builder -Root $Root -ComponentPath $ComponentPath -OutputRoot $second -BuildConfiguration $BuildConfiguration -RuntimeIdentifier $RuntimeIdentifier

$firstPackages = @(Get-ChildItem -LiteralPath $first -Filter '*.plabpkg' -File)
$secondPackages = @(Get-ChildItem -LiteralPath $second -Filter '*.plabpkg' -File)
if ($firstPackages.Count -ne 1 -or $secondPackages.Count -ne 1) {
    throw 'Reproducibility check requires exactly one package in each output root.'
}
$firstPackage = $firstPackages[0]
$secondPackage = $secondPackages[0]
$firstHash = (Get-FileHash -LiteralPath $firstPackage.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
$secondHash = (Get-FileHash -LiteralPath $secondPackage.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
if ($firstHash -ne $secondHash) {
    throw "Package rebuild was not deterministic: first $firstHash, second $secondHash."
}

# Rebuilding into the same output root must be an immutable verified no-op.
& $builder -Root $Root -ComponentPath $ComponentPath -OutputRoot $first -BuildConfiguration $BuildConfiguration -RuntimeIdentifier $RuntimeIdentifier
$rebuildHash = (Get-FileHash -LiteralPath $firstPackage.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
if ($rebuildHash -ne $firstHash) {
    throw "Same-root rebuild changed the retained package: first $firstHash, rebuild $rebuildHash."
}

[pscustomobject]@{
    schemaVersion = 'protocol-lab.package-reproducibility-result.v1'
    componentPath = $ComponentPath
    buildConfiguration = $BuildConfiguration
    runtimeIdentifier = $RuntimeIdentifier
    sha256 = $firstHash
    firstPackage = $firstPackage.FullName
    secondPackage = $secondPackage.FullName
    deterministic = $true
    sameRootNoOp = $true
} | ConvertTo-Json -Depth 4
