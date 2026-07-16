[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$lock=Get-Content -LiteralPath (Join-Path $PSScriptRoot 'authority-lock.json') -Raw|ConvertFrom-Json
if($lock.commit-ne'cc149c76a3567b823491063d1a2bd42216539f70'){throw "Unexpected authority commit $($lock.commit)."}
foreach($property in $lock.files.PSObject.Properties){$path=Join-Path $PSScriptRoot $property.Name;if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw "Authority-locked file is missing: $($property.Name)"};$actual=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant();if($actual-ne[string]$property.Value){throw "Authority-locked file hash mismatch for $($property.Name): expected $($property.Value), observed $actual"}}
Write-Output "Validated DNS-over-TLS scenario package authority lock at $($lock.commit)."
