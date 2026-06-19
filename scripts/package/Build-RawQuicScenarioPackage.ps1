[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,

    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages')
)

$ErrorActionPreference = 'Stop'

& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
    -ComponentPath 'scenarios/raw-quic-transport' `
    -Root $Root `
    -OutputRoot $OutputRoot
