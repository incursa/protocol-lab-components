[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
)

$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path -LiteralPath $Root).Path
$documents = @(
    'README.md',
    'CONTRIBUTING.md',
    'docs/README.md',
    'docs/third-party-package-consumption.md',
    'scripts/package/README.md'
)
$errors = [System.Collections.Generic.List[string]]::new()
$linkPattern = [regex]'!?(?:\[[^\]]*\])\((?<target>[^)]+)\)'
$localPathPattern = [regex]'(?<![A-Za-z0-9])(?<path>[A-Za-z]:[\\/])'

foreach ($relativeDocument in $documents) {
    $documentPath = Join-Path $Root $relativeDocument
    if (-not (Test-Path -LiteralPath $documentPath -PathType Leaf)) {
        [void]$errors.Add("Missing public documentation entrypoint: $relativeDocument")
        continue
    }

    $content = Get-Content -LiteralPath $documentPath -Raw
    if ($localPathPattern.IsMatch($content)) {
        [void]$errors.Add("$relativeDocument contains an absolute workstation path.")
    }

    foreach ($match in $linkPattern.Matches($content)) {
        $target = $match.Groups['target'].Value.Trim().Trim('<', '>')
        if ($target -match '^(?:https?://|mailto:|#)') {
            continue
        }

        $pathPart = ($target -split '#', 2)[0]
        if ([string]::IsNullOrWhiteSpace($pathPart)) {
            continue
        }

        if ([System.IO.Path]::IsPathRooted($pathPart)) {
            [void]$errors.Add("$relativeDocument links to an absolute local path: $target")
            continue
        }

        $resolvedTarget = [System.IO.Path]::GetFullPath((Join-Path (Split-Path -Parent $documentPath) $pathPart))
        if (-not (Test-Path -LiteralPath $resolvedTarget)) {
            [void]$errors.Add("$relativeDocument has a broken repository link: $target")
        }
    }
}

if ($errors.Count -gt 0) {
    throw ($errors -join [Environment]::NewLine)
}

Write-Host "Validated $($documents.Count) public documentation entrypoint(s)."
