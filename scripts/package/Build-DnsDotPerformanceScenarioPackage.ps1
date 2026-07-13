[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),[switch]$AllowDirtySource)
$ErrorActionPreference='Stop'
& (Join-Path $Root 'scenarios/dns-dot-performance/validate.ps1')
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -ComponentPath 'scenarios/dns-dot-performance' -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
