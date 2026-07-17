[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$PackageDirectory = (Join-Path $Root 'artifacts/packages'),
    [string]$OutputRoot = (Join-Path $Root 'artifacts/grpc-h2-streaming-smoke'),
    [int]$Port = 19445
)

$ErrorActionPreference = 'Stop'
$Root = [IO.Path]::GetFullPath($Root)
$PackageDirectory = [IO.Path]::GetFullPath($PackageDirectory)
$OutputRoot = [IO.Path]::GetFullPath($OutputRoot)
$packages = [ordered]@{
    scenario = Join-Path $PackageDirectory 'org.protocol-lab.components.scenario.grpc-h2-performance.0.4.2.plabpkg'
    executor = Join-Path $PackageDirectory 'org.protocol-lab.components.executor.go-grpc-h2-executor.0.4.2.win-x64.plabpkg'
    target = Join-Path $PackageDirectory 'org.protocol-lab.components.implementation.go-grpc-h2.0.4.0.win-x64.plabpkg'
}
foreach ($entry in $packages.GetEnumerator()) {
    if (-not (Test-Path -LiteralPath $entry.Value -PathType Leaf)) { throw "Missing $($entry.Key) package: $($entry.Value)" }
}

Remove-Item -LiteralPath $OutputRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null
Add-Type -AssemblyName System.IO.Compression.FileSystem
foreach ($entry in $packages.GetEnumerator()) {
    [IO.Compression.ZipFile]::ExtractToDirectory($entry.Value, (Join-Path $OutputRoot $entry.Key))
}
& (Join-Path $OutputRoot 'scenario/validate.ps1')

$targetRoot = Join-Path $OutputRoot 'target'
$executorRoot = Join-Path $OutputRoot 'executor'
$target = Start-Process -FilePath (Join-Path $targetRoot 'bin/win-x64/go-grpc-h2.exe') `
    -WorkingDirectory $targetRoot -ArgumentList @('--listen', "127.0.0.1:$Port") -PassThru -WindowStyle Hidden `
    -RedirectStandardOutput (Join-Path $OutputRoot 'target.stdout.log') -RedirectStandardError (Join-Path $OutputRoot 'target.stderr.log')
Start-Sleep -Milliseconds 750
try {
    $scenarios = @(
        @{ id = 'grpc.h2.unary.echo'; requests = 1; responses = 1 },
        @{ id = 'grpc.h2.server-streaming.echo'; requests = 1; responses = 100 },
        @{ id = 'grpc.h2.client-streaming.echo'; requests = 100; responses = 1 },
        @{ id = 'grpc.h2.bidi-streaming.echo'; requests = 100; responses = 100 }
    )
    foreach ($scenario in $scenarios) {
        $env:PLAB_EXECUTOR_ID = 'go-grpc-h2-executor'
        $env:PLAB_EXECUTOR_VERSION = '0.4.2'
        $env:PLAB_LOAD_GENERATOR_ID = 'go-x-net-http2-grpc-load'
        $env:PLAB_LOAD_GENERATOR_VERSION = '0.4.2'
        $env:PLAB_SCENARIO_ID = $scenario.id
        $env:PLAB_LOAD_PROFILE_ID = 'grpc-h2-smoke'
        $env:PLAB_PROTOCOL = 'h2'
        $env:PLAB_PROTOCOL_VARIANT = 'grpc-over-h2-tls-alpn'
        $env:PLAB_CONNECTIONS = '1'
        $env:PLAB_CONCURRENCY = '1'
        $env:PLAB_STREAMS_PER_CONNECTION = '1'
        $env:PLAB_DURATION_SECONDS = '5'
        $env:PLAB_WARMUP_SECONDS = '1'
        $env:PLAB_REPETITION = '1'
        $artifactRoot = Join-Path $OutputRoot ("evidence/" + $scenario.id.Replace('.', '-'))
        New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
        $process = Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/go-grpc-h2-executor.exe') `
            -WorkingDirectory $executorRoot -ArgumentList @('--target-url', "https://127.0.0.1:$Port", '--output-dir', $artifactRoot) `
            -Wait -PassThru -WindowStyle Hidden -RedirectStandardOutput (Join-Path $artifactRoot 'load.stdout.log') -RedirectStandardError (Join-Path $artifactRoot 'load.stderr.log')
        if ($process.ExitCode -ne 0) { throw "$($scenario.id) executor exited $($process.ExitCode)." }
        $result = Get-Content -LiteralPath (Join-Path $artifactRoot 'result.json') -Raw | ConvertFrom-Json
        if ($result.passed -ne $true -or $result.scenarioId -ne $scenario.id -or $result.request.count -ne $scenario.requests -or $result.response.count -ne $scenario.responses -or $result.metrics.completedOperations -ne 1 -or $result.metrics.failedOperations -ne 0 -or $result.metrics.timedOutOperations -ne 0) {
            throw "$($scenario.id) extracted-package evidence failed the exact outcome/cardinality gate."
        }
        if ($scenario.requests -gt 1 -and $result.observation.clientHalfClosed -ne $true) { throw "$($scenario.id) did not prove client half-close." }
        if ($scenario.responses -gt 1 -and $result.observation.streamComplete -ne $true) { throw "$($scenario.id) did not prove stream completion." }
        if ($scenario.id -eq 'grpc.h2.bidi-streaming.echo' -and $result.observation.orderedEcho -ne $true) { throw "$($scenario.id) did not prove ordered echo." }
        Write-Host "$($scenario.id): requests=$($scenario.requests) responses=$($scenario.responses) completed=1 failed=0 timedOut=0"
    }
}
finally {
    Stop-Process -Id $target.Id -Force -ErrorAction SilentlyContinue
    Wait-Process -Id $target.Id -ErrorAction SilentlyContinue
}
