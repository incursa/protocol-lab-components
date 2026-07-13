[CmdletBinding()]
param(
    [string]$PackageRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-mtls-packages'),
    [string]$ArtifactRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-mtls-extracted-smoke')
)

$ErrorActionPreference='Stop'
$PackageRoot=[IO.Path]::GetFullPath($PackageRoot)
$ArtifactRoot=[IO.Path]::GetFullPath($ArtifactRoot)
$repoRoot=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
if(-not $ArtifactRoot.StartsWith([IO.Path]::GetFullPath((Join-Path $repoRoot 'artifacts')),[StringComparison]::OrdinalIgnoreCase)){throw 'TLS mTLS smoke artifacts must remain under this worktree artifacts directory.'}

function Resolve-OnePackage([string]$Pattern){$matches=@(Get-ChildItem -LiteralPath $PackageRoot -Filter $Pattern -File);if($matches.Count-ne 1){throw "Expected one package matching $Pattern, observed $($matches.Count)."};return $matches[0].FullName}
function Expand-Package([string]$Archive,[string]$Destination){New-Item -ItemType Directory -Force $Destination|Out-Null;[IO.Compression.ZipFile]::ExtractToDirectory($Archive,$Destination);$manifest=Get-Content (Join-Path $Destination 'protocol-lab-package.json') -Raw|ConvertFrom-Json;if($manifest.schemaVersion-ne 'protocol-lab-package-v2'){throw "$Archive is not package-v2."};return $manifest}

if(Test-Path $ArtifactRoot){Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force}
New-Item -ItemType Directory -Force $ArtifactRoot|Out-Null
$scenarioArchive=Resolve-OnePackage 'org.protocol-lab.components.scenario.tls13-handshake-performance.0.2.0.plabpkg'
$executorArchive=Resolve-OnePackage 'org.protocol-lab.components.executor.go-tls13-mtls-executor.0.1.0.win-x64.plabpkg'
$targetArchive=Resolve-OnePackage 'org.protocol-lab.components.implementation.go-tls13-mtls.0.1.0.win-x64.plabpkg'
$scenarioRoot=Join-Path $ArtifactRoot scenario;$executorRoot=Join-Path $ArtifactRoot executor;$targetRoot=Join-Path $ArtifactRoot target
$scenarioManifest=Expand-Package $scenarioArchive $scenarioRoot;$executorManifest=Expand-Package $executorArchive $executorRoot;$targetManifest=Expand-Package $targetArchive $targetRoot
$authority=Get-Content (Join-Path $scenarioRoot 'authority-lock.json') -Raw|ConvertFrom-Json
if($authority.commit-ne '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'){throw 'Authority commit mismatch.'}
if($authority.files.'scenarios/tls/handshake/mutual-auth.yaml'-ne '355686d36b75d9c0854f29c3f7ac14b526edbe42d004b469b8a370be9c19ec02'){throw 'Mutual-auth scenario authority hash mismatch.'}
if($executorManifest.providedTestExecutors[0].scenarios-notcontains 'tls.handshake.mutual-auth'){throw 'Executor package does not claim the exact scenario.'}
if($targetManifest.providedImplementations[0].scenarios-notcontains 'tls.handshake.mutual-auth'){throw 'Target package does not claim the exact scenario.'}
if(Test-Path (Join-Path $targetRoot 'certs/client.pem')){throw 'Client leaf leaked into the target package.'}
if(Test-Path (Join-Path $executorRoot 'certs/client-root.pem')){throw 'Client trust anchor leaked into the executor package.'}

$envNames=@('PLAB_LISTEN_ADDRESS','PLAB_SCENARIO_ID','PLAB_TLS_CERT_FILE','PLAB_TLS_KEY_FILE','PLAB_TLS_CLIENT_ROOT_CERTIFICATE_PATH','PLAB_TARGET_BASE_URL','PLAB_ARTIFACT_DIR','PLAB_TLS_ROOT_CERTIFICATE_PATH','PLAB_TLS_CLIENT_CERTIFICATE_PATH','PLAB_TLS_CLIENT_KEY_PATH','PLAB_EXECUTOR_ID','PLAB_EXECUTOR_VERSION','PLAB_LOAD_GENERATOR_ID','PLAB_LOAD_GENERATOR_VERSION','PLAB_PROTOCOL','PLAB_PROTOCOL_VARIANT','PLAB_LOAD_PROFILE_ID')
$saved=@{};foreach($name in $envNames){$saved[$name]=[Environment]::GetEnvironmentVariable($name,'Process')}
$target=$null
try{
    $port=18445
    $env:PLAB_LISTEN_ADDRESS="127.0.0.1:$port";$env:PLAB_SCENARIO_ID='tls.handshake.mutual-auth'
    $env:PLAB_TLS_CERT_FILE=Join-Path $targetRoot 'certs/leaf.pem';$env:PLAB_TLS_KEY_FILE=Join-Path $targetRoot 'certs/leaf-key.pem';$env:PLAB_TLS_CLIENT_ROOT_CERTIFICATE_PATH=Join-Path $targetRoot 'certs/client-root.pem'
    $targetStdout=Join-Path $ArtifactRoot 'target.stdout.log';$targetStderr=Join-Path $ArtifactRoot 'target.stderr.log'
    $target=Start-Process -FilePath (Join-Path $targetRoot 'bin/win-x64/go-tls13-mtls.exe') -RedirectStandardOutput $targetStdout -RedirectStandardError $targetStderr -WindowStyle Hidden -PassThru
    $ready=$false;for($attempt=0;$attempt-lt 50;$attempt++){if((Test-Path $targetStdout)-and((Get-Content $targetStdout -Raw)-match '"eventName":"ready"')){$ready=$true;break};Start-Sleep -Milliseconds 100}
    if(-not $ready){throw 'TLS mTLS target did not become ready.'}

    $executorArtifacts=Join-Path $ArtifactRoot 'executor-artifacts';New-Item -ItemType Directory -Force $executorArtifacts|Out-Null
    $env:PLAB_TARGET_BASE_URL="tls://127.0.0.1:$port";$env:PLAB_ARTIFACT_DIR=$executorArtifacts
    $env:PLAB_TLS_ROOT_CERTIFICATE_PATH=Join-Path $executorRoot 'certs/root.pem';$env:PLAB_TLS_CLIENT_CERTIFICATE_PATH=Join-Path $executorRoot 'certs/client.pem';$env:PLAB_TLS_CLIENT_KEY_PATH=Join-Path $executorRoot 'certs/client-key.pem'
    $env:PLAB_EXECUTOR_ID='go-tls13-mtls-executor';$env:PLAB_EXECUTOR_VERSION='0.1.0';$env:PLAB_LOAD_GENERATOR_ID='go-crypto-tls13-mtls-load';$env:PLAB_LOAD_GENERATOR_VERSION='0.1.0'
    $env:PLAB_PROTOCOL='tls';$env:PLAB_PROTOCOL_VARIANT='tls1.3-full-mutual-auth';$env:PLAB_LOAD_PROFILE_ID='tls-smoke'
    $loadStdout=Join-Path $executorArtifacts 'load.stdout.log';$loadStderr=Join-Path $executorArtifacts 'load.stderr.log'
    $execution=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/go-tls13-mtls-executor.exe') -ArgumentList '--validation-only' -RedirectStandardOutput $loadStdout -RedirectStandardError $loadStderr -WindowStyle Hidden -Wait -PassThru
    if($execution.ExitCode-ne 0){throw "TLS mTLS executor failed with exit code $($execution.ExitCode)."}
    foreach($required in @('validation.json','protocol-proof.json','tls-negotiation.json','peer-auth-proof.json','result.json','tls-executor-result.json','tls-load-summary.json','executor-identity.json','load-generator-identity.json','load.stdout.log','load.stderr.log')){if(-not(Test-Path (Join-Path $executorArtifacts $required))){throw "Required artifact missing: $required"}}
    $result=Get-Content (Join-Path $executorArtifacts 'tls-executor-result.json') -Raw|ConvertFrom-Json
    $proof=Get-Content (Join-Path $executorArtifacts 'peer-auth-proof.json') -Raw|ConvertFrom-Json
    if($result.scenarioId-ne 'tls.handshake.mutual-auth'-or$result.executor.id-ne 'go-tls13-mtls-executor'-or$result.executor.version-ne '0.1.0'-or$result.loadGenerator.id-ne 'go-crypto-tls13-mtls-load'-or$result.loadGenerator.version-ne '0.1.0'){throw 'Executor or generator identity mismatch.'}
    if($result.protocolProof.tlsVersion-ne 'TLS1.3'-or$result.protocolProof.alpn-ne 'protocol-lab-tls'-or$result.protocolProof.keyExchangeGroup-ne 'X25519'-or$result.protocolProof.cipherSuite-ne 'TLS_AES_128_GCM_SHA256'-or$result.protocolProof.didResume-or$result.protocolProof.earlyDataAttempted-or$result.protocolProof.applicationDataBytesSent-ne 0-or$result.protocolProof.applicationDataBytesReceived-ne 0){throw 'Exact TLS negotiation gate failed.'}
    if($proof.serverCertificateDerSha256-ne 'cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e'-or$proof.serverCertificateSpkiSha256-ne '407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0'-or$proof.clientCertificateDerSha256-ne 'ca2e4f661e7b29cfc516c48f53c05be0ef59fb6cc410cb205f5759e07a5deb20'-or$proof.clientCertificateSpkiSha256-ne '4b3a176400147e50a4efc3a7a26f66a9dec74a11042b7565eadd85b1ee27c0fb'){throw 'Certificate identity proof mismatch.'}
    if(-not$proof.serverCertificateVerified-or-not$proof.clientCertificateVerified-or-not$proof.mutualAuthenticated-or$proof.clientCertificateChainSentCount-ne 1-or$proof.clientTrustAnchorSent){throw 'Mutual authentication proof gate failed.'}
    if($result.metrics.completedOperations-ne 1-or$result.metrics.failedOperations-ne 0-or$result.metrics.timedOutOperations-ne 0-or$result.metrics.totalTransferredBytes-ne 0){throw 'Operation outcome gate failed.'}

    foreach($unsupported in @('tls.handshake.full','tls.handshake.resumed','tls.handshake.full.tls12','tls.handshake.full.chacha20','tls.early-data.accepted','tls.early-data.rejected','tls.key-update.diagnostic','tls.record.throughput','tls.record.coverage')){
        $unsupportedRoot=Join-Path $ArtifactRoot ('unsupported-'+($unsupported-replace '\.','-'));New-Item -ItemType Directory -Force $unsupportedRoot|Out-Null
        $env:PLAB_SCENARIO_ID=$unsupported;$env:PLAB_ARTIFACT_DIR=$unsupportedRoot
        $unsupportedRun=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/go-tls13-mtls-executor.exe') -RedirectStandardOutput (Join-Path $unsupportedRoot 'load.stdout.log') -RedirectStandardError (Join-Path $unsupportedRoot 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
        if($unsupportedRun.ExitCode-ne 3){throw "$unsupported did not exit as unsupported."}
        $unsupportedEvidence=Get-Content (Join-Path $unsupportedRoot 'unsupported.json') -Raw|ConvertFrom-Json
        if($unsupportedEvidence.status-ne 'unsupported'-or$unsupportedEvidence.scenarioId-ne $unsupported){throw "$unsupported evidence mismatch."}
    }

    [pscustomobject]@{
        authorityCommit=$authority.commit
        scenarioPackageSha256=(Get-FileHash $scenarioArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        executorPackageSha256=(Get-FileHash $executorArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        targetPackageSha256=(Get-FileHash $targetArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        completedOperations=[int]$result.metrics.completedOperations
        failedOperations=[int]$result.metrics.failedOperations
        timedOutOperations=[int]$result.metrics.timedOutOperations
        unsupportedScenarioIds=@('tls.handshake.full','tls.handshake.resumed','tls.handshake.full.tls12','tls.handshake.full.chacha20','tls.early-data.accepted','tls.early-data.rejected','tls.key-update.diagnostic','tls.record.throughput','tls.record.coverage')
        evidenceRoot=$ArtifactRoot
    }|ConvertTo-Json -Depth 8
}finally{
    if($null-ne $target-and-not $target.HasExited){Stop-Process -Id $target.Id -Force}
    foreach($name in $envNames){[Environment]::SetEnvironmentVariable($name,$saved[$name],'Process')}
}
