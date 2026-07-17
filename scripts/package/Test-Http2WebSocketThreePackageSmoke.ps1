[CmdletBinding()]
param(
    [ValidateSet('win-x64', 'linux-x64')]
    [string]$RuntimeIdentifier = 'win-x64',
    [string]$PackageRoot = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/http2-websocket-packages'),
    [string]$ArtifactRoot = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path "artifacts/http2-websocket-extracted-smoke-$RuntimeIdentifier")
)

$ErrorActionPreference = 'Stop'
$PackageRoot = [IO.Path]::GetFullPath($PackageRoot)
$ArtifactRoot = [IO.Path]::GetFullPath($ArtifactRoot)
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$artifactsRoot = [IO.Path]::GetFullPath((Join-Path $repoRoot 'artifacts'))
if (-not $ArtifactRoot.StartsWith($artifactsRoot, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Smoke artifacts must remain under this worktree artifacts directory.'
}

function Resolve-One([string]$Pattern) {
    $matches = @(Get-ChildItem -LiteralPath $PackageRoot -Filter $Pattern -File)
    if ($matches.Count -ne 1) { throw "Expected one $Pattern package, observed $($matches.Count)." }
    return $matches[0].FullName
}

function Expand-One([string]$Archive, [string]$Destination) {
    New-Item -ItemType Directory -Force $Destination | Out-Null
    [IO.Compression.ZipFile]::ExtractToDirectory($Archive, $Destination)
    $manifest = Get-Content (Join-Path $Destination 'protocol-lab-package.json') -Raw | ConvertFrom-Json
    if ($manifest.schemaVersion -ne 'protocol-lab-package-v2') { throw "$Archive is not package-v2" }
    return $manifest
}

function Convert-ToWslPath([string]$Path) {
    $converted = (& wsl.exe --exec wslpath -a -u ([IO.Path]::GetFullPath($Path))).Trim()
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($converted)) { throw "Could not translate $Path to WSL." }
    return $converted
}

if (Test-Path $ArtifactRoot) { Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force }
New-Item -ItemType Directory -Force $ArtifactRoot | Out-Null

$scenarioArchive = Resolve-One 'org.protocol-lab.components.scenario.http2-websocket-performance.0.1.2.plabpkg'
$executorArchive = Resolve-One "org.protocol-lab.components.executor.go-http2-websocket-executor.0.2.0.$RuntimeIdentifier.plabpkg"
$targetArchive = Resolve-One "org.protocol-lab.components.implementation.kestrel-http2-websocket.0.1.2.$RuntimeIdentifier.plabpkg"
$scenarioRoot = Join-Path $ArtifactRoot 'scenario'
$executorRoot = Join-Path $ArtifactRoot 'executor'
$targetRoot = Join-Path $ArtifactRoot 'target'
$scenarioManifest = Expand-One $scenarioArchive $scenarioRoot
$executorManifest = Expand-One $executorArchive $executorRoot
$targetManifest = Expand-One $targetArchive $targetRoot

$authority = Get-Content (Join-Path $scenarioRoot 'authority-lock.json') -Raw | ConvertFrom-Json
if ($authority.commit -ne '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574') { throw 'authority commit mismatch' }
if ($scenarioManifest.packageVersion -ne '0.1.2' -or $executorManifest.packageVersion -ne '0.2.0' -or $targetManifest.packageVersion -ne '0.1.2') { throw 'immutable package version mismatch' }
if ($scenarioManifest.providedScenarios.Count -ne 6 -or $executorManifest.providedTestExecutors[0].scenarios.Count -ne 6 -or $targetManifest.providedImplementations[0].scenarios.Count -ne 6) { throw 'six-ID package claim mismatch' }
$loadProfileEntries = @($scenarioManifest.entryManifests | Where-Object { $_ -like 'load-profiles/*' } | Sort-Object)
if (($loadProfileEntries -join ',') -ne 'load-profiles/diagnostic.yaml,load-profiles/websocket-comparison.yaml,load-profiles/websocket-smoke.yaml') { throw 'scenario load-profile entry mismatch' }
foreach ($license in @('golang-x-net-LICENSE.txt', 'golang-x-text-LICENSE.txt')) {
    if (-not (Test-Path (Join-Path $executorRoot "third-party/$license"))) { throw "missing $license" }
}
foreach ($notice in @('LICENSE.txt', 'ThirdPartyNotices.txt')) {
    if (-not (Test-Path (Join-Path $targetRoot "third-party/$notice"))) { throw "missing $notice" }
}

$port = if ($RuntimeIdentifier -eq 'win-x64') { 18451 } else { 18452 }
$targetOut = Join-Path $ArtifactRoot 'target.stdout.log'
$targetErr = Join-Path $ArtifactRoot 'target.stderr.log'
$target = $null
$linuxTargetPid = $null
$environmentNames = @(
    'PLAB_LISTEN_ADDRESS', 'PLAB_SCENARIO_ID', 'PLAB_TLS_CERTIFICATE_PATH', 'PLAB_TLS_PRIVATE_KEY_PATH',
    'PLAB_TARGET_BASE_URL', 'PLAB_ARTIFACT_DIR', 'PLAB_TLS_ROOT_CERTIFICATE_PATH', 'PLAB_EXECUTOR_ID',
    'PLAB_EXECUTOR_VERSION', 'PLAB_LOAD_GENERATOR_ID', 'PLAB_LOAD_GENERATOR_VERSION', 'PLAB_PROTOCOL',
    'PLAB_PROTOCOL_VARIANT', 'PLAB_LOAD_PROFILE_ID', 'PLAB_CONNECTIONS', 'PLAB_CONCURRENCY',
    'PLAB_STREAMS_PER_CONNECTION', 'PLAB_DURATION_SECONDS', 'PLAB_WARMUP_SECONDS', 'PLAB_COOLDOWN_SECONDS',
    'PLAB_REPETITION', 'PLAB_REQUEST_TIMEOUT_SECONDS'
)
$saved = @{}
foreach ($name in $environmentNames) { $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }

try {
    if ($RuntimeIdentifier -eq 'win-x64') {
        $targetBinary = Join-Path $targetRoot 'bin/win-x64/KestrelHttp2WebSocket.exe'
        $executorBinary = Join-Path $executorRoot 'bin/win-x64/go-http2-websocket-executor.exe'
        $env:PLAB_LISTEN_ADDRESS = "127.0.0.1:$port"
        $env:PLAB_SCENARIO_ID = 'http2.websocket.rfc8441.extended-connect'
        $env:PLAB_TLS_CERTIFICATE_PATH = Join-Path $targetRoot 'certs/leaf.pem'
        $env:PLAB_TLS_PRIVATE_KEY_PATH = Join-Path $targetRoot 'certs/leaf-key.pem'
        $target = Start-Process $targetBinary -RedirectStandardOutput $targetOut -RedirectStandardError $targetErr -WindowStyle Hidden -PassThru
    }
    else {
        if (-not (Get-Command wsl.exe -ErrorAction SilentlyContinue)) { throw 'linux-x64 extracted smoke requires WSL.' }
        $targetBinary = Convert-ToWslPath (Join-Path $targetRoot 'bin/linux-x64/KestrelHttp2WebSocket')
        $executorBinary = Convert-ToWslPath (Join-Path $executorRoot 'bin/linux-x64/go-http2-websocket-executor')
        $targetCertificate = Convert-ToWslPath (Join-Path $targetRoot 'certs/leaf.pem')
        $targetKey = Convert-ToWslPath (Join-Path $targetRoot 'certs/leaf-key.pem')
        & wsl.exe --exec chmod +x $targetBinary $executorBinary
        if ($LASTEXITCODE -ne 0) { throw 'chmod for extracted Linux executables failed.' }
        $targetArguments = @('--exec', 'env', "PLAB_LISTEN_ADDRESS=127.0.0.1:$port", 'PLAB_SCENARIO_ID=http2.websocket.rfc8441.extended-connect', "PLAB_TLS_CERTIFICATE_PATH=$targetCertificate", "PLAB_TLS_PRIVATE_KEY_PATH=$targetKey", $targetBinary)
        $target = Start-Process wsl.exe -ArgumentList $targetArguments -RedirectStandardOutput $targetOut -RedirectStandardError $targetErr -WindowStyle Hidden -PassThru
    }

    $ready = $false
    for ($index = 0; $index -lt 600; $index++) {
        if ((Test-Path $targetOut) -and ((Get-Content $targetOut -Raw) -match '"eventName":"ready"')) { $ready = $true; break }
        Start-Sleep -Milliseconds 100
    }
    if (-not $ready) { throw "target not ready: $(Get-Content $targetErr -Raw)" }
    if ($RuntimeIdentifier -eq 'linux-x64') {
        $targetPids = @((& wsl.exe --exec pgrep -f -- $targetBinary) -split "`n" | Where-Object { $_ -match '^\d+$' })
        if ($targetPids.Count -ne 1) { throw "Expected one extracted Linux target process, observed $($targetPids.Count)." }
        $linuxTargetPid = $targetPids[0]
    }
    $targetReady = ((Get-Content $targetOut) | Where-Object { $_ -like '*"eventName":"ready"*' } | Select-Object -First 1) | ConvertFrom-Json
    if ($targetReady.implementationId -ne 'kestrel-http2-websocket' -or $targetReady.implementationVersion -ne '0.1.2' -or $targetReady.protocolVersion -ne 'HTTP/2' -or $targetReady.settingsEnableConnectProtocol -ne 1 -or $targetReady.alpn -ne 'h2' -or $targetReady.certificateDerSha256 -ne 'fe996190f39355e3cfc201cbb7e2cba962a701b94ed08ff49e68e830216d0109' -or $targetReady.certificateSpkiSha256 -ne 'c2440fbe955033f341ca625c1804e21b50066d952ab24a4b53007dc1cfbf410c') { throw 'target readiness mismatch' }

    $commonEnvironment = [ordered]@{
        PLAB_TARGET_BASE_URL = "https://127.0.0.1:$port"
        PLAB_TLS_ROOT_CERTIFICATE_PATH = if ($RuntimeIdentifier -eq 'win-x64') { Join-Path $executorRoot 'certs/root.pem' } else { Convert-ToWslPath (Join-Path $executorRoot 'certs/root.pem') }
        PLAB_EXECUTOR_ID = 'go-http2-websocket-executor'
        PLAB_EXECUTOR_VERSION = '0.2.0'
        PLAB_LOAD_GENERATOR_ID = 'go-x-net-http2-websocket-load'
        PLAB_LOAD_GENERATOR_VERSION = '0.2.0'
        PLAB_PROTOCOL = 'h2'
        PLAB_PROTOCOL_VARIANT = 'websocket-h2-extended-connect'
        PLAB_REPETITION = '1'
    }

    $ids = @(
        'http2.websocket.rfc8441.extended-connect', 'http2.websocket.rfc8441.control-frames',
        'http2.websocket.rfc8441.text-echo', 'http2.websocket.rfc8441.binary-echo',
        'http2.websocket.rfc8441.close', 'http2.websocket.rfc8441.multi-message-text-echo'
    )
    $outcomes = @()
    foreach ($id in $ids) {
        $multi = $id -eq 'http2.websocket.rfc8441.multi-message-text-echo'
        $profile = if ($multi) { 'diagnostic' } else { 'websocket-smoke' }
        $cellRoot = Join-Path $ArtifactRoot ($id -replace '\.', '-')
        New-Item -ItemType Directory -Force $cellRoot | Out-Null
        $cellEnvironment = [ordered]@{}
        foreach ($pair in $commonEnvironment.GetEnumerator()) { $cellEnvironment[$pair.Key] = $pair.Value }
        $cellEnvironment.PLAB_SCENARIO_ID = $id
        $cellEnvironment.PLAB_ARTIFACT_DIR = if ($RuntimeIdentifier -eq 'win-x64') { $cellRoot } else { Convert-ToWslPath $cellRoot }
        $cellEnvironment.PLAB_LOAD_PROFILE_ID = $profile
        $cellEnvironment.PLAB_CONNECTIONS = '1'
        $cellEnvironment.PLAB_CONCURRENCY = if ($multi) { '8' } else { '1' }
        $cellEnvironment.PLAB_STREAMS_PER_CONNECTION = if ($multi) { '8' } else { '1' }
        $cellEnvironment.PLAB_DURATION_SECONDS = if ($multi) { '10' } else { '5' }
        $cellEnvironment.PLAB_WARMUP_SECONDS = '1'
        $cellEnvironment.PLAB_COOLDOWN_SECONDS = if ($multi) { '1' } else { '0' }
        $cellEnvironment.PLAB_REQUEST_TIMEOUT_SECONDS = if ($multi) { '10' } else { '5' }
        $stdout = Join-Path $cellRoot 'load.stdout.log'
        $stderr = Join-Path $cellRoot 'load.stderr.log'

        if ($RuntimeIdentifier -eq 'win-x64') {
            foreach ($pair in $cellEnvironment.GetEnumerator()) { [Environment]::SetEnvironmentVariable($pair.Key, [string]$pair.Value, 'Process') }
            $run = Start-Process $executorBinary -RedirectStandardOutput $stdout -RedirectStandardError $stderr -WindowStyle Hidden -Wait -PassThru
            $exitCode = $run.ExitCode
        }
        else {
            $arguments = @('--exec', 'env')
            foreach ($pair in $cellEnvironment.GetEnumerator()) { $arguments += "$($pair.Key)=$($pair.Value)" }
            $arguments += $executorBinary
            & wsl.exe @arguments 1> $stdout 2> $stderr
            $exitCode = $LASTEXITCODE
        }
        if ($exitCode -ne 0) { throw "$RuntimeIdentifier $id failed: $(Get-Content $stderr -Raw)" }

        $required = @('validation.json', 'protocol-proof.json', 'websocket-summary.json', 'payload-hash.json', 'http2-frame-summary.json', 'frame-summary.json', 'result.json', 'http2-websocket-executor-result.json', 'websocket-load-summary.json', 'websocket-warmup-summary.json', 'executor-identity.json', 'load-generator-identity.json', 'load.stdout.log', 'load.stderr.log')
        if ($multi) { $required += 'http2-websocket-topology.json' }
        foreach ($name in $required) { if (-not (Test-Path (Join-Path $cellRoot $name))) { throw "$id missing $name" } }

        $result = Get-Content (Join-Path $cellRoot 'result.json') -Raw | ConvertFrom-Json
        $proof = $result.protocolProof
        if ($result.authorityCommit -ne $authority.commit -or $result.executor.id -ne 'go-http2-websocket-executor' -or $result.executor.version -ne '0.2.0' -or $result.loadGenerator.id -ne 'go-x-net-http2-websocket-load' -or $result.loadGenerator.version -ne '0.2.0') { throw "$id identity gate failed" }
        if ($result.requestedLoad.profileId -ne $profile -or $result.requestedLoad.connections -ne 1 -or $result.requestedLoad.repetitions -ne 1 -or $result.requestedLoad.warmupSeconds -ne 1) { throw "$id requested profile gate failed" }
        if ($proof.tlsVersion -ne 'TLS 1.3' -or $proof.alpn -ne 'h2' -or $proof.didResume -or $proof.certificateDerSha256 -ne 'fe996190f39355e3cfc201cbb7e2cba962a701b94ed08ff49e68e830216d0109' -or $proof.certificateSpkiSha256 -ne 'c2440fbe955033f341ca625c1804e21b50066d952ab24a4b53007dc1cfbf410c' -or $proof.settingsEnableConnectProtocol -ne 1 -or $proof.requestPseudoHeaders.':method' -ne 'CONNECT' -or $proof.requestPseudoHeaders.':protocol' -ne 'websocket' -or $proof.requestPseudoHeaders.':scheme' -ne 'https' -or $proof.requestPseudoHeaders.':authority' -ne 'websocket.plab.test' -or $proof.requestPseudoHeaders.':path' -ne '/websocket' -or $proof.responseStatus -ne 200 -or $proof.secWebSocketAcceptPresent -or $proof.secWebSocketKeyPresent -or $proof.subprotocolPresent -or $proof.extensionsPresent -or $proof.prohibitedRequestHeadersPresent -or -not $proof.clientMaskObserved -or $proof.closeSent -ne 1000 -or $proof.closeReceived -ne 1000 -or -not $proof.cleanCompletion) { throw "$id common proof failed" }
        if ($result.metrics.completedOperations -le 0 -or $result.metrics.failedOperations -ne 0 -or $result.metrics.timedOutOperations -ne 0 -or $result.metrics.messagesPerSecond -lt 0 -or $result.metrics.bytesPerSecond -lt 0) { throw "$id outcomes/metrics failed" }
        if ($multi) {
            $topology = Get-Content (Join-Path $cellRoot 'http2-websocket-topology.json') -Raw | ConvertFrom-Json
            if ($result.requestedLoad.durationSeconds -ne 10 -or $result.requestedLoad.cooldownSeconds -ne 1 -or $result.requestedLoad.concurrency -ne 8 -or $result.requestedLoad.streamsPerConnection -ne 8 -or $result.requestedLoad.operationTimeoutSeconds -ne 10 -or $result.effectiveLoad.configuredCapacity.connections -ne 1 -or $result.effectiveLoad.observed.activeConnections -ne 1 -or $result.effectiveLoad.observed.activeStreams -ne 8 -or $result.effectiveLoad.observed.effectiveConcurrency -ne 8 -or $proof.messageCount -ne 100 -or $proof.messageBytes -ne 12 -or $proof.payloadSha256 -ne '504585b0bb4fd77012ea2575efbcdb58f4c33e6b543e9567a65896d213720c29' -or -not $proof.strictOrdering -or -not $proof.connectionReused -or $topology.connectionCount -ne 1 -or $topology.authenticatedTlsConnections -ne 1 -or $topology.observedActiveStreams -ne 8 -or $topology.effectiveConcurrency -ne 8 -or (@($topology.streamIds) -join ',') -ne '1,3,5,7,9,11,13,15') { throw 'multi-message topology/order gate failed' }
            $frame = Get-Content (Join-Path $cellRoot 'frame-summary.json') -Raw | ConvertFrom-Json
            if ($frame.serverMaskedFrames -ne 0 -or -not $frame.strictPerStreamOrdering -or $frame.clientMaskedFrames -le 800 -or $frame.uniqueClientMaskKeys -le 1) { throw 'multi-message masking/frame gate failed' }
        }
        elseif ($result.requestedLoad.durationSeconds -ne 5 -or $result.requestedLoad.cooldownSeconds -ne 0 -or $result.requestedLoad.concurrency -ne 1 -or $result.requestedLoad.streamsPerConnection -ne 1 -or $result.requestedLoad.operationTimeoutSeconds -ne 5 -or $result.effectiveLoad.observed.activeConnections -ne 1 -or $result.effectiveLoad.observed.activeStreams -ne 1 -or $result.effectiveLoad.observed.effectiveConcurrency -ne 1) { throw "$id core profile gate failed" }
        if ($id -eq 'http2.websocket.rfc8441.control-frames' -and (-not $proof.pingSent -or -not $proof.pongReceived)) { throw 'ping/pong proof failed' }
        if ($id -eq 'http2.websocket.rfc8441.text-echo' -and $proof.payloadSha256 -ne '504585b0bb4fd77012ea2575efbcdb58f4c33e6b543e9567a65896d213720c29') { throw 'text hash failed' }
        if ($id -eq 'http2.websocket.rfc8441.binary-echo' -and ($proof.messageBytes -ne 1024 -or $proof.payloadSha256 -ne '9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a')) { throw 'binary hash failed' }
        $outcomes += [pscustomobject]@{ scenarioId = $id; loadProfileId = $profile; completed = $result.metrics.completedOperations; messages = $result.metrics.completedMessages; failed = 0; timedOut = 0; observedConnections = $result.metrics.observedActiveConnections; observedStreams = $result.metrics.observedActiveStreams; effectiveConcurrency = $result.metrics.effectiveConcurrency }
    }

    $unsupported = @(
        'http1.websocket.rfc6455.cleartext.upgrade', 'http1.websocket.rfc6455.cleartext.control-frames', 'http1.websocket.rfc6455.cleartext.text-echo', 'http1.websocket.rfc6455.cleartext.binary-echo', 'http1.websocket.rfc6455.cleartext.close',
        'http1.websocket.rfc6455.tls.upgrade', 'http1.websocket.rfc6455.tls.control-frames', 'http1.websocket.rfc6455.tls.text-echo', 'http1.websocket.rfc6455.tls.binary-echo', 'http1.websocket.rfc6455.tls.close', 'http1.websocket.rfc6455.tls.subprotocol-text-echo', 'http1.websocket.rfc6455.tls.permessage-deflate-binary-echo',
        'http3.websocket.rfc9220.extended-connect', 'http3.websocket.rfc9220.control-frames', 'http3.websocket.rfc9220.text-echo', 'http3.websocket.rfc9220.binary-echo', 'http3.websocket.rfc9220.close', 'http3.websocket.rfc9220.fragmented-binary-echo'
    )
    foreach ($id in $unsupported) {
        $cellRoot = Join-Path $ArtifactRoot ('unsupported-' + ($id -replace '\.', '-'))
        New-Item -ItemType Directory -Force $cellRoot | Out-Null
        if ($RuntimeIdentifier -eq 'win-x64') {
            $env:PLAB_SCENARIO_ID = $id; $env:PLAB_ARTIFACT_DIR = $cellRoot
            $run = Start-Process $executorBinary -RedirectStandardOutput (Join-Path $cellRoot 'load.stdout.log') -RedirectStandardError (Join-Path $cellRoot 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
            $exitCode = $run.ExitCode
        }
        else {
            $arguments = @('--exec', 'env', "PLAB_SCENARIO_ID=$id", "PLAB_ARTIFACT_DIR=$(Convert-ToWslPath $cellRoot)", 'PLAB_EXECUTOR_ID=go-http2-websocket-executor', 'PLAB_EXECUTOR_VERSION=0.2.0', 'PLAB_LOAD_GENERATOR_ID=go-x-net-http2-websocket-load', 'PLAB_LOAD_GENERATOR_VERSION=0.2.0', 'PLAB_PROTOCOL=h2', 'PLAB_PROTOCOL_VARIANT=websocket-h2-extended-connect', $executorBinary)
            & wsl.exe @arguments 1> (Join-Path $cellRoot 'load.stdout.log') 2> (Join-Path $cellRoot 'load.stderr.log')
            $exitCode = $LASTEXITCODE
        }
        if ($exitCode -ne 3) { throw "$id did not fail unsupported" }
        $unsupportedResult = Get-Content (Join-Path $cellRoot 'result.json') -Raw | ConvertFrom-Json
        if ($unsupportedResult.status -ne 'unsupported' -or $unsupportedResult.scenarioId -ne $id) { throw "$id unsupported result mismatch" }
    }

    $unknownRoot = Join-Path $ArtifactRoot 'unknown'
    New-Item -ItemType Directory -Force $unknownRoot | Out-Null
    if ($RuntimeIdentifier -eq 'win-x64') {
        $env:PLAB_SCENARIO_ID = 'http2.websocket.unknown'; $env:PLAB_ARTIFACT_DIR = $unknownRoot
        $unknownRun = Start-Process $executorBinary -RedirectStandardOutput (Join-Path $unknownRoot 'load.stdout.log') -RedirectStandardError (Join-Path $unknownRoot 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
        $unknownExitCode = $unknownRun.ExitCode
    }
    else {
        & wsl.exe --exec env 'PLAB_SCENARIO_ID=http2.websocket.unknown' "PLAB_ARTIFACT_DIR=$(Convert-ToWslPath $unknownRoot)" $executorBinary 1> (Join-Path $unknownRoot 'load.stdout.log') 2> (Join-Path $unknownRoot 'load.stderr.log')
        $unknownExitCode = $LASTEXITCODE
    }
    if ($unknownExitCode -ne 2) { throw 'unknown ID did not fail closed with exit 2' }

    $targetProof = Get-Content $targetOut -Raw
    $accepted = ([regex]::Matches($targetProof, '"eventName":"extended-connect-accepted"')).Count
    $closed = ([regex]::Matches($targetProof, '"eventName":"websocket-clean-close"')).Count
    if ($accepted -lt 13 -or $closed -lt 13 -or ([regex]::Matches($targetProof, '"clientMaskObserved":true')).Count -lt 13) { throw 'target raw-frame masking/close/topology proof incomplete' }

    [pscustomobject]@{
        runtimeIdentifier = $RuntimeIdentifier
        authorityCommit = $authority.commit
        scenarioPackageSha256 = (Get-FileHash $scenarioArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        executorPackageSha256 = (Get-FileHash $executorArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        targetPackageSha256 = (Get-FileHash $targetArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        targetAcceptedStreams = $accepted
        targetCleanCloses = $closed
        outcomes = $outcomes
        unsupportedCount = $unsupported.Count
        evidenceRoot = $ArtifactRoot
    } | ConvertTo-Json -Depth 8
}
finally {
    if ($null -ne $linuxTargetPid) {
        & wsl.exe --exec kill -TERM $linuxTargetPid 2>$null
    }
    if ($null -ne $target -and -not $target.HasExited) { Stop-Process -Id $target.Id -Force }
    foreach ($name in $environmentNames) { [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process') }
}
