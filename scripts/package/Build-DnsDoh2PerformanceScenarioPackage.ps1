[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),[switch]$AllowDirtySource)
$ErrorActionPreference='Stop'
$manifest=Get-Content (Join-Path $Root 'scenarios/dns-doh2-performance/protocol-lab-package.json') -Raw|ConvertFrom-Json
if($manifest.packageVersion-ne '0.1.1'){throw 'DoH2 scenario package version must be 0.1.1.'}
if((@($manifest.providedSuites.suiteId)-join ',')-ne 'dns-doh2-performance-smoke'){throw 'DoH2 scenario package providedSuites mismatch.'}
& (Join-Path $Root 'scenarios/dns-doh2-performance/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -ComponentPath 'scenarios/dns-doh2-performance' -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
