[CmdletBinding()]
param(
    [string]$HostName = "127.0.0.1",
    [int]$Port = 4433,
    [string]$H3SpecExecutable = "h3spec",
    [string[]]$H3SpecPrefixArguments = @(),
    [string[]]$Match = @("HTTP/3", "QPACK"),
    [string[]]$Skip = @(),
    [ValidateSet("focused", "full", "qpack")]
    [string]$Mode = "focused",
    [int]$TimeoutMilliseconds = 5000,
    [string]$OutputRoot = "artifacts/h3spec-http3-qpack",
    [switch]$AcquireH3Spec,
    [string]$AcquireH3SpecVersion = "v0.1.13",
    [switch]$NoValidateCertificate,
    [switch]$PlanOnly,
    [switch]$FailOnH3SpecFailures
)

$ErrorActionPreference = 'Stop'

function Resolve-ComponentPath {
    param([Parameter(Mandatory)][string]$Path)

    if ([System.IO.Path]::IsPathRooted($Path)) {
        return $Path
    }

    return Join-Path $PSScriptRoot $Path
}

function Get-PythonCommand {
    foreach ($name in @('python', 'python3', 'py')) {
        $command = Get-Command $name -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($null -ne $command) {
            return $command.Source
        }
    }

    return $null
}

function Invoke-H3SpecParser {
    param(
        [Parameter(Mandatory)][string]$StdoutPath,
        [Parameter(Mandatory)][string]$StderrPath,
        [Parameter(Mandatory)][string]$MetadataPath,
        [Parameter(Mandatory)][string]$ResultsPath,
        [Parameter(Mandatory)][string]$ReportPath
    )

    $python = Get-PythonCommand
    if ([string]::IsNullOrWhiteSpace($python)) {
        $metadata = Get-Content -LiteralPath $MetadataPath -Raw | ConvertFrom-Json
        [ordered]@{
            tool = 'h3spec'
            metadata = $metadata
            summary = [ordered]@{
                status = if ($PlanOnly) { 'not-run' } else { 'unparsed' }
                exitCode = $metadata.exitCode
                selectedCases = 0
                selectionStatus = if (@($metadata.match).Count -gt 0) { 'no-selected-cases' } else { 'unfiltered' }
                failures = 0
            }
            cases = @()
            failures = @()
        } | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $ResultsPath -Encoding UTF8

        @(
            '# h3spec HTTP/3 Server Triage Report'
            ''
            'Python was not available, so stdout/stderr parsing was skipped.'
        ) | Set-Content -LiteralPath $ReportPath -Encoding UTF8
        return
    }

    $parserPath = Join-Path $PSScriptRoot 'scripts/parse-h3spec-results.py'
    $parserArguments = @(
        $parserPath,
        '--stdout', $StdoutPath,
        '--stderr', $StderrPath,
        '--metadata', $MetadataPath,
        '--json-output', $ResultsPath,
        '--markdown-output', $ReportPath
    )

    if ((Split-Path -Leaf $python) -match '^py(\.exe)?$') {
        & $python -3 @parserArguments
    }
    else {
        & $python @parserArguments
    }

    if ($LASTEXITCODE -ne 0) {
        throw "h3spec parser failed with exit code $LASTEXITCODE."
    }
}

$resolvedOutputRoot = Resolve-ComponentPath -Path $OutputRoot
New-Item -ItemType Directory -Force -Path $resolvedOutputRoot | Out-Null

$stdoutPath = Join-Path $resolvedOutputRoot 'h3spec-stdout.txt'
$stderrPath = Join-Path $resolvedOutputRoot 'h3spec-stderr.txt'
$metadataPath = Join-Path $resolvedOutputRoot 'h3spec-metadata.json'
$resultsPath = Join-Path $resolvedOutputRoot 'h3spec-results.json'
$reportPath = Join-Path $resolvedOutputRoot 'h3spec-report.md'

switch ($Mode) {
    "full" {
        $Match = @()
    }
    "qpack" {
        $Match = @("QPACK")
    }
}

$effectiveHostName = $HostName
if ($AcquireH3Spec) {
    $toolManifestJson = & (Join-Path $PSScriptRoot 'Install-H3SpecTool.ps1') -Version $AcquireH3SpecVersion -PassThruJson
    $toolManifest = $toolManifestJson | ConvertFrom-Json
    $H3SpecExecutable = [string]$toolManifest.executable
    $H3SpecPrefixArguments = @($toolManifest.prefixArguments)

    if ($HostName -in @('127.0.0.1', 'localhost') -and -not [string]::IsNullOrWhiteSpace([string]$toolManifest.recommendedHostName)) {
        $effectiveHostName = [string]$toolManifest.recommendedHostName
    }
}

$h3specArguments = @()
if ($NoValidateCertificate) {
    $h3specArguments += '--no-validate'
}

foreach ($item in $Match) {
    $h3specArguments += '--match'
    $h3specArguments += $item
}

foreach ($item in $Skip) {
    $h3specArguments += '--skip'
    $h3specArguments += $item
}

$h3specArguments += '--timeout'
$h3specArguments += "$TimeoutMilliseconds"
$h3specArguments += $effectiveHostName
$h3specArguments += "$Port"

$metadata = [ordered]@{
    executor = 'h3spec-http3-qpack'
    mode = $Mode
    tool = 'h3spec'
    toolVersion = $AcquireH3SpecVersion
    executable = $H3SpecExecutable
    prefixArguments = $H3SpecPrefixArguments
    arguments = $h3specArguments
    match = $Match
    skip = $Skip
    timeoutMilliseconds = $TimeoutMilliseconds
    noValidateCertificate = [bool]$NoValidateCertificate
    host = $HostName
    h3specTargetHost = $effectiveHostName
    port = $Port
    planOnly = [bool]$PlanOnly
    exitCode = $null
    status = 'not-run'
}

try {
    if ($PlanOnly) {
        Set-Content -LiteralPath $stdoutPath -Value "plan-only: $H3SpecExecutable $($H3SpecPrefixArguments + $h3specArguments -join ' ')" -Encoding UTF8
        Set-Content -LiteralPath $stderrPath -Value "" -Encoding UTF8
    }
    else {
        $invocationArguments = @($H3SpecPrefixArguments) + $h3specArguments
        & $H3SpecExecutable @invocationArguments > $stdoutPath 2> $stderrPath
        $exitCode = $LASTEXITCODE
        if ($null -eq $exitCode) {
            $exitCode = if ($?) { 0 } else { 1 }
        }

        $metadata['exitCode'] = $exitCode
        $metadata['status'] = if ($exitCode -eq 0) { 'passed' } else { 'failed' }
    }
}
finally {
    $metadata | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $metadataPath -Encoding UTF8
}

Invoke-H3SpecParser -StdoutPath $stdoutPath -StderrPath $stderrPath -MetadataPath $metadataPath -ResultsPath $resultsPath -ReportPath $reportPath

$parsed = Get-Content -LiteralPath $resultsPath -Raw | ConvertFrom-Json
if ($FailOnH3SpecFailures -and -not $PlanOnly) {
    if ([string]$parsed.summary.selectionStatus -eq 'no-selected-cases') {
        throw "h3spec selected no cases for the requested match filters. See $reportPath."
    }

    if ([int]$parsed.summary.failures -gt 0 -or ([int]$parsed.summary.exitCode) -ne 0) {
        throw "h3spec reported failures. See $reportPath."
    }
}

Write-Host "h3spec executor artifacts written to $resolvedOutputRoot"
