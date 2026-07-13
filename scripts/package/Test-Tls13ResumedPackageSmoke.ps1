[CmdletBinding()]
param(
    [string]$PackageRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-resumed-packages-final'),
    [string]$ArtifactRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-resumed-extracted-smoke')
)

$ErrorActionPreference='Stop'
$PackageRoot=[IO.Path]::GetFullPath($PackageRoot)
$ArtifactRoot=[IO.Path]::GetFullPath($ArtifactRoot)
$repoRoot=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
if(-not $ArtifactRoot.StartsWith([IO.Path]::GetFullPath((Join-Path $repoRoot 'artifacts')),[StringComparison]::OrdinalIgnoreCase)){
    throw 'TLS resumed smoke artifacts must remain under this worktree artifacts directory.'
}

function Resolve-OnePackage([string]$Pattern){
    $matches=@(Get-ChildItem -LiteralPath $PackageRoot -Filter $Pattern -File)
    if($matches.Count-ne 1){throw "Expected exactly one package matching $Pattern, observed $($matches.Count)."}
    return $matches[0].FullName
}
function Expand-Package([string]$Archive,[string]$Destination){
    New-Item -ItemType Directory -Force $Destination|Out-Null
    [IO.Compression.ZipFile]::ExtractToDirectory($Archive,$Destination)
    $manifest=Get-Content (Join-Path $Destination 'protocol-lab-package.json') -Raw|ConvertFrom-Json
    if($manifest.schemaVersion-ne 'protocol-lab-package-v2'){throw "Package $Archive is not package-v2."}
    return $manifest
}

if(Test-Path -LiteralPath $ArtifactRoot){Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force}
New-Item -ItemType Directory -Force $ArtifactRoot|Out-Null
$scenarioArchive=Resolve-OnePackage 'org.protocol-lab.components.scenario.tls13-handshake-performance.0.1.0.plabpkg'
$executorArchive=Resolve-OnePackage 'org.protocol-lab.components.executor.go-tls13-executor.0.2.0.win-x64.plabpkg'
$targetArchive=Resolve-OnePackage 'org.protocol-lab.components.implementation.go-tls13.0.1.0.win-x64.plabpkg'
$scenarioRoot=Join-Path $ArtifactRoot 'scenario'
$executorRoot=Join-Path $ArtifactRoot 'executor'
$targetRoot=Join-Path $ArtifactRoot 'target'
$scenarioManifest=Expand-Package $scenarioArchive $scenarioRoot
$executorManifest=Expand-Package $executorArchive $executorRoot
$targetManifest=Expand-Package $targetArchive $targetRoot
$authority=Get-Content (Join-Path $scenarioRoot 'authority-lock.json') -Raw|ConvertFrom-Json
if($authority.commit-ne '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'){throw 'Scenario package authority commit mismatch.'}
if($executorManifest.providedTestExecutors[0].scenarios-notcontains 'tls.handshake.resumed'){throw 'Executor package does not claim exact resumed scenario support.'}
if($targetManifest.providedImplementations[0].scenarios-notcontains 'tls.handshake.resumed'){throw 'Target package does not claim exact resumed scenario support.'}

$env:PLAB_LISTEN_ADDRESS='127.0.0.1:18443'
$env:PLAB_TLS_CERT_FILE=Join-Path $targetRoot 'certs/leaf.pem'
$env:PLAB_TLS_KEY_FILE=Join-Path $targetRoot 'certs/leaf-key.pem'
$targetStdout=Join-Path $ArtifactRoot 'target.stdout.log'
$targetStderr=Join-Path $ArtifactRoot 'target.stderr.log'
$target=Start-Process -FilePath (Join-Path $targetRoot 'bin/win-x64/go-tls13.exe') -RedirectStandardOutput $targetStdout -RedirectStandardError $targetStderr -WindowStyle Hidden -PassThru
try{
    $ready=$false
    for($attempt=0;$attempt-lt 40;$attempt++){
        try{$tcp=[Net.Sockets.TcpClient]::new();$tcp.Connect('127.0.0.1',18443);$tcp.Dispose();$ready=$true;break}catch{Start-Sleep -Milliseconds 100}
    }
    if(-not $ready){throw 'Extracted Go TLS target did not become ready.'}
    $executorArtifacts=Join-Path $ArtifactRoot 'executor-artifacts'
    New-Item -ItemType Directory -Force $executorArtifacts|Out-Null
    $env:PLAB_TARGET_BASE_URL='tls://127.0.0.1:18443'
    $env:PLAB_ARTIFACT_DIR=$executorArtifacts
    $env:PLAB_TLS_ROOT_CERTIFICATE_PATH=Join-Path $executorRoot 'certs/root.pem'
    $env:PLAB_EXECUTOR_ID='go-tls13-executor';$env:PLAB_EXECUTOR_VERSION='0.2.0'
    $env:PLAB_LOAD_GENERATOR_ID='go-crypto-tls13-handshake-load';$env:PLAB_LOAD_GENERATOR_VERSION='0.2.0'
    $env:PLAB_PROTOCOL='tls';$env:PLAB_PROTOCOL_VARIANT='tls1.3-psk-resumed'
    $env:PLAB_SCENARIO_ID='tls.handshake.resumed';$env:PLAB_LOAD_PROFILE_ID='tls-smoke'
    $env:PLAB_CONNECTIONS='1';$env:PLAB_CONCURRENCY='1';$env:PLAB_DURATION_SECONDS='5';$env:PLAB_WARMUP_SECONDS='1';$env:PLAB_REPETITION='1';$env:PLAB_REQUEST_TIMEOUT_SECONDS='5'
    $loadStdout=Join-Path $executorArtifacts 'load.stdout.log'
    $loadStderr=Join-Path $executorArtifacts 'load.stderr.log'
    $execution=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/go-tls13-executor.exe') -RedirectStandardOutput $loadStdout -RedirectStandardError $loadStderr -WindowStyle Hidden -Wait -PassThru
    if($execution.ExitCode-ne 0){throw "Extracted resumed executor failed with exit code $($execution.ExitCode)."}
    $result=Get-Content (Join-Path $executorArtifacts 'tls-executor-result.json') -Raw|ConvertFrom-Json
    $proof=Get-Content (Join-Path $executorArtifacts 'resumption-proof.json') -Raw|ConvertFrom-Json
    if($result.executor.id-ne 'go-tls13-executor' -or $result.executor.version-ne '0.2.0' -or
       $result.loadGenerator.id-ne 'go-crypto-tls13-handshake-load' -or $result.loadGenerator.version-ne '0.2.0' -or
       $result.protocolProof.tlsVersion-ne 'TLS1.3' -or $result.protocolProof.alpn-ne 'protocol-lab-tls' -or $result.protocolProof.keyExchangeGroup-ne 'X25519' -or
       -not $result.protocolProof.didResume -or $result.protocolProof.earlyDataAttempted -or
       $result.protocolProof.applicationDataBytesSent-ne 0 -or $result.protocolProof.applicationDataBytesReceived-ne 0 -or
       $result.metrics.completedOperations-lt 1 -or $result.metrics.failedOperations-ne 0 -or $result.metrics.timedOutOperations-ne 0){
        throw 'Extracted resumed result failed the exact TLS validity gate.'
    }
    if($proof.resumptionPolicy-ne 'accepted-psk-single-use-ticket' -or
       $proof.prerequisitePolicy-ne 'unmeasured-source-session-per-measured-operation' -or
       $proof.warmupIsolation-ne 'warmup-state-not-reused-by-measurement' -or
       $proof.sourceSession.didResume -or -not $proof.measuredSession.didResume -or
       -not $proof.sourceHandshakeOutsideMeasuredWindow -or -not $proof.sessionTicketConsumedExactlyOnce -or
       $proof.warmupSessionStateReusedByMeasurement){throw 'Extracted resumption proof failed the contract gate.'}
    [pscustomobject]@{
        authorityCommit=$authority.commit
        scenarioPackageSha256=(Get-FileHash $scenarioArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        executorPackageSha256=(Get-FileHash $executorArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        targetPackageSha256=(Get-FileHash $targetArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        completedOperations=[int]$result.metrics.completedOperations
        failedOperations=[int]$result.metrics.failedOperations
        timedOutOperations=[int]$result.metrics.timedOutOperations
        didResume=[bool]$result.protocolProof.didResume
        evidenceRoot=$ArtifactRoot
    }|ConvertTo-Json -Depth 5
}finally{
    if($null-ne $target -and -not $target.HasExited){Stop-Process -Id $target.Id -Force}
}
