[CmdletBinding()]
param(
    [int]$Port = 5447,
    [switch]$PlanOnly,
    [string]$OutputRoot = 'artifacts/quic-go-raw'
)

$ErrorActionPreference = 'Stop'

$componentRoot = $PSScriptRoot
$artifactRoot = if ([System.IO.Path]::IsPathRooted($OutputRoot)) { $OutputRoot } else { Join-Path $componentRoot $OutputRoot }
New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null

$commandPath = Join-Path $artifactRoot 'command.txt'
$resultPath = Join-Path $artifactRoot 'result.json'
if ($IsWindows) {
    $binaryRelativePath = 'bin/windows-x64/quic-go-raw.exe'
}
elseif ($IsLinux) {
    $binaryRelativePath = 'bin/linux-x64/quic-go-raw'
}
else {
    throw 'quic-go-raw only supports Windows and Linux hosts.'
}

$binaryPath = Join-Path $componentRoot $binaryRelativePath
$command = $binaryRelativePath

Set-Content -LiteralPath $commandPath -Value $command

if ($PlanOnly) {
    [ordered]@{
        status = 'planned'
        port = $Port
        command = $command
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $resultPath
    Write-Host "Planned quic-go raw QUIC command at $commandPath"
    return
}

$env:PLAB_QUIC_PORT = [string]$Port
& $binaryPath
