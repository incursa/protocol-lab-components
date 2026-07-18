[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$ComponentId,
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$GraphPath = (Join-Path $Root 'release/component-graph.v1.json')
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'ProtocolLabComponentRelease.Common.ps1')
$graph = Get-ProtocolLabComponentReleaseGraph -GraphPath $GraphPath
Get-ProtocolLabComponentClosure -Graph $graph -ComponentId $ComponentId -Root $Root | ConvertTo-Json -Depth 32
