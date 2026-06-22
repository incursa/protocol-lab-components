[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages')
)

$ErrorActionPreference = 'Stop'

& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -ComponentPath 'scenarios/http3-peer-characterization' `
    -Root $Root `
    -OutputRoot $OutputRoot
