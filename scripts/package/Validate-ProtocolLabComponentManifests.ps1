[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
)

$ErrorActionPreference = 'Stop'

$manifestFiles = Get-ChildItem -LiteralPath $Root -Recurse -Filter 'package.protocol-lab.json' |
    Where-Object {
        $_.FullName -notmatch '[\\/](artifacts|packages|bin|obj)[\\/]'
    }

if (-not $manifestFiles) {
    throw "No package.protocol-lab.json files found under $Root."
}

$ids = @{}
$errors = New-Object System.Collections.Generic.List[string]

foreach ($file in $manifestFiles) {
    try {
        $manifest = Get-Content -LiteralPath $file.FullName -Raw | ConvertFrom-Json
    }
    catch {
        $errors.Add("$($file.FullName): invalid JSON: $($_.Exception.Message)")
        continue
    }

    foreach ($required in @('packageId', 'packageKind', 'version', 'component', 'entrypoints', 'requirements', 'provenance')) {
        if (-not $manifest.PSObject.Properties.Name.Contains($required)) {
            $errors.Add("$($file.FullName): missing required property '$required'.")
        }
    }

    if ($manifest.packageKind -and $manifest.packageKind -notin @('implementation', 'test-executor')) {
        $errors.Add("$($file.FullName): packageKind must be 'implementation' or 'test-executor'.")
    }

    if ($manifest.version -and $manifest.version -notmatch '^\d+\.\d+\.\d+([+-][0-9A-Za-z.-]+)?$') {
        $errors.Add("$($file.FullName): version '$($manifest.version)' is not a semantic version.")
    }

    if ($manifest.packageId) {
        if ($ids.ContainsKey($manifest.packageId)) {
            $errors.Add("$($file.FullName): duplicate packageId '$($manifest.packageId)' also used by $($ids[$manifest.packageId]).")
        }
        else {
            $ids[$manifest.packageId] = $file.FullName
        }
    }
}

if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Error $_ }
    throw "Protocol Lab component manifest validation failed with $($errors.Count) error(s)."
}

Write-Host "Validated $($manifestFiles.Count) Protocol Lab component manifest(s)."
