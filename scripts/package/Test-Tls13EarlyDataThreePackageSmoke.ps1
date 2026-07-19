[CmdletBinding()]
param(
    [string]$PackageRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-early-data-packages'),
    [string]$ArtifactRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-early-data-extracted-smoke')
)

$ErrorActionPreference='Stop'
$PackageRoot=[IO.Path]::GetFullPath($PackageRoot);$ArtifactRoot=[IO.Path]::GetFullPath($ArtifactRoot)
$repoRoot=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
if(-not$ArtifactRoot.StartsWith([IO.Path]::GetFullPath((Join-Path $repoRoot 'artifacts')),[StringComparison]::OrdinalIgnoreCase)){throw 'TLS early-data smoke artifacts must remain under this worktree artifacts directory.'}

function Resolve-OnePackage([string]$Pattern){$matches=@(Get-ChildItem -LiteralPath $PackageRoot -Filter $Pattern -File);if($matches.Count-ne 1){throw "Expected one package matching $Pattern, observed $($matches.Count)."};$matches[0].FullName}
function Expand-Package([string]$Archive,[string]$Destination){New-Item -ItemType Directory -Force $Destination|Out-Null;[IO.Compression.ZipFile]::ExtractToDirectory($Archive,$Destination);$manifest=Get-Content (Join-Path $Destination 'protocol-lab-package.json') -Raw|ConvertFrom-Json;if($manifest.schemaVersion-ne'protocol-lab-package-v2'){throw "$Archive is not package-v2."};$manifest}

if(Test-Path $ArtifactRoot){Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force}
New-Item -ItemType Directory -Force $ArtifactRoot|Out-Null
$scenarioArchive=Resolve-OnePackage 'org.protocol-lab.components.scenario.tls13-handshake-performance.0.2.2.plabpkg'
$executorArchive=Resolve-OnePackage 'org.protocol-lab.components.executor.rustls-tls13-early-data-executor.0.1.0.win-x64.plabpkg'
$targetArchive=Resolve-OnePackage 'org.protocol-lab.components.implementation.rustls-tls13-early-data.0.1.0.win-x64.plabpkg'
$scenarioRoot=Join-Path $ArtifactRoot scenario;$executorRoot=Join-Path $ArtifactRoot executor;$targetRoot=Join-Path $ArtifactRoot target
$scenarioManifest=Expand-Package $scenarioArchive $scenarioRoot;$executorManifest=Expand-Package $executorArchive $executorRoot;$targetManifest=Expand-Package $targetArchive $targetRoot
$authority=Get-Content (Join-Path $scenarioRoot 'authority-lock.json') -Raw|ConvertFrom-Json
if($authority.commit-ne'd5b78d7c07ef0e8a600e92887da2aa150ab89a60'){throw 'Authority commit mismatch.'}
if($authority.files.'scenarios/tls/early-data/accepted.yaml'-ne'2f11fe6d69b5dc568b5c1a9c6549c3114de19101eb5ba0eaecd8403bfec6af78'-or$authority.files.'scenarios/tls/early-data/rejected.yaml'-ne'24a7d05bb773496da96db18821a724d2c4dc9d3e0f11fa1dd6795453b724b377'){throw 'Early-data scenario authority hash mismatch.'}
foreach($id in @('tls.early-data.accepted','tls.early-data.rejected')){
    if($executorManifest.providedTestExecutors[0].scenarios-notcontains$id){throw "Executor package does not claim $id."}
    if($targetManifest.providedImplementations[0].scenarios-notcontains$id){throw "Target package does not claim $id."}
}
foreach($root in @($executorRoot,$targetRoot)){
    if(-not(Test-Path (Join-Path $root 'third-party-licenses/index.json'))){throw 'Third-party license index missing from extracted package.'}
    $licenseIndex=Get-Content (Join-Path $root 'third-party-licenses/index.json') -Raw|ConvertFrom-Json
    foreach($dependency in @('rustls','rustls-rustcrypto','rustls-pemfile')){
        $entry=@($licenseIndex.packages|Where-Object name -eq $dependency)
        if($entry.Count-ne 1-or$entry[0].licenseFiles.Count-eq 0){throw "$dependency license material missing from extracted package."}
    }
}
if(Test-Path (Join-Path $executorRoot 'certs/leaf-key.pem')){throw 'Server private key leaked into executor package.'}
if(Test-Path (Join-Path $targetRoot 'certs/root.pem')){throw 'Out-of-band trust anchor leaked into target package.'}

$saved=@{};$envNames=@('PLAB_LISTEN_ADDRESS','PLAB_SCENARIO_ID','PLAB_TLS_CERT_FILE','PLAB_TLS_KEY_FILE','PLAB_TARGET_BASE_URL','PLAB_ARTIFACT_DIR','PLAB_TLS_ROOT_CERTIFICATE_PATH','PLAB_EXECUTOR_ID','PLAB_EXECUTOR_VERSION','PLAB_LOAD_GENERATOR_ID','PLAB_LOAD_GENERATOR_VERSION','PLAB_PROTOCOL','PLAB_PROTOCOL_VARIANT','PLAB_LOAD_PROFILE_ID')
foreach($name in $envNames){$saved[$name]=[Environment]::GetEnvironmentVariable($name,'Process')}
$caseEvidence=@();$target=$null
try{
    $cases=@(
        @{id='tls.early-data.accepted';variant='tls1.3-zero-rtt-accepted';outcome='accepted';port=18446;accepted=$true;retry=0;transferred=1024},
        @{id='tls.early-data.rejected';variant='tls1.3-zero-rtt-rejected';outcome='rejected';port=18447;accepted=$false;retry=1024;transferred=2048}
    )
    foreach($case in $cases){
        $caseRoot=Join-Path $ArtifactRoot ($case.outcome+'-cell');New-Item -ItemType Directory -Force $caseRoot|Out-Null
        $env:PLAB_LISTEN_ADDRESS="127.0.0.1:$($case.port)";$env:PLAB_SCENARIO_ID=$case.id
        $env:PLAB_TLS_CERT_FILE=Join-Path $targetRoot 'certs/leaf.pem';$env:PLAB_TLS_KEY_FILE=Join-Path $targetRoot 'certs/leaf-key.pem'
        $targetStdout=Join-Path $caseRoot 'target.stdout.log';$targetStderr=Join-Path $caseRoot 'target.stderr.log'
        $target=Start-Process -FilePath (Join-Path $targetRoot 'bin/win-x64/rustls-tls13-early-data.exe') -RedirectStandardOutput $targetStdout -RedirectStandardError $targetStderr -WindowStyle Hidden -PassThru
        $ready=$false;for($attempt=0;$attempt-lt 100;$attempt++){if((Test-Path $targetStdout)-and((Get-Content $targetStdout -Raw)-match '"eventName":"ready"')){$ready=$true;break};Start-Sleep -Milliseconds 50}
        if(-not$ready){throw "$($case.id) target did not become ready."}
        $executorArtifacts=Join-Path $caseRoot 'executor-artifacts';New-Item -ItemType Directory -Force $executorArtifacts|Out-Null
        $env:PLAB_TARGET_BASE_URL="tls://127.0.0.1:$($case.port)";$env:PLAB_ARTIFACT_DIR=$executorArtifacts;$env:PLAB_TLS_ROOT_CERTIFICATE_PATH=Join-Path $executorRoot 'certs/root.pem'
        $env:PLAB_EXECUTOR_ID='rustls-tls13-early-data-executor';$env:PLAB_EXECUTOR_VERSION='0.1.0';$env:PLAB_LOAD_GENERATOR_ID='rustls-tls13-early-data-load';$env:PLAB_LOAD_GENERATOR_VERSION='0.1.0'
        $env:PLAB_PROTOCOL='tls';$env:PLAB_PROTOCOL_VARIANT=$case.variant;$env:PLAB_LOAD_PROFILE_ID='tls-diagnostic'
        $run=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/rustls-tls13-early-data-executor.exe') -RedirectStandardOutput (Join-Path $executorArtifacts 'load.stdout.log') -RedirectStandardError (Join-Path $executorArtifacts 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
        if($run.ExitCode-ne 0){throw "$($case.id) executor failed with exit code $($run.ExitCode)."}
        foreach($required in @('validation.json','protocol-proof.json','tls-negotiation.json','resumption-proof.json','early-data-proof.json','payload-hash.json','result.json','tls-load-summary.json','tls-executor-result.json','executor-identity.json','load-generator-identity.json','load.stdout.log','load.stderr.log')){if(-not(Test-Path (Join-Path $executorArtifacts $required))){throw "$($case.id) required artifact missing: $required"}}
        $result=Get-Content (Join-Path $executorArtifacts 'tls-executor-result.json') -Raw|ConvertFrom-Json;$early=Get-Content (Join-Path $executorArtifacts 'early-data-proof.json') -Raw|ConvertFrom-Json;$resume=Get-Content (Join-Path $executorArtifacts 'resumption-proof.json') -Raw|ConvertFrom-Json;$payload=Get-Content (Join-Path $executorArtifacts 'payload-hash.json') -Raw|ConvertFrom-Json
        if($result.scenarioId-ne$case.id-or$result.executor.id-ne'rustls-tls13-early-data-executor'-or$result.loadGenerator.id-ne'rustls-tls13-early-data-load'){throw "$($case.id) identity mismatch."}
        if($result.protocolProof.tlsVersion-ne'TLS1.3'-or$result.protocolProof.alpn-ne'protocol-lab-tls'-or$result.protocolProof.cipherSuite-ne'TLS_AES_128_GCM_SHA256'-or$result.protocolProof.keyExchangeGroup-ne'X25519'-or-not$result.protocolProof.didResume-or-not$result.protocolProof.earlyDataAttempted-or$result.protocolProof.earlyDataAccepted-ne$case.accepted){throw "$($case.id) exact TLS/resumption/outcome proof failed."}
        if($result.protocolProof.certificateDerSha256-ne'cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e'-or$result.protocolProof.certificateSpkiSha256-ne'407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0'-or-not$result.protocolProof.certificateVerified){throw "$($case.id) certificate proof failed."}
        if($early.observedOutcome-ne$case.outcome-or-not$early.earlyDataOffered-or$early.earlyDataOfferedBytes-ne1024-or$early.postHandshakeRetryBytes-ne$case.retry-or$early.applicationEffectCount-ne1-or-not$early.zeroDuplicateEffects){throw "$($case.id) early-data semantic proof failed."}
        if($case.outcome-eq'rejected'-and-not$early.applicationRetriedExactlyOnceAfterHandshake){throw 'Rejected cell did not prove one retry.'}
        if(-not$resume.sessionTicketAvailableAfterSource-or-not$resume.sessionTicketConsumedExactlyOnce-or-not$resume.measuredSession.didResume){throw "$($case.id) single-use resumption proof failed."}
        if($payload.payloadLength-ne1024-or$payload.payloadOctet-ne90-or$payload.payloadSha256-ne'e8fb68ce4d4d002dba40c0a459d96807c96ded1c2fdefae3f56f8a0c06a4fecf'){throw "$($case.id) payload identity failed."}
        if($result.metrics.completedOperations-ne1-or$result.metrics.failedOperations-ne0-or$result.metrics.timedOutOperations-ne0-or$result.metrics.totalTransferredBytes-ne$case.transferred){throw "$($case.id) operation metrics failed."}
        Start-Sleep -Milliseconds 100
        $targetProofs=@(Get-Content $targetStdout|ForEach-Object{try{$_|ConvertFrom-Json -ErrorAction Stop}catch{}}|Where-Object {$_.eventName-eq'target-proof'-and$_.connectionRole-eq'measured'})
        if($targetProofs.Count-ne1){throw "$($case.id) measured target proof count mismatch."}
        $targetProof=$targetProofs[0]
        if(-not$targetProof.didResume-or$targetProof.earlyDataOutcome-ne$case.outcome-or$targetProof.applicationEffectCount-ne1-or-not$targetProof.zeroDuplicateEffects-or$targetProof.payloadSha256-ne'e8fb68ce4d4d002dba40c0a459d96807c96ded1c2fdefae3f56f8a0c06a4fecf'){throw "$($case.id) target-side proof failed."}
        if($case.outcome-eq'accepted'-and($targetProof.earlyDataBytesDelivered-ne1024-or$targetProof.postHandshakeRetryBytesDelivered-ne0)){throw 'Accepted target byte proof failed.'}
        if($case.outcome-eq'rejected'-and($targetProof.earlyDataBytesDelivered-ne0-or$targetProof.postHandshakeRetryBytesDelivered-ne1024)){throw 'Rejected target byte proof failed.'}
        $caseEvidence+=[ordered]@{scenarioId=$case.id;didResume=$true;earlyDataOutcome=$case.outcome;completedOperations=1;failedOperations=0;timedOutOperations=0;totalTransferredBytes=$case.transferred;evidenceRoot=$caseRoot}
        if(-not$target.HasExited){Stop-Process -Id $target.Id -Force};$target=$null
    }

    foreach($unsupported in @('tls.handshake.full','tls.handshake.resumed','tls.handshake.full.tls12','tls.handshake.full.chacha20','tls.handshake.mutual-auth','tls.key-update.diagnostic','tls.record.coverage','tls.record.throughput')){
        $unsupportedRoot=Join-Path $ArtifactRoot ('unsupported-'+($unsupported-replace'\.','-'));New-Item -ItemType Directory -Force $unsupportedRoot|Out-Null
        $env:PLAB_SCENARIO_ID=$unsupported;$env:PLAB_ARTIFACT_DIR=$unsupportedRoot
        $unsupportedRun=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/rustls-tls13-early-data-executor.exe') -RedirectStandardOutput (Join-Path $unsupportedRoot 'load.stdout.log') -RedirectStandardError (Join-Path $unsupportedRoot 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
        if($unsupportedRun.ExitCode-ne3){throw "$unsupported did not exit unsupported."}
        $unsupportedEvidence=Get-Content (Join-Path $unsupportedRoot 'unsupported.json') -Raw|ConvertFrom-Json
        if($unsupportedEvidence.status-ne'unsupported'-or$unsupportedEvidence.scenarioId-ne$unsupported){throw "$unsupported evidence mismatch."}
    }
    $unknownRoot=Join-Path $ArtifactRoot 'unknown';New-Item -ItemType Directory -Force $unknownRoot|Out-Null;$env:PLAB_SCENARIO_ID='tls.early-data.unknown';$env:PLAB_ARTIFACT_DIR=$unknownRoot
    $unknownRun=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/rustls-tls13-early-data-executor.exe') -RedirectStandardOutput (Join-Path $unknownRoot 'load.stdout.log') -RedirectStandardError (Join-Path $unknownRoot 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
    if($unknownRun.ExitCode-ne2-or(Test-Path (Join-Path $unknownRoot 'unsupported.json'))){throw 'Unknown scenario did not fail separately from unsupported.'}

    [ordered]@{
        authorityCommit=$authority.commit
        scenarioPackageSha256=(Get-FileHash $scenarioArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        executorPackageSha256=(Get-FileHash $executorArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        targetPackageSha256=(Get-FileHash $targetArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        cases=$caseEvidence
        unsupportedScenarioIds=@('tls.handshake.full','tls.handshake.resumed','tls.handshake.full.tls12','tls.handshake.full.chacha20','tls.handshake.mutual-auth','tls.key-update.diagnostic','tls.record.coverage','tls.record.throughput')
        evidenceRoot=$ArtifactRoot
    }|ConvertTo-Json -Depth 8
}finally{
    if($null-ne$target-and-not$target.HasExited){Stop-Process -Id $target.Id -Force}
    foreach($name in $envNames){[Environment]::SetEnvironmentVariable($name,$saved[$name],'Process')}
}
