[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$lock = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'authority-lock.json') -Raw | ConvertFrom-Json
foreach ($property in $lock.files.PSObject.Properties) {
    $path = Join-Path $PSScriptRoot $property.Name
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Authority-locked file is missing: $($property.Name)"
    }
    $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne [string]$property.Value) {
        throw "Authority-locked file hash mismatch for $($property.Name): expected $($property.Value), observed $actual"
    }
}
Write-Output "Validated HTTP/1 scenario package authority lock at $($lock.commit)."
