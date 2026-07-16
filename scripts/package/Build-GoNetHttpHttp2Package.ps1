[CmdletBinding()]
param(
    [ValidateSet('win-x64','linux-x64')][string]$RuntimeIdentifier='win-x64',
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),
    [switch]$SkipSmoke,
    [switch]$AllowDirtySource
)
& (Join-Path $PSScriptRoot 'Build-GoNetHttpOriginPackage.ps1') -ComponentName go-nethttp-http2 -RuntimeIdentifier $RuntimeIdentifier -Root $Root -OutputRoot $OutputRoot -SkipSmoke:$SkipSmoke -AllowDirtySource:$AllowDirtySource
