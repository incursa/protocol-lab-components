[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$lock=Get-Content -LiteralPath (Join-Path $PSScriptRoot 'authority-lock.json') -Raw|ConvertFrom-Json
if($lock.commit-ne'dd518aee19d73fb1477320644785fa070b1b62f1'){throw "Unexpected authority commit $($lock.commit)."}
foreach($property in $lock.files.PSObject.Properties){$path=Join-Path $PSScriptRoot $property.Name;if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw "Authority-locked file missing: $($property.Name)"};$actual=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant();if($actual-ne[string]$property.Value){throw "Authority hash mismatch for $($property.Name): expected $($property.Value), observed $actual"}}
$manifest=Get-Content -LiteralPath (Join-Path $PSScriptRoot 'protocol-lab-package.json') -Raw|ConvertFrom-Json
if($manifest.packageVersion-ne'0.2.0'){throw 'WebTransport scenario package version must be 0.2.0.'}
if((@($manifest.providedSuites.suiteId)-join ',')-ne'webtransport-performance-smoke,webtransport-datagram-performance-smoke'){throw 'WebTransport suite declaration mismatch.'}
foreach($path in @('load-profiles/webtransport-smoke.yaml','load-profiles/webtransport-datagram-smoke.yaml')){if($path-notin@($manifest.entryManifests)){throw "WebTransport load-profile entry manifest missing: $path"}}
Write-Output "Validated WebTransport scenario package authority lock at $($lock.commit)."
