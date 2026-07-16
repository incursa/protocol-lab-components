[CmdletBinding()]
param(
  [string]$Image = 'incursa-protocol-lab-bind9-classic-authority:0.1.0',
  [int]$Port = 15354,
  [switch]$SkipBuild,
  [switch]$PlanOnly,
  [switch]$ProofOnly,
  [string]$OutputRoot = 'artifacts/bind9-classic-authority'
)
$ErrorActionPreference = 'Stop'
$out = if ([IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $PSScriptRoot $OutputRoot }
New-Item -ItemType Directory -Force $out | Out-Null
$build = @('build', '--pull', '-t', $Image, 'docker')
$proof = @('run', '--rm', '--entrypoint', 'named', $Image, '-V')
$run = @('run', '--rm', '-p', "${Port}:53/udp", '-p', "${Port}:53/tcp", $Image)
@('docker ' + ($build -join ' '), 'docker ' + ($proof -join ' '), 'docker ' + ($run -join ' ')) | Set-Content (Join-Path $out 'command.txt')
if ($PlanOnly) {
  @{ status = 'planned'; image = $Image; protocols = @('dns-udp', 'dns-tcp') } | ConvertTo-Json | Set-Content (Join-Path $out 'result.json')
  return
}
Push-Location $PSScriptRoot
try {
  if (-not $SkipBuild) {
    & docker @build
    if ($LASTEXITCODE -ne 0) { throw 'Docker build failed.' }
  }
  $version = (& docker @proof 2>&1 | Out-String).Trim()
  $version | Set-Content (Join-Path $out 'version.txt')
  if ($version -notmatch [regex]::Escape('BIND 9.20.24')) { throw "Version proof failed: $version" }
  if ($ProofOnly) {
    @{ status = 'proved'; version = $version; protocols = @('dns-udp', 'dns-tcp') } | ConvertTo-Json | Set-Content (Join-Path $out 'result.json')
    return
  }
  & docker @run
} finally {
  Pop-Location
}
