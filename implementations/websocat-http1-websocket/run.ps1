[CmdletBinding()]
param(
    [string]$Image = 'incursa-protocol-lab-websocat-http1-websocket:0.1.0',
    [int]$Port = 18082,
    [switch]$SkipBuild,
    [switch]$PlanOnly,
    [switch]$ProofOnly,
    [string]$OutputRoot = 'artifacts/websocat-http1-websocket'
)
$ErrorActionPreference = 'Stop'
$componentRoot = $PSScriptRoot
$artifactRoot = if ([IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $componentRoot $OutputRoot }
New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
$buildArgs = @('build','--pull','-f','docker/Websocat.Dockerfile','-t',$Image,'docker')
$proofArgs = @('run','--rm','--entrypoint','/usr/local/bin/websocat',$Image,'--version')
$serverArgs = @('run','--rm','-p',"${Port}:18081/tcp",$Image)
@("docker $($buildArgs -join ' ')","docker $($proofArgs -join ' ')","docker $($serverArgs -join ' ')") | Set-Content (Join-Path $artifactRoot 'command.txt')
if ($PlanOnly) { @{status='planned';image=$Image;port=$Port} | ConvertTo-Json | Set-Content (Join-Path $artifactRoot 'result.json'); return }
Push-Location $componentRoot
try {
    if (-not $SkipBuild) { & docker @buildArgs; if ($LASTEXITCODE -ne 0) { throw "Docker build failed with exit code $LASTEXITCODE." } }
    $version = (& docker @proofArgs 2>&1 | Out-String).Trim(); if ($LASTEXITCODE -ne 0 -or $version -notmatch '1\.14\.1') { throw "websocat version proof failed: $version" }
    $version | Set-Content (Join-Path $artifactRoot 'version.txt')
    if ($ProofOnly) { @{status='proved';image=$Image;version=$version} | ConvertTo-Json | Set-Content (Join-Path $artifactRoot 'result.json'); return }
    & docker @serverArgs > (Join-Path $artifactRoot 'stdout.txt') 2> (Join-Path $artifactRoot 'stderr.txt')
} finally { Pop-Location }
