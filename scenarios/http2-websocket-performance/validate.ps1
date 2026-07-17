[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$lock=Get-Content -LiteralPath (Join-Path $PSScriptRoot 'authority-lock.json') -Raw|ConvertFrom-Json
if($lock.commit-ne'8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'){throw "Unexpected authority commit $($lock.commit)."}
foreach($property in $lock.files.PSObject.Properties){$path=Join-Path $PSScriptRoot $property.Name;if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw "Authority-locked file missing: $($property.Name)"};$actual=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant();if($actual-ne[string]$property.Value){throw "Authority hash mismatch for $($property.Name): expected $($property.Value), observed $actual"}}
$manifest=Get-Content -LiteralPath (Join-Path $PSScriptRoot 'protocol-lab-package.json') -Raw|ConvertFrom-Json
if($manifest.packageVersion-ne'0.1.2'){throw 'HTTP/2 WebSocket scenario package version must be 0.1.2.'}
if((@($manifest.providedSuites.suiteId)-join ',')-ne'http2-websocket-performance-smoke,http2-websocket-performance-comparison,http2-websocket-multi-message-diagnostic'){throw 'HTTP/2 WebSocket suite declarations mismatch.'}
$expectedLoadProfiles=@('load-profiles/websocket-smoke.yaml','load-profiles/websocket-comparison.yaml','load-profiles/diagnostic.yaml')
foreach($loadProfile in $expectedLoadProfiles){if($loadProfile-notin@($manifest.entryManifests)){throw "HTTP/2 WebSocket load-profile entry manifest missing: $loadProfile"}}
Write-Output "Validated HTTP/2 WebSocket scenario package authority lock at $($lock.commit)."
