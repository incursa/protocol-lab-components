[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),[switch]$AllowDirtySource)
$ErrorActionPreference='Stop'
& (Join-Path $Root 'scenarios/webtransport-performance/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath 'scenarios/webtransport-performance' -AllowDirtySource:$AllowDirtySource
