[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$lock=Get-Content -LiteralPath (Join-Path $PSScriptRoot 'authority-lock.json') -Raw|ConvertFrom-Json
if($lock.commit-ne'5b113ee75e6f4e329f638751580c9e6cf0c9a99e'){throw "Unexpected authority commit $($lock.commit)."}
foreach($property in $lock.files.PSObject.Properties){$path=Join-Path $PSScriptRoot $property.Name;if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw "Authority-locked file missing: $($property.Name)"};$actual=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant();if($actual-ne[string]$property.Value){throw "Authority hash mismatch for $($property.Name): expected $($property.Value), observed $actual"}}
$manifest=Get-Content -LiteralPath (Join-Path $PSScriptRoot 'protocol-lab-package.json') -Raw|ConvertFrom-Json
if($manifest.packageVersion-ne'0.1.0'){throw 'WebTransport scenario package version must be 0.1.0.'}
if((@($manifest.providedSuites.suiteId)-join ',')-ne'webtransport-performance-smoke'){throw 'WebTransport suite declaration mismatch.'}
if('load-profiles/webtransport-smoke.yaml'-notin@($manifest.entryManifests)){throw 'WebTransport load-profile entry manifest missing.'}
Write-Output "Validated WebTransport scenario package authority lock at $($lock.commit)."
