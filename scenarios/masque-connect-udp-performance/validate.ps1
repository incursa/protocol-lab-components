[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$expectedCommit='d8082f5ae7d872e9e556f9696177079929483c58'
$lock=Get-Content -LiteralPath (Join-Path $PSScriptRoot 'authority-lock.json') -Raw|ConvertFrom-Json
if($lock.commit-ne$expectedCommit){throw "Unexpected authority commit $($lock.commit)."}
foreach($property in $lock.files.PSObject.Properties){$path=Join-Path $PSScriptRoot $property.Name;if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw "Authority-locked file missing: $($property.Name)"};$actual=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant();if($actual-ne[string]$property.Value){throw "Authority hash mismatch for $($property.Name): expected $($property.Value), observed $actual"}}
$manifest=Get-Content -LiteralPath (Join-Path $PSScriptRoot 'protocol-lab-package.json') -Raw|ConvertFrom-Json
if($manifest.packageVersion-ne'0.1.1'){throw 'MASQUE scenario package version must be 0.1.1.'}
if((@($manifest.providedSuites.suiteId)-join ',')-ne'masque-connect-udp-performance-comparison'){throw 'MASQUE suite declaration mismatch.'}
if('load-profiles/masque-connect-udp-comparison.yaml'-notin@($manifest.entryManifests)){throw 'MASQUE load-profile entry manifest missing.'}
Write-Output "Validated MASQUE scenario package authority lock at $($lock.commit)."
