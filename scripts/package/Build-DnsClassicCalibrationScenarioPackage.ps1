[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),[switch]$AllowDirtySource)
$ErrorActionPreference='Stop'
if(-not[IO.Path]::IsPathRooted($OutputRoot)){$OutputRoot=Join-Path $Root $OutputRoot}
& (Join-Path $Root 'scenarios/dns-classic-calibration/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -ComponentPath 'scenarios/dns-classic-calibration' -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
