[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),[switch]$AllowDirtySource)
$ErrorActionPreference='Stop'
$manifest=Get-Content (Join-Path $Root 'scenarios/dns-doh2-performance/protocol-lab-package.json') -Raw|ConvertFrom-Json
if($manifest.packageVersion-ne '0.3.0'){throw 'DoH2 scenario package version must be 0.3.0.'}
if((@($manifest.providedSuites.suiteId)-join ',')-ne 'dns-doh2-performance-smoke,dns-doh2-interoperability-smoke,dns-doh2-resolver-interoperability-smoke'){throw 'DoH2 scenario package providedSuites mismatch.'}
& (Join-Path $Root 'scenarios/dns-doh2-performance/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -ComponentPath 'scenarios/dns-doh2-performance' -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
