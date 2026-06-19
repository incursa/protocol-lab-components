[CmdletBinding()]
param(
    [string]$ArtifactsRoot = "artifacts/tools",
    [string]$Version = "v0.1.13",
    [switch]$Force,
    [switch]$PassThruJson
)

$ErrorActionPreference = 'Stop'

function Resolve-ComponentPath {
    param([Parameter(Mandatory)][string]$Path)

    if ([System.IO.Path]::IsPathRooted($Path)) {
        return $Path
    }

    return Join-Path $PSScriptRoot $Path
}

$toolRoot = Join-Path (Resolve-ComponentPath -Path $ArtifactsRoot) "h3spec-$Version"
$assetName = "h3spec-linux-x86_64"
$downloadUrl = "https://github.com/summerwind/h3spec/releases/download/$Version/$assetName"
$binaryPath = Join-Path $toolRoot $assetName
$wrapperPath = Join-Path $toolRoot 'Invoke-H3SpecDocker.ps1'

New-Item -ItemType Directory -Force -Path $toolRoot | Out-Null

if ($Force -or -not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $binaryPath
}

$wrapper = @'
[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$H3SpecArguments
)

$ErrorActionPreference = 'Stop'
$toolRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$dockerArguments = @(
    'run', '--rm',
    '-v', "${toolRoot}:/tools:ro",
    'ubuntu:24.04',
    'bash',
    '-lc',
    'cp /tools/h3spec-linux-x86_64 /tmp/h3spec && chmod +x /tmp/h3spec && exec /tmp/h3spec "$@"',
    'h3spec'
) + $H3SpecArguments

& docker @dockerArguments
exit $LASTEXITCODE
'@

Set-Content -LiteralPath $wrapperPath -Value $wrapper -Encoding UTF8

$manifest = [ordered]@{
    tool = 'h3spec'
    version = $Version
    asset = $assetName
    downloadUrl = $downloadUrl
    executable = $wrapperPath
    prefixArguments = @()
    wrapperImage = 'ubuntu:24.04'
    recommendedHostName = 'host.docker.internal'
}

$manifestPath = Join-Path $toolRoot 'h3spec-tool-manifest.json'
$manifest | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $manifestPath -Encoding UTF8

if ($PassThruJson) {
    $manifest | ConvertTo-Json -Depth 5
}
else {
    Write-Host "Prepared h3spec $Version at $toolRoot"
    Write-Host "Wrapper: $wrapperPath"
}
