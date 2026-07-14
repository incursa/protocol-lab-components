[CmdletBinding()]
param(
    [string]$PackageRoot = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/rfc9220-fragmented-packages'),
    [string]$ArtifactRoot = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/rfc9220-fragmented-extracted-smoke')
)

$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$PackageRoot = [IO.Path]::GetFullPath($PackageRoot)
$ArtifactRoot = [IO.Path]::GetFullPath($ArtifactRoot)
$allowedRoot = [IO.Path]::GetFullPath((Join-Path $repoRoot 'artifacts'))
if (-not $ArtifactRoot.StartsWith($allowedRoot, [StringComparison]::OrdinalIgnoreCase)) { throw 'Smoke artifacts must remain under the worktree artifacts directory.' }
if (Test-Path $ArtifactRoot) { Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force }
New-Item -ItemType Directory -Force $ArtifactRoot | Out-Null
Add-Type -AssemblyName System.IO.Compression.FileSystem

function Resolve-One([string]$pattern) {
    $matches = @(Get-ChildItem -LiteralPath $PackageRoot -Filter $pattern -File)
    if ($matches.Count -ne 1) { throw "Expected one $pattern package, observed $($matches.Count)." }
    return $matches[0].FullName
}
function Expand-One([string]$archive, [string]$destination) {
    New-Item -ItemType Directory -Force $destination | Out-Null
    [IO.Compression.ZipFile]::ExtractToDirectory($archive, $destination)
    return Get-Content (Join-Path $destination 'protocol-lab-package.json') -Raw | ConvertFrom-Json
}

$scenarioArchive = Resolve-One 'org.protocol-lab.components.scenario.aioquic-rfc9220-websocket.0.2.2.plabpkg'
$executorArchive = Resolve-One 'org.protocol-lab.components.executor.aioquic-rfc9220-websocket.0.2.1.plabpkg'
$targetArchive = Resolve-One 'org.protocol-lab.components.implementation.aioquic-http3.0.2.1.plabpkg'
$scenarioRoot = Join-Path $ArtifactRoot 'scenario'
$executorRoot = Join-Path $ArtifactRoot 'executor'
$targetRoot = Join-Path $ArtifactRoot 'target'
$scenarioManifest = Expand-One $scenarioArchive $scenarioRoot
$executorManifest = Expand-One $executorArchive $executorRoot
$targetManifest = Expand-One $targetArchive $targetRoot
if ($scenarioManifest.providedScenarios.Count -ne 6 -or $executorManifest.providedTestExecutors[0].scenarios.Count -ne 6) { throw 'Scenario or executor package does not claim exactly six RFC9220 identities.' }
if ((@($scenarioManifest.providedLoadProfiles.loadProfileId) -join ',') -ne 'websocket-smoke,diagnostic') { throw 'Scenario package load-profile declarations mismatch.' }
if ((@($scenarioManifest.providedSuites.suiteId) -join ',') -ne 'aioquic-rfc9220-websocket-proof,aioquic-rfc9220-websocket-fragmentation-diagnostic') { throw 'Scenario package suite declarations mismatch.' }
if (@($targetManifest.providedImplementations[0].scenarios | Where-Object { $_ -like 'http3.websocket.rfc9220.*' }).Count -ne 6) { throw 'Target package does not claim exactly six RFC9220 identities.' }
$authority = Get-Content (Join-Path $scenarioRoot 'authority-lock.json') -Raw | ConvertFrom-Json
if ($authority.authorityCommit -ne '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574') { throw 'Authority commit mismatch.' }
$profilePath = Join-Path $scenarioRoot 'load-profiles/websocket-smoke.yaml'
if (-not (Test-Path $profilePath) -or (Get-FileHash $profilePath -Algorithm SHA256).Hash.ToLowerInvariant() -ne 'f2005bfa254815f7d4975aefc39f0b9a6da79b0d2507178775cd4b0b3032c645') { throw 'Extracted websocket-smoke authority bytes mismatch.' }
$diagnosticPath = Join-Path $scenarioRoot 'load-profiles/diagnostic.yaml'
if (-not (Test-Path $diagnosticPath) -or (Get-FileHash $diagnosticPath -Algorithm SHA256).Hash.ToLowerInvariant() -ne '0e0b798a876a7cdf309e9f0138bff089b92666d60d9a69037b7e0d1b1ef34968') { throw 'Extracted diagnostic authority bytes mismatch.' }
$coreSuite = Get-Content (Join-Path $scenarioRoot 'suites/aioquic-rfc9220-websocket-proof.yaml') -Raw
$diagnosticSuite = Get-Content (Join-Path $scenarioRoot 'suites/aioquic-rfc9220-websocket-fragmentation-diagnostic.yaml') -Raw
if ($coreSuite -match 'fragmented-binary-echo' -or $coreSuite -notmatch 'loadProfileId: websocket-smoke' -or $diagnosticSuite -notmatch 'loadProfileId: diagnostic' -or $diagnosticSuite -notmatch 'http3.websocket.rfc9220.fragmented-binary-echo') { throw 'Extracted RFC9220 suite routing mismatch.' }
foreach ($root in @($executorRoot, $targetRoot)) { if (-not (Test-Path (Join-Path $root 'third-party/aioquic-LICENSE.txt'))) { throw "aioquic license missing from $root" } }
& python (Join-Path $scenarioRoot 'tests/test_authority_parity.py') --scenario-root $scenarioRoot --executor-root $executorRoot --target-root $targetRoot | Out-Host
if ($LASTEXITCODE -ne 0) { throw 'Extracted six-scenario authority parity failed.' }

$executorImage = 'incursa-protocol-lab-aioquic-rfc9220-websocket:0.2.1-extracted-smoke'
$targetImage = 'incursa-protocol-lab-aioquic-http3:0.2.1-extracted-smoke'
& docker build --build-arg AIOQUIC_VERSION=1.3.0 -f (Join-Path $executorRoot 'docker/aioquic-rfc9220-websocket.Dockerfile') -t $executorImage $executorRoot | Out-Host
if ($LASTEXITCODE -ne 0) { throw 'Extracted executor image build failed.' }
& docker build --build-arg AIOQUIC_VERSION=1.3.0 -f (Join-Path $targetRoot 'docker/aioquic.Dockerfile') -t $targetImage $targetRoot | Out-Host
if ($LASTEXITCODE -ne 0) { throw 'Extracted target image build failed.' }
& docker run --rm --entrypoint python $executorImage -m unittest discover -s /work/tests -v | Out-Host
if ($LASTEXITCODE -ne 0) { throw 'Extracted executor tests failed.' }
& docker run --rm --entrypoint python $targetImage -m unittest discover -s /work/tests -v | Out-Host
if ($LASTEXITCODE -ne 0) { throw 'Extracted target tests failed.' }

$container = 'plab-rfc9220-fragmented-extracted-smoke'
& docker rm -f $container 2>$null | Out-Null
& docker run -d --name $container -p 18462:4433/udp $targetImage /usr/local/bin/aioquic-http3-server /www /certs/cert.pem /certs/priv.key 4433 | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Extracted target start failed.' }
try {
    $ready = $false
    for ($index = 0; $index -lt 100; $index++) {
        if ((& docker logs $container 2>&1) -match '"eventName": "ready"') { $ready = $true; break }
        Start-Sleep -Milliseconds 100
    }
    if (-not $ready) { throw "Extracted target did not become ready: $(& docker logs $container 2>&1)" }
    $ids = @(
        'http3.websocket.rfc9220.extended-connect',
        'http3.websocket.rfc9220.control-frames',
        'http3.websocket.rfc9220.text-echo',
        'http3.websocket.rfc9220.binary-echo',
        'http3.websocket.rfc9220.close',
        'http3.websocket.rfc9220.fragmented-binary-echo'
    )
    $outcomes = @()
    foreach ($scenarioId in $ids) {
        $loadProfileId = if ($scenarioId -eq 'http3.websocket.rfc9220.fragmented-binary-echo') { 'diagnostic' } else { 'websocket-smoke' }
        $env:PLAB_LOAD_PROFILE_ID = $loadProfileId
        $output = Join-Path $ArtifactRoot ('evidence/' + ($scenarioId -replace '\.', '-'))
        & (Join-Path $executorRoot 'execute.ps1') -ScenarioId $scenarioId -TargetUrl 'https://127.0.0.1:18462/websocket-proof' -OutputRoot $output -Image $executorImage -SkipBuild
        if ($LASTEXITCODE -ne 0) { throw "$scenarioId executor failed with exit code $LASTEXITCODE" }
        foreach ($required in @('validation.json', 'protocol-proof.json', 'websocket-summary.json', 'payload-hash.json', 'frame-summary.json', 'client-result.json', 'result.json', 'load.stdout.log', 'load.stderr.log', 'executor-identity.json')) {
            if (-not (Test-Path (Join-Path $output $required))) { throw "$scenarioId missing $required" }
        }
        $result = Get-Content (Join-Path $output 'result.json') -Raw | ConvertFrom-Json
        $proof = $result.protocolProof
        if ($result.scenarioId -ne $scenarioId -or $result.status -ne 'passed' -or $proof.protocol -ne 'h3' -or $proof.tlsVersion -ne 'TLS 1.3' -or $proof.alpn -ne 'h3' -or $proof.settingsEnableConnectProtocol -ne 1 -or -not $proof.noFallback -or $proof.requestPseudoHeaders.':method' -ne 'CONNECT' -or $proof.requestPseudoHeaders.':protocol' -ne 'websocket' -or $proof.requestPseudoHeaders.':scheme' -ne 'https' -or $proof.requestPseudoHeaders.':authority' -ne 'websocket.plab.test' -or $proof.requestPseudoHeaders.':path' -ne '/websocket-proof' -or $proof.responseStatus -ne 200 -or $proof.secWebSocketAcceptPresent -or $proof.secWebSocketProtocolPresent -or $proof.secWebSocketExtensionsPresent -or -not $proof.clientMaskObserved -or $proof.closeSent -ne 1000 -or $proof.closeReceived -ne 1000 -or $result.metrics.completedOperations -ne 1 -or $result.metrics.failedOperations -ne 0 -or $result.metrics.timedOutOperations -ne 0) { throw "$scenarioId common proof mismatch" }
        if ($scenarioId -eq 'http3.websocket.rfc9220.fragmented-binary-echo') {
            if (($proof.fragmentPayloadBytes -join ',') -ne '1024,2048,2928' -or ($proof.fragmentOpcodes -join ',') -ne 'binary,continuation,continuation' -or ($proof.fragmentFin -join ',') -ne 'False,False,True' -or $proof.interleavedControlFrames -or $proof.reassembledPayloadBytes -ne 6000 -or $proof.reassembledPayloadSha256 -ne '8f8d8f75d55c80475ffb0c12b1ede7083d6df689e8ef04f05176c5050873bfb7') { throw 'Fragmented binary proof mismatch.' }
        }
        $outcomes += [pscustomobject]@{ scenarioId = $scenarioId; loadProfileId = $loadProfileId; completed = 1; failed = 0; timedOut = 0 }
    }
    $unsupported = @('websocket.echo', 'http1.websocket.rfc6455.cleartext.upgrade', 'http1.websocket.rfc6455.cleartext.control-frames', 'http1.websocket.rfc6455.cleartext.text-echo', 'http1.websocket.rfc6455.cleartext.binary-echo', 'http1.websocket.rfc6455.cleartext.close', 'http1.websocket.rfc6455.tls.upgrade', 'http1.websocket.rfc6455.tls.control-frames', 'http1.websocket.rfc6455.tls.text-echo', 'http1.websocket.rfc6455.tls.binary-echo', 'http1.websocket.rfc6455.tls.close', 'http1.websocket.rfc6455.tls.subprotocol-text-echo', 'http1.websocket.rfc6455.tls.permessage-deflate-binary-echo', 'http2.websocket.rfc8441.extended-connect', 'http2.websocket.rfc8441.control-frames', 'http2.websocket.rfc8441.text-echo', 'http2.websocket.rfc8441.binary-echo', 'http2.websocket.rfc8441.close', 'http2.websocket.rfc8441.multi-message-text-echo')
    foreach ($scenarioId in $unsupported) {
        $output = Join-Path $ArtifactRoot ('unsupported/' + ($scenarioId -replace '\.', '-'))
        $process = Start-Process pwsh -ArgumentList @('-NoProfile', '-File', (Join-Path $executorRoot 'execute.ps1'), '-ScenarioId', $scenarioId, '-OutputRoot', $output, '-SkipBuild') -Wait -PassThru -WindowStyle Hidden
        if ($process.ExitCode -ne 3) { throw "$scenarioId did not return unsupported." }
    }
    $unknownOutput = Join-Path $ArtifactRoot 'unknown'
    $unknown = Start-Process pwsh -ArgumentList @('-NoProfile', '-File', (Join-Path $executorRoot 'execute.ps1'), '-ScenarioId', 'http3.websocket.rfc9220.unknown', '-OutputRoot', $unknownOutput, '-SkipBuild') -Wait -PassThru -WindowStyle Hidden
    if ($unknown.ExitCode -ne 2) { throw 'Unknown RFC9220 identity did not fail closed.' }
    $targetLog = (& docker logs $container 2>&1) -join "`n"
    $targetLog | Set-Content (Join-Path $ArtifactRoot 'target.log') -Encoding utf8NoBOM
    if (([regex]::Matches($targetLog, 'rfc9220-websocket-clean-close')).Count -lt 6 -or ([regex]::Matches($targetLog, 'rfc9220-fragmented-binary-reassembled')).Count -ne 1) { throw 'Target close or fragmentation proof incomplete.' }
    [pscustomobject]@{
        authorityCommit = $authority.authorityCommit
        scenarioPackageSha256 = (Get-FileHash $scenarioArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        executorPackageSha256 = (Get-FileHash $executorArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        targetPackageSha256 = (Get-FileHash $targetArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        outcomes = $outcomes
        unsupportedCount = $unsupported.Count
        evidenceRoot = $ArtifactRoot
    } | ConvertTo-Json -Depth 8
}
finally { & docker rm -f $container 2>$null | Out-Null }
