$ErrorActionPreference='Stop'
$root=$PSScriptRoot;$lock=Get-Content (Join-Path $root 'authority-lock.json') -Raw|ConvertFrom-Json
if($lock.commit-ne'8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'){throw 'Authority commit mismatch.'}
foreach($property in $lock.files.PSObject.Properties){$path=Join-Path $root $property.Name;if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw "Missing authority file $($property.Name)"};$hash=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant();if($hash-ne$property.Value){throw "Authority hash mismatch for $($property.Name)"}}
$manifest=Get-Content (Join-Path $root 'protocol-lab-package.json') -Raw|ConvertFrom-Json
if(@($manifest.providedScenarios).Count-ne 7){throw 'Expected seven exact DoH3 scenarios.'}
Write-Host 'DoH3 scenario package authority validation passed.'
