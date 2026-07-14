[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),[switch]$AllowDirtySource)
$ErrorActionPreference='Stop'
$manifest=Get-Content (Join-Path $Root 'scenarios/dns-doq-performance/protocol-lab-package.json') -Raw|ConvertFrom-Json
if($manifest.packageVersion-ne '0.1.1'){throw 'DoQ scenario package version must be 0.1.1.'}
if((@($manifest.providedSuites.suiteId)-join ',')-ne 'dns-doq-performance-smoke'){throw 'DoQ scenario package providedSuites mismatch.'}
& (Join-Path $Root 'scenarios/dns-doq-performance/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -ComponentPath 'scenarios/dns-doq-performance' -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
