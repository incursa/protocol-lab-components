$ErrorActionPreference='Stop'
$root=$PSScriptRoot;$lock=Get-Content (Join-Path $root 'authority-lock.json') -Raw|ConvertFrom-Json
if($lock.commit-ne'c0475b05cb80362760ac57e58ecfa1610a766c10'){throw 'Authority commit mismatch.'}
foreach($property in $lock.files.PSObject.Properties){$path=Join-Path $root $property.Name;if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw "Missing authority file $($property.Name)"};$hash=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant();if($hash-ne$property.Value){throw "Authority hash mismatch for $($property.Name)"}}
$manifest=Get-Content (Join-Path $root 'protocol-lab-package.json') -Raw|ConvertFrom-Json
if(@($manifest.providedScenarios).Count-ne 8){throw 'Expected seven strict DoH3 scenarios plus one authoritative interoperability scenario.'}
Write-Host 'DoH3 scenario package authority validation passed.'
