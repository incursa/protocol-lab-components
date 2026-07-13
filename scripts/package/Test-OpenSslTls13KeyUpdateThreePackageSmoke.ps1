[CmdletBinding()]
param(
    [ValidateSet('win-x64','linux-x64')][string]$RuntimeIdentifier='win-x64',
    [string]$PackageRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-key-update-packages'),
    [string]$ArtifactRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-key-update-extracted-smoke'),
    [string]$PublicRoot='C:\shared\src\incursa\protocol-lab',
    [string]$LinuxImage='alpine@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412'
)

$ErrorActionPreference='Stop'
$repoRoot=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$PackageRoot=[IO.Path]::GetFullPath($PackageRoot);$ArtifactRoot=[IO.Path]::GetFullPath($ArtifactRoot)
if(-not$ArtifactRoot.StartsWith([IO.Path]::GetFullPath((Join-Path $repoRoot artifacts)),[StringComparison]::OrdinalIgnoreCase)){throw 'KeyUpdate smoke artifacts must remain under this worktree artifacts directory.'}
if((git -C $PublicRoot rev-parse HEAD)-ne'8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'){throw 'Public authority checkout is not at 8c4bbe8.'}

function Resolve-OnePackage([string]$Pattern){$matches=@(Get-ChildItem -LiteralPath $PackageRoot -Filter $Pattern -File);if($matches.Count-ne1){throw "Expected one package matching $Pattern, observed $($matches.Count)."};$matches[0]}
function Expand-Package([IO.FileInfo]$Archive,[string]$Destination){New-Item -ItemType Directory -Force $Destination|Out-Null;[IO.Compression.ZipFile]::ExtractToDirectory($Archive.FullName,$Destination);$manifest=Get-Content (Join-Path $Destination protocol-lab-package.json)-Raw|ConvertFrom-Json;if($manifest.schemaVersion-ne'protocol-lab-package-v2'){throw "$($Archive.Name) is not package-v2."};$manifest}
function Set-CommonEnvironment([string]$EvidenceRoot,[string]$ScenarioHash,[string]$ExecutorHash,[string]$TargetHash){
    $env:PLAB_SCENARIO_ID='tls.key-update.diagnostic';$env:PLAB_TARGET_BASE_URL='tls://127.0.0.1:18460';$env:PLAB_ARTIFACT_DIR=$EvidenceRoot
    $env:PLAB_EXECUTOR_ID='openssl-tls13-key-update-executor';$env:PLAB_EXECUTOR_VERSION='0.1.0';$env:PLAB_LOAD_GENERATOR_ID='openssl-tls13-key-update-load';$env:PLAB_LOAD_GENERATOR_VERSION='0.1.0'
    $env:PLAB_PROTOCOL='tls';$env:PLAB_PROTOCOL_VARIANT='tls1.3-key-update';$env:PLAB_LOAD_PROFILE_ID='tls-diagnostic';$env:PLAB_RUN_ID="local-key-update-$RuntimeIdentifier";$env:PLAB_CELL_ID="local-key-update-$RuntimeIdentifier-cell"
    $env:PLAB_RUN_PLAN_SHA256='edfd29de78f98f6873975db194321e08776d5ed58b52b6e26b15db249678b196';$env:PLAB_SCENARIO_PACKAGE_SHA256=$ScenarioHash;$env:PLAB_EXECUTOR_PACKAGE_SHA256=$ExecutorHash;$env:PLAB_IMPLEMENTATION_PACKAGE_SHA256=$TargetHash
}

if(Test-Path $ArtifactRoot){Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force}
New-Item -ItemType Directory -Force $ArtifactRoot|Out-Null
$scenarioArchive=Resolve-OnePackage 'org.protocol-lab.components.scenario.tls13-handshake-performance.0.2.0.plabpkg'
$executorArchive=Resolve-OnePackage "org.protocol-lab.components.executor.openssl-tls13-key-update-executor.0.1.0.$RuntimeIdentifier.plabpkg"
$targetArchive=Resolve-OnePackage "org.protocol-lab.components.implementation.openssl-tls13-key-update.0.1.0.$RuntimeIdentifier.plabpkg"
$scenarioRoot=Join-Path $ArtifactRoot scenario;$executorRoot=Join-Path $ArtifactRoot executor;$targetRoot=Join-Path $ArtifactRoot target
$scenarioManifest=Expand-Package $scenarioArchive $scenarioRoot;$executorManifest=Expand-Package $executorArchive $executorRoot;$targetManifest=Expand-Package $targetArchive $targetRoot
$scenarioHash=(Get-FileHash $scenarioArchive.FullName -Algorithm SHA256).Hash.ToLowerInvariant();$executorHash=(Get-FileHash $executorArchive.FullName -Algorithm SHA256).Hash.ToLowerInvariant();$targetHash=(Get-FileHash $targetArchive.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
$authority=Get-Content (Join-Path $scenarioRoot authority-lock.json)-Raw|ConvertFrom-Json
if($authority.commit-ne'8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'-or$authority.files.'scenarios/tls/diagnostic/key-update.yaml'-ne'10d9324acc0437fbefc6a3b29cb68937f45f52ff058d7a2111025180235ad9e7'-or$authority.files.'load-profiles/tls-diagnostic.yaml'-ne'2bd9c53844fe77990a8b9888e5e260ea6979b193ef7537aba5b24b40f8253599'-or$authority.files.'fixtures/public-contracts/tls-profile-v2/valid/tls13-aes128gcm-p256-server-auth-v2.json'-ne'6da293573d86a3521b9fe8d768d757f0a67a14d55859211ed43154a72fddbf46'){throw 'KeyUpdate authority lock mismatch.'}
if($executorManifest.providedTestExecutors[0].scenarios.Count-ne1-or$executorManifest.providedTestExecutors[0].scenarios[0]-ne'tls.key-update.diagnostic'-or$targetManifest.providedImplementations[0].scenarios.Count-ne1-or$targetManifest.providedImplementations[0].scenarios[0]-ne'tls.key-update.diagnostic'){throw 'Exact package claims mismatch.'}
foreach($root in @($executorRoot,$targetRoot)){if(-not(Test-Path (Join-Path $root 'third-party-licenses/openssl/LICENSE-APACHE-2.0.txt'))){throw 'OpenSSL license material is missing.'}}
if(Test-Path (Join-Path $executorRoot 'certs/leaf-key.pem')){throw 'Server private key leaked into executor package.'};if(Test-Path (Join-Path $targetRoot 'certs/root.pem')){throw 'Trust anchor leaked into target package.'}

$cellRoot=Join-Path $ArtifactRoot cell;New-Item -ItemType Directory -Force $cellRoot|Out-Null
if($RuntimeIdentifier-eq'win-x64'){
    $saved=@{};$names=@('PLAB_LISTEN_ADDRESS','PLAB_SCENARIO_ID','PLAB_TLS_CERT_FILE','PLAB_TLS_KEY_FILE','PLAB_TARGET_BASE_URL','PLAB_ARTIFACT_DIR','PLAB_TLS_ROOT_CERTIFICATE_PATH','PLAB_EXECUTOR_ID','PLAB_EXECUTOR_VERSION','PLAB_LOAD_GENERATOR_ID','PLAB_LOAD_GENERATOR_VERSION','PLAB_PROTOCOL','PLAB_PROTOCOL_VARIANT','PLAB_LOAD_PROFILE_ID','PLAB_RUN_ID','PLAB_CELL_ID','PLAB_RUN_PLAN_SHA256','PLAB_SCENARIO_PACKAGE_SHA256','PLAB_EXECUTOR_PACKAGE_SHA256','PLAB_IMPLEMENTATION_PACKAGE_SHA256');foreach($name in $names){$saved[$name]=[Environment]::GetEnvironmentVariable($name,'Process')}
    $target=$null
    try{
        $env:PLAB_LISTEN_ADDRESS='127.0.0.1:18460';$env:PLAB_SCENARIO_ID='tls.key-update.diagnostic';$env:PLAB_TLS_CERT_FILE=Join-Path $targetRoot 'certs/leaf.pem';$env:PLAB_TLS_KEY_FILE=Join-Path $targetRoot 'certs/leaf-key.pem'
        $target=Start-Process -FilePath (Join-Path $targetRoot 'bin/win-x64/openssl-tls13-key-update.exe') -RedirectStandardOutput (Join-Path $cellRoot target.stdout.log) -RedirectStandardError (Join-Path $cellRoot target.stderr.log) -WindowStyle Hidden -PassThru
        $ready=$false;for($attempt=0;$attempt-lt100;$attempt++){if((Test-Path (Join-Path $cellRoot target.stdout.log))-and((Get-Content (Join-Path $cellRoot target.stdout.log)-Raw)-match'"eventName":"ready"')){$ready=$true;break};Start-Sleep -Milliseconds 50};if(-not$ready){throw 'Extracted KeyUpdate target did not become ready.'}
        Set-CommonEnvironment $cellRoot $scenarioHash $executorHash $targetHash;$env:PLAB_TLS_ROOT_CERTIFICATE_PATH=Join-Path $executorRoot 'certs/root.pem'
        $run=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/openssl-tls13-key-update-executor.exe') -RedirectStandardOutput (Join-Path $cellRoot load.stdout.log) -RedirectStandardError (Join-Path $cellRoot load.stderr.log) -WindowStyle Hidden -Wait -PassThru
        if($run.ExitCode-ne0){throw "Extracted KeyUpdate executor failed with $($run.ExitCode): $(Get-Content (Join-Path $cellRoot load.stderr.log)-Raw)"}
        $target.WaitForExit(5000)|Out-Null;if(-not$target.HasExited-or$target.ExitCode-ne0){throw 'Extracted KeyUpdate target did not exit successfully.'}
    }finally{if($null-ne$target-and-not$target.HasExited){Stop-Process -Id $target.Id -Force};foreach($name in $names){[Environment]::SetEnvironmentVariable($name,$saved[$name],'Process')}}
}else{
    $mountTarget=$targetRoot.Replace('\','/');$mountExecutor=$executorRoot.Replace('\','/');$mountEvidence=$cellRoot.Replace('\','/')
    $command=@'
export PLAB_LISTEN_ADDRESS=127.0.0.1:18460 PLAB_SCENARIO_ID=tls.key-update.diagnostic PLAB_TLS_CERT_FILE=/target/certs/leaf.pem PLAB_TLS_KEY_FILE=/target/certs/leaf-key.pem
/target/bin/linux-x64/openssl-tls13-key-update > /evidence/target.stdout.log 2> /evidence/target.stderr.log &
pid=$!
for i in $(seq 1 100); do grep -q '"eventName":"ready"' /evidence/target.stdout.log 2>/dev/null && break; sleep 0.05; done
export PLAB_TARGET_BASE_URL=tls://127.0.0.1:18460 PLAB_ARTIFACT_DIR=/evidence PLAB_TLS_ROOT_CERTIFICATE_PATH=/executor/certs/root.pem
export PLAB_EXECUTOR_ID=openssl-tls13-key-update-executor PLAB_EXECUTOR_VERSION=0.1.0 PLAB_LOAD_GENERATOR_ID=openssl-tls13-key-update-load PLAB_LOAD_GENERATOR_VERSION=0.1.0
export PLAB_PROTOCOL=tls PLAB_PROTOCOL_VARIANT=tls1.3-key-update PLAB_LOAD_PROFILE_ID=tls-diagnostic PLAB_RUN_ID=local-key-update-linux-x64 PLAB_CELL_ID=local-key-update-linux-x64-cell
export PLAB_RUN_PLAN_SHA256=edfd29de78f98f6873975db194321e08776d5ed58b52b6e26b15db249678b196 PLAB_SCENARIO_PACKAGE_SHA256={0} PLAB_EXECUTOR_PACKAGE_SHA256={1} PLAB_IMPLEMENTATION_PACKAGE_SHA256={2}
/executor/bin/linux-x64/openssl-tls13-key-update-executor > /evidence/load.stdout.log 2> /evidence/load.stderr.log
code=$?
wait $pid
targetCode=$?
test $code -eq 0 -a $targetCode -eq 0
'@ -f $scenarioHash,$executorHash,$targetHash
    & docker run --rm -v "${mountTarget}:/target:ro" -v "${mountExecutor}:/executor:ro" -v "${mountEvidence}:/evidence" $LinuxImage sh -lc $command
    if($LASTEXITCODE-ne0){throw "Extracted Linux KeyUpdate smoke failed: $(Get-Content (Join-Path $cellRoot load.stderr.log)-Raw) $(Get-Content (Join-Path $cellRoot target.stderr.log)-Raw)"}
}

foreach($required in @('validation.json','protocol-proof.json','tls-negotiation.json','key-update-proof.json','payload-hash.json','result.json','tls-executor-result.json','protocol-execution-result-v2.json','executor-identity.json','load-generator-identity.json','load.stdout.log','load.stderr.log','target.stdout.log','target.stderr.log')){if(-not(Test-Path (Join-Path $cellRoot $required))){throw "Required KeyUpdate artifact missing: $required"}}
$proof=Get-Content (Join-Path $cellRoot key-update-proof.json)-Raw|ConvertFrom-Json;$result=Get-Content (Join-Path $cellRoot tls-executor-result.json)-Raw|ConvertFrom-Json;$v2=Get-Content (Join-Path $cellRoot protocol-execution-result-v2.json)-Raw|ConvertFrom-Json
if($proof.initiator-ne'client'-or$proof.requestedUpdates-ne1-or$proof.peerUpdateRequested-or$proof.clientMessageCallbackSentCount-ne1-or$proof.clientMessageCallbackReceivedCount-ne0-or$proof.targetAcknowledgedReceivedCount-ne1-or$proof.keyUpdateRequestByte-ne0-or$proof.clientWriteGenerationAfter-ne1-or$proof.serverReadGenerationAfter-ne1-or$proof.serverWriteGenerationAfter-ne0-or-not$proof.postUpdateDataComplete-or$proof.trafficSecretsPublished){throw 'Exact KeyUpdate proof mismatch.'}
if($proof.postUpdateBytesClientToServer-ne65536-or$proof.postUpdateBytesServerToClient-ne65536-or$result.metrics.completedOperations-ne1-or$result.metrics.failedOperations-ne0-or$result.metrics.timedOutOperations-ne0-or$result.metrics.totalTransferredBytes-ne131072){throw 'KeyUpdate metrics or deterministic transfer mismatch.'}
if($v2.schemaVersion-ne'protocol-lab.protocol-execution-result.v2'-or$v2.familyEvidence.keyUpdateCount-ne1-or$v2.familyEvidence.payloadSha256-ne'944044fe482bc4e91085c15c5a923a1b9e02eac98d3bce04997d6dbecd2a5b8d'-or$v2.familyEvidence.resumption-ne'not-offered'-or$v2.familyEvidence.earlyData-ne'not-attempted'-or$v2.outcome-ne'completed'){throw 'Protocol Execution Result v2 input mismatch.'}
$targetProof=((Get-Content (Join-Path $cellRoot target.stdout.log))|Where-Object{$_-match'"eventName":"target-proof"'}|Select-Object -Last 1)|ConvertFrom-Json
if($targetProof.keyUpdateMessagesReceived-ne1-or$targetProof.keyUpdateMessagesSent-ne0-or$targetProof.peerUpdateRequested-or$targetProof.postUpdateBytesReceived-ne65536-or$targetProof.postUpdateBytesSent-ne65536-or$targetProof.trafficSecretsPublished){throw 'Target-side KeyUpdate proof mismatch.'}
$schemaPath=Join-Path $PublicRoot 'schemas/measurement/v2/protocol-execution-result.schema.json';$v2Path=Join-Path $cellRoot protocol-execution-result-v2.json
python -c "import json,jsonschema,sys; schema=json.load(open(sys.argv[1],encoding='utf-8')); value=json.load(open(sys.argv[2],encoding='utf-8')); jsonschema.Draft202012Validator(schema,format_checker=jsonschema.FormatChecker()).validate(value)" $schemaPath $v2Path
if($LASTEXITCODE-ne0){throw 'Protocol Execution Result v2 schema validation failed.'}
if((Get-ChildItem $cellRoot -File|Select-String -Pattern 'CLIENT_TRAFFIC_SECRET|SERVER_TRAFFIC_SECRET|CLIENT_HANDSHAKE_TRAFFIC_SECRET|SERVER_HANDSHAKE_TRAFFIC_SECRET' -SimpleMatch).Count-ne0){throw 'Traffic-secret material was published.'}

$unsupported=@('tls.handshake.full','tls.handshake.resumed','tls.handshake.full.tls12','tls.handshake.full.chacha20','tls.handshake.mutual-auth','tls.early-data.accepted','tls.early-data.rejected','tls.record.coverage','tls.record.throughput')
if($RuntimeIdentifier-eq'win-x64'){
    foreach($id in $unsupported){$root=Join-Path $ArtifactRoot ('unsupported-'+($id-replace'\.','-'));New-Item -ItemType Directory -Force $root|Out-Null;$env:PLAB_SCENARIO_ID=$id;$env:PLAB_ARTIFACT_DIR=$root;$run=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/openssl-tls13-key-update-executor.exe') -RedirectStandardOutput (Join-Path $root load.stdout.log) -RedirectStandardError (Join-Path $root load.stderr.log) -WindowStyle Hidden -Wait -PassThru;if($run.ExitCode-ne3){throw "$id did not exit unsupported."};$evidence=Get-Content (Join-Path $root unsupported.json)-Raw|ConvertFrom-Json;if($evidence.status-ne'unsupported'-or$evidence.scenarioId-ne$id){throw "$id unsupported evidence mismatch."}}
    $unknownRoot=Join-Path $ArtifactRoot unknown;New-Item -ItemType Directory -Force $unknownRoot|Out-Null;$env:PLAB_SCENARIO_ID='tls.key-update.unknown';$env:PLAB_ARTIFACT_DIR=$unknownRoot;$unknown=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/openssl-tls13-key-update-executor.exe') -RedirectStandardOutput (Join-Path $unknownRoot load.stdout.log) -RedirectStandardError (Join-Path $unknownRoot load.stderr.log) -WindowStyle Hidden -Wait -PassThru;if($unknown.ExitCode-ne2-or(Test-Path (Join-Path $unknownRoot unsupported.json))){throw 'Unknown TLS identity did not remain distinct.'}
}

[ordered]@{authorityCommit=$authority.commit;runtimeIdentifier=$RuntimeIdentifier;scenarioPackageSha256=$scenarioHash;executorPackageSha256=$executorHash;targetPackageSha256=$targetHash;completedOperations=1;failedOperations=0;timedOutOperations=0;totalTransferredBytes=131072;keyUpdateRequested=1;keyUpdateObservedByTarget=1;peerUpdateRequested=$false;postUpdateBytesPerDirection=65536;trafficSecretsPublished=$false;unsupportedScenarioIds=$unsupported;evidenceRoot=$ArtifactRoot}|ConvertTo-Json -Depth 6
