[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$PackageDirectory = (Join-Path $Root 'artifacts/packages'),
    [string]$OutputRoot = (Join-Path $Root 'artifacts/grpc-h2-terminal-channel-smoke'),
    [int]$Port = 19446
)

$ErrorActionPreference = 'Stop'
$Root = [IO.Path]::GetFullPath($Root)
$PackageDirectory = [IO.Path]::GetFullPath($PackageDirectory)
$OutputRoot = [IO.Path]::GetFullPath($OutputRoot)
$packages = [ordered]@{
    scenario = Join-Path $PackageDirectory 'org.protocol-lab.components.scenario.grpc-h2-performance.0.4.1.plabpkg'
    executor = Join-Path $PackageDirectory 'org.protocol-lab.components.executor.go-grpc-h2-executor.0.4.1.win-x64.plabpkg'
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
        @{ id = 'grpc.h2.trailers-only-status'; profile = 'grpc-h2-diagnostic'; duration = '10'; repetition = '1'; variant = 'grpc-over-h2-tls-alpn' },
        @{ id = 'grpc.h2.deadline-exceeded'; profile = 'grpc-h2-diagnostic'; duration = '10'; repetition = '1'; variant = 'grpc-over-h2-tls-alpn' },
        @{ id = 'grpc.h2.client-cancellation'; profile = 'grpc-h2-diagnostic'; duration = '10'; repetition = '1'; variant = 'grpc-over-h2-tls-alpn' },
        @{ id = 'grpc.h2.unary.echo-new-channel'; profile = 'grpc-h2-channel-churn'; duration = '30'; repetition = '3'; variant = 'grpc-over-h2-tls-new-channel' }
    )
    foreach ($scenario in $scenarios) {
        $env:PLAB_EXECUTOR_ID = 'go-grpc-h2-executor'
        $env:PLAB_EXECUTOR_VERSION = '0.4.1'
        $env:PLAB_LOAD_GENERATOR_ID = 'go-x-net-http2-grpc-load'
        $env:PLAB_LOAD_GENERATOR_VERSION = '0.4.1'
        $env:PLAB_SCENARIO_ID = $scenario.id
        $env:PLAB_LOAD_PROFILE_ID = $scenario.profile
        $env:PLAB_PROTOCOL = 'h2'
        $env:PLAB_PROTOCOL_VARIANT = $scenario.variant
        $env:PLAB_CONNECTIONS = '1'
        $env:PLAB_CONCURRENCY = '1'
        $env:PLAB_STREAMS_PER_CONNECTION = '1'
        $env:PLAB_DURATION_SECONDS = $scenario.duration
        $env:PLAB_WARMUP_SECONDS = '0'
        $env:PLAB_REPETITION = $scenario.repetition
        $artifactRoot = Join-Path $OutputRoot ("evidence/" + $scenario.id.Replace('.', '-'))
        New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
        $stdout = Join-Path $artifactRoot 'load.stdout.log'
        $stderr = Join-Path $artifactRoot 'load.stderr.log'
        $process = Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/go-grpc-h2-executor.exe') `
            -WorkingDirectory $executorRoot -ArgumentList @('--target-url', "https://127.0.0.1:$Port", '--output-dir', $artifactRoot) `
            -Wait -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
        if ($process.ExitCode -ne 0) { throw "$($scenario.id) executor exited $($process.ExitCode)." }
        $result = Get-Content -LiteralPath (Join-Path $artifactRoot 'result.json') -Raw | ConvertFrom-Json
        if ($result.passed -ne $true -or $result.scenarioId -ne $scenario.id -or $result.metrics.failedOperations -ne 0 -or $result.metrics.timedOutOperations -ne 0) {
            throw "$($scenario.id) extracted-package evidence failed the common outcome gate."
        }
        if ($result.executorId -ne 'go-grpc-h2-executor' -or $result.executorVersion -ne '0.4.1' -or
            $result.loadGeneratorId -ne 'go-x-net-http2-grpc-load' -or $result.loadGeneratorVersion -ne '0.4.1') {
            throw "$($scenario.id) substituted the executor or load-generator identity."
        }
        if ($result.protocol.tlsVersion -ne 'TLS1.3' -or $result.protocol.alpn -ne 'h2' -or
            $result.protocol.httpVersion -ne 'HTTP/2.0' -or $result.protocol.fallbackDetected -ne $false) {
            throw "$($scenario.id) failed exact TLS/H2 protocol proof."
        }
        foreach ($artifact in @('validation.json', 'protocol-proof.json', 'grpc-summary.json', 'tls-negotiation.json',
                'result.json', 'executor-identity.json', 'load-generator-identity.json', 'grpc-request-frame.bin',
                'grpc-response-frame.bin', 'tls-peer-certificate.der')) {
            if (-not (Test-Path -LiteralPath (Join-Path $artifactRoot $artifact) -PathType Leaf)) {
                throw "$($scenario.id) did not preserve required artifact $artifact."
            }
        }

        switch ($scenario.id) {
            'grpc.h2.trailers-only-status' {
                if ($result.metrics.completedOperations -ne 1 -or $result.metrics.deadlineExceededOperations -ne 0 -or
                    $result.metrics.cancelledOperations -ne 0 -or $result.response.httpStatus -ne 200 -or
                    $result.response.contentType -ne 'application/grpc+proto' -or $result.response.grpcStatus -ne '3' -or
                    $result.response.grpcMessage -ne 'plab invalid fixture' -or $result.response.trailersOnly -ne $true -or
                    $result.response.trailersPresent -ne $true -or $result.response.noResponseData -ne $true -or
                    $result.response.expectedTerminalOutcome -ne $true) {
                    throw 'grpc.h2.trailers-only-status failed exact terminal-status proof.'
                }
            }
            'grpc.h2.deadline-exceeded' {
                if ($result.metrics.completedOperations -ne 1 -or $result.metrics.deadlineExceededOperations -ne 1 -or
                    $result.metrics.cancelledOperations -ne 0 -or $result.response.grpcStatus -ne '4' -or
                    $result.response.deadlineFired -ne $true -or $result.response.noResponseData -ne $true -or
                    $result.response.expectedTerminalOutcome -ne $true) {
                    throw 'grpc.h2.deadline-exceeded failed exact deadline proof.'
                }
            }
            'grpc.h2.client-cancellation' {
                if ($result.metrics.completedOperations -ne 1 -or $result.metrics.deadlineExceededOperations -ne 0 -or
                    $result.metrics.cancelledOperations -ne 1 -or $result.response.grpcStatus -ne '1' -or
                    $result.response.readyInitialMetadata -ne $true -or $result.response.clientCancelTriggered -ne $true -or
                    $result.response.noResponseData -ne $true -or $result.response.expectedTerminalOutcome -ne $true) {
                    throw 'grpc.h2.client-cancellation failed exact client-cancellation proof.'
                }
            }
            'grpc.h2.unary.echo-new-channel' {
                if ($result.metrics.completedOperations -ne 10 -or $result.metrics.deadlineExceededOperations -ne 0 -or
                    $result.metrics.cancelledOperations -ne 0 -or $result.protocol.requested -ne 'grpc-over-h2-new-channel' -or
                    $result.protocol.observed -ne 'grpc-over-h2-new-channel' -or $result.channel.channelsCreated -ne 10 -or
                    $result.channel.connectionsEstablished -ne 10 -or $result.channel.preEstablished -ne $false -or
                    $result.channel.reusedForMeasuredOperation -ne $false -or $result.channel.newChannelPerOperation -ne $true -or
                    $result.channel.channelDisposedAfterEachOperation -ne $true -or $result.response.grpcStatus -ne '0') {
                    throw 'grpc.h2.unary.echo-new-channel failed exact channel-churn proof.'
                }
            }
        }
        Write-Host "$($scenario.id): completed=$($result.metrics.completedOperations) failed=0 timedOut=0"
    }

    $env:PLAB_SCENARIO_ID = 'grpc.h2.not-a-contract-identity'
    $unknownRoot = Join-Path $OutputRoot 'evidence/unknown-identity'
    New-Item -ItemType Directory -Force -Path $unknownRoot | Out-Null
    $unknown = Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/go-grpc-h2-executor.exe') `
        -WorkingDirectory $executorRoot -ArgumentList @('--target-url', "https://127.0.0.1:$Port", '--output-dir', $unknownRoot) `
        -Wait -PassThru -WindowStyle Hidden -RedirectStandardOutput (Join-Path $unknownRoot 'stdout.log') `
        -RedirectStandardError (Join-Path $unknownRoot 'stderr.log')
    if ($unknown.ExitCode -ne 2) { throw "Unknown gRPC scenario must fail closed with exit 2; observed $($unknown.ExitCode)." }
}
finally {
    Stop-Process -Id $target.Id -Force -ErrorAction SilentlyContinue
    Wait-Process -Id $target.Id -ErrorAction SilentlyContinue
}
