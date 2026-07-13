[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/http1-websocket-tls-packages'),
    [string]$SmokeRoot = (Join-Path $Root 'artifacts/http1-websocket-tls-three-package-smoke'),
    [string[]]$ScenarioIds = @('http1.websocket.rfc6455.tls.upgrade','http1.websocket.rfc6455.tls.control-frames','http1.websocket.rfc6455.tls.text-echo','http1.websocket.rfc6455.tls.binary-echo','http1.websocket.rfc6455.tls.close'),
    [switch]$SkipBuild
)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression.FileSystem
if (-not $SkipBuild) {
    & (Join-Path $PSScriptRoot 'Build-Http1WebSocketTlsScenarioPackage.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-GoHttp1WebSocketTlsExecutorPackage.ps1') win-x64 -Root $Root -OutputRoot $OutputRoot -AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-GoHttp1WebSocketTlsImplementationPackage.ps1') win-x64 -Root $Root -OutputRoot $OutputRoot -AllowDirtySource
}
$scenarioPackage = Get-ChildItem $OutputRoot -File -Filter 'org.protocol-lab.components.scenario.http1-websocket-tls-performance.0.2.0.plabpkg' | Select-Object -First 1
$executorPackage = Get-ChildItem $OutputRoot -File -Filter 'org.protocol-lab.components.executor.go-http1-websocket-tls-executor.0.2.0.win-x64.plabpkg' | Select-Object -First 1
$targetPackage = Get-ChildItem $OutputRoot -File -Filter 'org.protocol-lab.components.implementation.go-http1-websocket-tls.0.2.0.win-x64.plabpkg' | Select-Object -First 1
foreach ($package in @($scenarioPackage,$executorPackage,$targetPackage)) { if ($null -eq $package) { throw 'Expected package artifact not found.' } }
Remove-Item -LiteralPath $SmokeRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $SmokeRoot | Out-Null
$scenarioRoot=Join-Path $SmokeRoot 'scenario'; $executorRoot=Join-Path $SmokeRoot 'executor'; $targetRoot=Join-Path $SmokeRoot 'target'
[IO.Compression.ZipFile]::ExtractToDirectory($scenarioPackage.FullName,$scenarioRoot)
[IO.Compression.ZipFile]::ExtractToDirectory($executorPackage.FullName,$executorRoot)
[IO.Compression.ZipFile]::ExtractToDirectory($targetPackage.FullName,$targetRoot)
& pwsh -NoLogo -NoProfile -File (Join-Path $scenarioRoot 'validate.ps1')
if ($LASTEXITCODE -ne 0) { throw 'Extracted scenario package validation failed.' }
$targetStdout=Join-Path $SmokeRoot 'target.stdout.log'; $targetStderr=Join-Path $SmokeRoot 'target.stderr.log'
$targetBinary=Join-Path $targetRoot 'bin/win-x64/go-http1-websocket-tls.exe'; $executorBinary=Join-Path $executorRoot 'bin/win-x64/go-http1-websocket-tls-executor.exe'
$saved = @{}
$variables = @('PLAB_TARGET_PORT','PLAB_HTTP1_WEBSOCKET_TLS_LISTEN','PLAB_TLS_CERTIFICATE_PATH','PLAB_TLS_PRIVATE_KEY_PATH','PLAB_TLS_ROOT_CERTIFICATE_PATH','PLAB_EXECUTOR_ID','PLAB_EXECUTOR_VERSION','PLAB_LOAD_GENERATOR_ID','PLAB_LOAD_GENERATOR_VERSION','PLAB_PROTOCOL','PLAB_PROTOCOL_VARIANT','PLAB_LOAD_PROFILE_ID','PLAB_CONNECTIONS','PLAB_CONCURRENCY','PLAB_DURATION_SECONDS','PLAB_WARMUP_SECONDS','PLAB_REPETITION','PLAB_OPERATION_TIMEOUT_MILLISECONDS','PLAB_SCENARIO_ID')
foreach ($name in $variables) { $saved[$name]=[Environment]::GetEnvironmentVariable($name,'Process') }
$process=$null
try {
    $smokePort = Get-Random -Minimum 20000 -Maximum 40000
    $env:PLAB_TARGET_PORT=[string]$smokePort
    $env:PLAB_HTTP1_WEBSOCKET_TLS_LISTEN=("[::1]:{0}" -f $smokePort)
    $env:PLAB_TLS_CERTIFICATE_PATH=Join-Path $targetRoot 'certs/leaf.pem'
    $env:PLAB_TLS_PRIVATE_KEY_PATH=Join-Path $targetRoot 'certs/leaf-key.pem'
    $env:PLAB_TLS_ROOT_CERTIFICATE_PATH=Join-Path $executorRoot 'certs/root.pem'
    $process=Start-Process -FilePath $targetBinary -WorkingDirectory $targetRoot -RedirectStandardOutput $targetStdout -RedirectStandardError $targetStderr -WindowStyle Hidden -PassThru
    $deadline=(Get-Date).AddSeconds(10)
    while ((Get-Date) -lt $deadline) {
        if ($process.HasExited) { throw "Target exited early with $($process.ExitCode)." }
        if ((Test-Path $targetStdout) -and ((Get-Content $targetStdout -Raw) -match '"status":"ready"')) { break }
        Start-Sleep -Milliseconds 100
    }
    if (-not (Test-Path $targetStdout) -or -not ((Get-Content $targetStdout -Raw) -match '"status":"ready"')) { throw 'Target readiness evidence was not observed.' }
    $ready = Get-Content $targetStdout -First 1 | ConvertFrom-Json
    if ($ready.implementationId -ne 'go-http1-websocket-tls' -or $ready.version -ne '0.2.0' -or $ready.protocol -ne 'h1' -or $ready.protocolVersion -ne 'HTTP/1.1' -or $ready.protocolVariant -ne 'websocket-h1-tls1.3-upgrade' -or $ready.transportSecurity -ne 'tls' -or $ready.tlsVersion -ne 'TLS1.3' -or $ready.alpn -ne 'http/1.1') { throw 'Target readiness identity or exact protocol proof mismatch.' }
    $env:PLAB_EXECUTOR_ID='go-http1-websocket-tls-executor'; $env:PLAB_EXECUTOR_VERSION='0.2.0'
    $env:PLAB_LOAD_GENERATOR_ID='go-http1-websocket-tls-load'; $env:PLAB_LOAD_GENERATOR_VERSION='0.2.0'
    $env:PLAB_PROTOCOL='h1'; $env:PLAB_PROTOCOL_VARIANT='websocket-h1-tls1.3-upgrade'; $env:PLAB_LOAD_PROFILE_ID='websocket-smoke'
    $env:PLAB_CONNECTIONS='1'; $env:PLAB_CONCURRENCY='1'; $env:PLAB_DURATION_SECONDS='5'; $env:PLAB_WARMUP_SECONDS='1'; $env:PLAB_REPETITION='1'; $env:PLAB_OPERATION_TIMEOUT_MILLISECONDS='5000'
    $results=@()
    foreach ($scenarioId in $ScenarioIds) {
        $env:PLAB_SCENARIO_ID=$scenarioId
        $artifactRoot=Join-Path $SmokeRoot ($scenarioId -replace '[^a-zA-Z0-9.-]','-')
        New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
        $stdout=Join-Path $artifactRoot 'load.stdout.log'; $stderr=Join-Path $artifactRoot 'load.stderr.log'
        $run=Start-Process -FilePath $executorBinary -WorkingDirectory $executorRoot -ArgumentList @('--target-url',("https://[::1]:{0}/websocket" -f $smokePort),'--root-certificate',(Join-Path $executorRoot 'certs/root.pem'),'--output-dir',$artifactRoot) -RedirectStandardOutput $stdout -RedirectStandardError $stderr -WindowStyle Hidden -PassThru -Wait
        if ($run.ExitCode -ne 0) { throw "$scenarioId executor exit code $($run.ExitCode)." }
        $result=Get-Content (Join-Path $artifactRoot 'websocket-executor-result.json') -Raw | ConvertFrom-Json
        if ($result.status -ne 'passed' -or $result.scenarioId -ne $scenarioId -or $result.executor.id -ne 'go-http1-websocket-tls-executor' -or $result.executor.version -ne '0.2.0' -or $result.loadGenerator.id -ne 'go-http1-websocket-tls-load' -or $result.loadGenerator.version -ne '0.2.0' -or $result.protocolProof.requestedProtocol -ne 'websocket-over-h1-tls' -or $result.protocolProof.observedProtocol -ne 'websocket-over-h1-tls' -or $result.protocolProof.protocolVariant -ne 'websocket-h1-tls1.3-upgrade' -or $result.protocolProof.fallbackDetected -ne $false -or $result.protocolProof.tls.version -ne 'TLS 1.3' -or $result.protocolProof.tls.alpn -ne 'http/1.1' -or $result.protocolProof.tls.serverName -ne 'websocket.plab.test' -or $result.protocolProof.tls.didResume -ne $false -or $result.protocolProof.tls.earlyData -ne $false -or $result.protocolProof.tls.verifiedChainCount -lt 1 -or $result.metrics.completedOperations -le 0 -or $result.metrics.failedOperations -ne 0 -or $result.metrics.timedOutOperations -ne 0) { throw "$scenarioId normalized evidence failed validation." }
        $handshakeAggregate=$result.protocolProof.handshakeAggregate
        if ($null -eq $handshakeAggregate -or $handshakeAggregate.binding -ne 'http1-upgrade' -or $handshakeAggregate.openingHandshakes -le 0 -or $handshakeAggregate.keyReuseCount -ne 0 -or $handshakeAggregate.invalidDecodedKeyCount -ne 0 -or $handshakeAggregate.acceptMismatchCount -ne 0 -or $handshakeAggregate.upgradeRequestHeadersMatched -ne $true -or $handshakeAggregate.upgradeResponseHeadersMatched -ne $true -or [string]::IsNullOrWhiteSpace($handshakeAggregate.sampleSecWebSocketKey) -or [string]::IsNullOrWhiteSpace($handshakeAggregate.sampleSecWebSocketAccept)) { throw "$scenarioId handshake aggregate failed validation." }
        $websocket=$result.protocolProof.websocket
        if ($scenarioId -eq 'http1.websocket.rfc6455.tls.subprotocol-text-echo') {
            if ($websocket.handshake.subprotocolOffered -ne 'plab.echo.v1' -or $websocket.handshake.subprotocolAccepted -ne 'plab.echo.v1' -or $websocket.handshake.subprotocolRequested -ne $true -or $websocket.handshake.subprotocolNegotiated -ne $true -or $websocket.handshake.extensionsRequested -ne $false -or $websocket.handshake.extensionsNegotiated -ne $false -or $websocket.requestRsv1 -ne $false -or $websocket.responseRsv1 -ne $false -or $websocket.requestOpcode -ne 'text' -or $websocket.responseOpcode -ne 'text' -or $websocket.payloadHashMatched -ne $true -or $websocket.decompressedPayloadSha256 -ne '504585b0bb4fd77012ea2575efbcdb58f4c33e6b543e9567a65896d213720c29') { throw "$scenarioId exact subprotocol proof failed validation." }
        }
        if ($scenarioId -eq 'http1.websocket.rfc6455.tls.permessage-deflate-binary-echo') {
            $extension='permessage-deflate; client_no_context_takeover; server_no_context_takeover'
            if ($websocket.handshake.subprotocolRequested -ne $false -or $websocket.handshake.subprotocolNegotiated -ne $false -or $websocket.handshake.extensionOffered -ne $extension -or $websocket.handshake.extensionAccepted -ne $extension -or $websocket.handshake.extensionsRequested -ne $true -or $websocket.handshake.extensionsNegotiated -ne $true -or $websocket.handshake.clientNoContextTakeover -ne $true -or $websocket.handshake.serverNoContextTakeover -ne $true -or $websocket.requestRsv1 -ne $true -or $websocket.responseRsv1 -ne $true -or $websocket.compressedMessage -ne $true -or $websocket.requestOpcode -ne 'binary' -or $websocket.responseOpcode -ne 'binary' -or $websocket.payloadHashMatched -ne $true -or $websocket.decompressedPayloadSha256 -ne '9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a') { throw "$scenarioId exact permessage-deflate proof failed validation." }
        }
        foreach ($artifact in @('validation.json','protocol-proof.json','websocket-summary.json','tls-negotiation.json','payload-hash.json','result.json','handshake-summary.json','frame-summary.json','handshake-request.txt','handshake-response.txt','load.stdout.log','load.stderr.log')) { if (-not (Test-Path (Join-Path $artifactRoot $artifact) -PathType Leaf)) { throw "$scenarioId missing $artifact" } }
        $results += [ordered]@{scenarioId=$scenarioId;completedOperations=$result.metrics.completedOperations;failedOperations=$result.metrics.failedOperations;timedOutOperations=$result.metrics.timedOutOperations;artifactRoot=$artifactRoot}
    }
    $unsupportedIds=@('websocket.echo','http1.websocket.rfc6455.cleartext.upgrade','http1.websocket.rfc6455.cleartext.control-frames','http1.websocket.rfc6455.cleartext.text-echo','http1.websocket.rfc6455.cleartext.binary-echo','http1.websocket.rfc6455.cleartext.close','http2.websocket.rfc8441.extended-connect','http2.websocket.rfc8441.control-frames','http2.websocket.rfc8441.text-echo','http2.websocket.rfc8441.binary-echo','http2.websocket.rfc8441.close','http2.websocket.rfc8441.multi-message-text-echo','http3.websocket.rfc9220.extended-connect','http3.websocket.rfc9220.control-frames','http3.websocket.rfc9220.text-echo','http3.websocket.rfc9220.binary-echo','http3.websocket.rfc9220.close','http3.websocket.rfc9220.fragmented-binary-echo')
    $unsupportedProbes=@()
    foreach ($unsupportedId in $unsupportedIds) {
        $env:PLAB_SCENARIO_ID=$unsupportedId
        $unsupportedRoot=Join-Path $SmokeRoot ('unsupported-' + ($unsupportedId -replace '[^a-zA-Z0-9.-]','-')); New-Item -ItemType Directory -Force -Path $unsupportedRoot | Out-Null
        $unsupportedRun=Start-Process -FilePath $executorBinary -WorkingDirectory $executorRoot -ArgumentList @('--target-url',("https://[::1]:{0}/websocket" -f $smokePort),'--output-dir',$unsupportedRoot) -RedirectStandardOutput (Join-Path $unsupportedRoot 'load.stdout.log') -RedirectStandardError (Join-Path $unsupportedRoot 'load.stderr.log') -WindowStyle Hidden -PassThru -Wait
        if ($unsupportedRun.ExitCode -ne 3) { throw "Known unsupported identity $unsupportedId exited $($unsupportedRun.ExitCode), expected 3." }
        $unsupported=Get-Content (Join-Path $unsupportedRoot 'result.json') -Raw | ConvertFrom-Json
        if ($unsupported.schemaVersion -ne 'protocol-lab.unsupported.v1' -or $unsupported.status -ne 'unsupported' -or $unsupported.scenarioId -ne $unsupportedId) { throw "Unsupported evidence mismatch for $unsupportedId." }
        $unsupportedProbes += $unsupported
    }
    $env:PLAB_SCENARIO_ID='http1.websocket.rfc6455.tls.not-a-contract'
    $unknownRoot=Join-Path $SmokeRoot 'unknown'; New-Item -ItemType Directory -Force -Path $unknownRoot | Out-Null
    $unknownRun=Start-Process -FilePath $executorBinary -WorkingDirectory $executorRoot -ArgumentList @('--target-url',("https://[::1]:{0}/websocket" -f $smokePort),'--output-dir',$unknownRoot) -RedirectStandardOutput (Join-Path $unknownRoot 'load.stdout.log') -RedirectStandardError (Join-Path $unknownRoot 'load.stderr.log') -WindowStyle Hidden -PassThru -Wait
    if ($unknownRun.ExitCode -ne 2) { throw "Unknown identity exited $($unknownRun.ExitCode), expected 2." }
    $summary=[ordered]@{status='passed';authorityCommit='8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574';packages=@($scenarioPackage.FullName,$executorPackage.FullName,$targetPackage.FullName);results=$results;unsupportedProbes=$unsupportedProbes;unknownIdentityExitCode=$unknownRun.ExitCode}
    $summary | ConvertTo-Json -Depth 10 | Set-Content (Join-Path $SmokeRoot 'smoke-summary.json') -Encoding utf8NoBOM
    $summary
} finally {
    if ($null -ne $process -and -not $process.HasExited) { Stop-Process -Id $process.Id -Force; $process.WaitForExit() }
    foreach ($name in $variables) { [Environment]::SetEnvironmentVariable($name,$saved[$name],'Process') }
}
