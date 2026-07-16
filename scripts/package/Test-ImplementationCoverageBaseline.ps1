[CmdletBinding()]
param(
    [string]$RepositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$SiteRepositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot '../../../protocol-lab-site')).Path
)

$ErrorActionPreference = 'Stop'
$errors = [System.Collections.Generic.List[string]]::new()

function Add-ValidationError {
    param([string]$Message)
    $errors.Add($Message)
}

function Assert-ExactSet {
    param(
        [string]$Label,
        [object[]]$Expected,
        [object[]]$Actual
    )

    $expectedValues = @($Expected | ForEach-Object { [string]$_ } | Sort-Object -Unique)
    $actualValues = @($Actual | ForEach-Object { [string]$_ } | Sort-Object -Unique)
    $missing = @($expectedValues | Where-Object { $_ -notin $actualValues })
    $unexpected = @($actualValues | Where-Object { $_ -notin $expectedValues })
    if ($missing.Count -gt 0 -or $unexpected.Count -gt 0) {
        Add-ValidationError "$Label differs. Missing=[$($missing -join ', ')]; Unexpected=[$($unexpected -join ', ')]."
    }
}

function Read-FrontmatterValue {
    param(
        [string]$Content,
        [string]$Name
    )

    $match = [regex]::Match($Content, "(?m)^$([regex]::Escape($Name)):\s*(.+?)\s*$")
    if (-not $match.Success) {
        return $null
    }

    return $match.Groups[1].Value
}

$baselinePath = Join-Path $RepositoryRoot 'docs/implementation-coverage-baseline.json'
$inventoryPath = Join-Path $RepositoryRoot 'docs/quic-http3-implementation-inventory.json'
$siteCatalogPath = Join-Path $SiteRepositoryRoot 'src/content/implementations'

$baseline = Get-Content -Raw -LiteralPath $baselinePath | ConvertFrom-Json
$inventory = Get-Content -Raw -LiteralPath $inventoryPath | ConvertFrom-Json

if ($baseline.schemaVersion -ne 'protocol-lab-implementation-coverage-baseline-v1') {
    Add-ValidationError "Unexpected baseline schemaVersion '$($baseline.schemaVersion)'."
}

$manifestFiles = @(Get-ChildItem -LiteralPath (Join-Path $RepositoryRoot 'implementations') -Recurse -Filter 'protocol-lab-package.json' -File | Sort-Object FullName)
$baselinePackagesByPath = @{}
foreach ($entry in $baseline.localImplementationPackages) {
    if ($baselinePackagesByPath.ContainsKey($entry.path)) {
        Add-ValidationError "Duplicate local package path '$($entry.path)'."
        continue
    }
    $baselinePackagesByPath[$entry.path] = $entry
}

$actualManifestPaths = @($manifestFiles | ForEach-Object {
    $_.FullName.Substring($RepositoryRoot.Length + 1).Replace('\', '/')
})
Assert-ExactSet -Label 'Local implementation manifest paths' -Expected @($baselinePackagesByPath.Keys) -Actual $actualManifestPaths

foreach ($manifestFile in $manifestFiles) {
    $relativePath = $manifestFile.FullName.Substring($RepositoryRoot.Length + 1).Replace('\', '/')
    if (-not $baselinePackagesByPath.ContainsKey($relativePath)) {
        continue
    }

    $entry = $baselinePackagesByPath[$relativePath]
    $manifest = Get-Content -Raw -LiteralPath $manifestFile.FullName | ConvertFrom-Json
    if ($entry.packageId -ne $manifest.packageId) {
        Add-ValidationError "$relativePath packageId is '$($manifest.packageId)', baseline has '$($entry.packageId)'."
    }
    if ($entry.version -ne $manifest.packageVersion) {
        Add-ValidationError "$relativePath version is '$($manifest.packageVersion)', baseline has '$($entry.version)'."
    }

    $implementationIds = @($manifest.providedImplementations | ForEach-Object { $_.implementationId })
    $protocols = @($manifest.providedImplementations | ForEach-Object { $_.protocols } | Sort-Object -Unique)
    Assert-ExactSet -Label "$relativePath implementation IDs" -Expected @($entry.implementationIds) -Actual $implementationIds
    Assert-ExactSet -Label "$relativePath protocols" -Expected @($entry.protocols) -Actual $protocols
}

$implementationDirectories = @(Get-ChildItem -LiteralPath (Join-Path $RepositoryRoot 'implementations') -Directory)
$actualDecisionDirectories = @($implementationDirectories | Where-Object {
    -not (Test-Path -LiteralPath (Join-Path $_.FullName 'protocol-lab-package.json'))
} | ForEach-Object {
    "implementations/$($_.Name)"
} | Sort-Object)
$baselineDecisionDirectories = @($baseline.localDecisionDirectories | ForEach-Object { $_.path })
Assert-ExactSet -Label 'Local decision directories' -Expected $baselineDecisionDirectories -Actual $actualDecisionDirectories

foreach ($decision in $baseline.localDecisionDirectories) {
    if (-not (Test-Path -LiteralPath (Join-Path $RepositoryRoot $decision.decisionFile))) {
        Add-ValidationError "Decision file '$($decision.decisionFile)' does not exist."
    }
    if ($decision.evidenceStatus -ne 'closed-by-decision' -or [string]::IsNullOrWhiteSpace($decision.blocker)) {
        Add-ValidationError "Decision directory '$($decision.path)' must retain closed-by-decision status and a blocker."
    }
}

$inventoryIds = @($inventory.implementations | ForEach-Object { $_.id })
$baselineInventoryIds = @($baseline.quicHttp3InventoryMappings | ForEach-Object { $_.inventoryId })
Assert-ExactSet -Label 'QUIC/HTTP3 inventory IDs' -Expected $baselineInventoryIds -Actual $inventoryIds

if ([int]$inventory.summary.implementationCount -ne $inventory.implementations.Count) {
    Add-ValidationError "Inventory summary implementationCount is $($inventory.summary.implementationCount), actual is $($inventory.implementations.Count)."
}
$packageableCount = @($inventory.implementations | Where-Object { $_.packageFeasibility -ne 'planned' -and $_.packageFeasibility -ne 'blocked' }).Count
$designFirstCount = @($inventory.implementations | Where-Object { $_.packageFeasibility -eq 'planned' -or $_.packageFeasibility -eq 'blocked' }).Count
if ([int]$inventory.summary.packageableNowCount -ne $packageableCount) {
    Add-ValidationError "Inventory summary packageableNowCount is $($inventory.summary.packageableNowCount), derived is $packageableCount."
}
if ([int]$inventory.summary.blockedOrDesignFirstCount -ne $designFirstCount) {
    Add-ValidationError "Inventory summary blockedOrDesignFirstCount is $($inventory.summary.blockedOrDesignFirstCount), derived is $designFirstCount."
}

$inventoryById = @{}
foreach ($item in $inventory.implementations) { $inventoryById[$item.id] = $item }
$packageIds = @($baseline.localImplementationPackages | ForEach-Object { $_.packageId })
foreach ($mapping in $baseline.quicHttp3InventoryMappings) {
    if ($inventoryById.ContainsKey($mapping.inventoryId) -and $mapping.inventoryStatus -ne $inventoryById[$mapping.inventoryId].packageFeasibility) {
        Add-ValidationError "Inventory '$($mapping.inventoryId)' status is '$($inventoryById[$mapping.inventoryId].packageFeasibility)', baseline has '$($mapping.inventoryStatus)'."
    }
    foreach ($packageId in $mapping.localPackageIds) {
        if ($packageId -notin $packageIds) {
            Add-ValidationError "Inventory '$($mapping.inventoryId)' references unknown local package '$packageId'."
        }
    }
}

$siteFiles = @(Get-ChildItem -LiteralPath $siteCatalogPath -Filter '*.md' -File | Sort-Object Name)
$actualSiteSlugs = @($siteFiles | ForEach-Object { $_.BaseName })
$baselineSiteSlugs = @($baseline.publicCatalogEntries | ForEach-Object { $_.slug })
Assert-ExactSet -Label 'Public authored catalog slugs' -Expected $baselineSiteSlugs -Actual $actualSiteSlugs

$siteEntriesBySlug = @{}
foreach ($entry in $baseline.publicCatalogEntries) { $siteEntriesBySlug[$entry.slug] = $entry }
foreach ($siteFile in $siteFiles) {
    if (-not $siteEntriesBySlug.ContainsKey($siteFile.BaseName)) {
        continue
    }

    $entry = $siteEntriesBySlug[$siteFile.BaseName]
    $content = Get-Content -Raw -LiteralPath $siteFile.FullName
    $status = Read-FrontmatterValue -Content $content -Name 'status'
    $protocolJson = Read-FrontmatterValue -Content $content -Name 'protocols'
    if ($entry.status -ne $status) {
        Add-ValidationError "Catalog '$($entry.slug)' status is '$status', baseline has '$($entry.status)'."
    }
    if ([string]::IsNullOrWhiteSpace($protocolJson)) {
        Add-ValidationError "Catalog '$($entry.slug)' has no protocols frontmatter."
    }
    else {
        $protocols = @($protocolJson | ConvertFrom-Json)
        Assert-ExactSet -Label "Catalog '$($entry.slug)' protocols" -Expected @($entry.protocols) -Actual $protocols
    }
    if (-not [string]::IsNullOrWhiteSpace($entry.inventoryId) -and $entry.inventoryId -notin $inventoryIds) {
        Add-ValidationError "Catalog '$($entry.slug)' references unknown inventory ID '$($entry.inventoryId)'."
    }
    foreach ($packageId in $entry.localPackageIds) {
        if ($packageId -notin $packageIds) {
            Add-ValidationError "Catalog '$($entry.slug)' references unknown local package '$packageId'."
        }
    }
}

$statusCounts = @{}
foreach ($entry in $baseline.localImplementationPackages) {
    $statusCounts[$entry.evidenceStatus] = 1 + [int]($statusCounts[$entry.evidenceStatus])
    if (($entry.evidenceStatus -eq 'current-version-live-proven' -or $entry.evidenceStatus -eq 'historical-version-live-proof-only' -or $entry.evidenceStatus -eq 'attempted-no-completed-evidence') -and @($entry.evidenceJobIds).Count -eq 0) {
        Add-ValidationError "Package '$($entry.packageId)' has evidence status '$($entry.evidenceStatus)' without an evidence job ID."
    }
}

$summaryChecks = @{
    localPackageCount = $baseline.localImplementationPackages.Count
    localDecisionDirectoryCount = $baseline.localDecisionDirectories.Count
    quicHttp3InventoryCount = $baseline.quicHttp3InventoryMappings.Count
    publicAuthoredCatalogCount = $baseline.publicCatalogEntries.Count
    currentVersionLiveProvenCount = [int]$statusCounts['current-version-live-proven']
    historicalVersionLiveProofOnlyCount = [int]$statusCounts['historical-version-live-proof-only']
    attemptedNoCompletedEvidenceCount = [int]$statusCounts['attempted-no-completed-evidence']
    noLiveProofInReviewedSourcesCount = [int]$statusCounts['no-live-proof-in-reviewed-sources']
}
foreach ($name in $summaryChecks.Keys) {
    if ([int]$baseline.summary.$name -ne [int]$summaryChecks[$name]) {
        Add-ValidationError "Baseline summary $name is $($baseline.summary.$name), derived is $($summaryChecks[$name])."
    }
}

if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Error $_ }
    throw "Implementation coverage baseline validation failed with $($errors.Count) error(s)."
}

Write-Host "Implementation coverage baseline is complete and current."
Write-Host "  Local packages: $($baseline.localImplementationPackages.Count)"
Write-Host "  Decision directories: $($baseline.localDecisionDirectories.Count)"
Write-Host "  QUIC/HTTP3 inventory entries: $($baseline.quicHttp3InventoryMappings.Count)"
Write-Host "  Public authored catalog entries: $($baseline.publicCatalogEntries.Count)"
Write-Host "  Evidence status counts: current=$($statusCounts['current-version-live-proven']); historical=$($statusCounts['historical-version-live-proof-only']); attempted=$($statusCounts['attempted-no-completed-evidence']); uncited=$($statusCounts['no-live-proof-in-reviewed-sources'])"
