[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),[switch]$AllowDirtySource)
$ErrorActionPreference='Stop'
$manifest=Get-Content (Join-Path $Root 'implementations/webtransport-go/protocol-lab-package.json') -Raw|ConvertFrom-Json
if($manifest.packageVersion-ne'0.1.0'){throw 'webtransport-go package version must be 0.1.0.'}
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath 'implementations/webtransport-go' -AllowDirtySource:$AllowDirtySource
