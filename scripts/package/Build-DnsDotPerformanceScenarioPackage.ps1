[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),[switch]$AllowDirtySource)
$ErrorActionPreference='Stop'
$manifest=Get-Content (Join-Path $Root 'scenarios/dns-dot-performance/protocol-lab-package.json') -Raw|ConvertFrom-Json
if($manifest.packageVersion-ne '0.2.0'){throw 'DoT scenario package version must be 0.2.0.'}
if((@($manifest.providedSuites.suiteId)-join ',')-ne 'dns-dot-performance-smoke,dns-dot-interoperability-smoke'){throw 'DoT scenario package providedSuites mismatch.'}
& (Join-Path $Root 'scenarios/dns-dot-performance/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -ComponentPath 'scenarios/dns-dot-performance' -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
