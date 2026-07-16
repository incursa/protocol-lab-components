[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'))
$ErrorActionPreference='Stop'
[xml](Get-Content (Join-Path $Root 'implementations/jetty-http-origin/source/web.xml') -Raw) | Out-Null
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath 'implementations/jetty-http-origin'
