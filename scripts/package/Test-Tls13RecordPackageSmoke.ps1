[CmdletBinding()]
param(
    [string]$PackageRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-record-packages'),
    [string]$ArtifactRoot=(Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-record-extracted-smoke')
)

$ErrorActionPreference='Stop'
$PackageRoot=[IO.Path]::GetFullPath($PackageRoot)
$ArtifactRoot=[IO.Path]::GetFullPath($ArtifactRoot)
$repoRoot=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
if(-not $ArtifactRoot.StartsWith([IO.Path]::GetFullPath((Join-Path $repoRoot 'artifacts')),[StringComparison]::OrdinalIgnoreCase)){throw 'TLS record smoke artifacts must remain under this worktree artifacts directory.'}

function Resolve-OnePackage([string]$Pattern){$matches=@(Get-ChildItem -LiteralPath $PackageRoot -Filter $Pattern -File);if($matches.Count-ne 1){throw "Expected one package matching $Pattern, observed $($matches.Count)."};return $matches[0].FullName}
function Expand-Package([string]$Archive,[string]$Destination){New-Item -ItemType Directory -Force $Destination|Out-Null;[IO.Compression.ZipFile]::ExtractToDirectory($Archive,$Destination);$m=Get-Content (Join-Path $Destination 'protocol-lab-package.json') -Raw|ConvertFrom-Json;if($m.schemaVersion-ne 'protocol-lab-package-v2'){throw "$Archive is not package-v2."};return $m}

if(Test-Path $ArtifactRoot){Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force}
New-Item -ItemType Directory -Force $ArtifactRoot|Out-Null
$scenarioArchive=Resolve-OnePackage 'org.protocol-lab.components.scenario.tls13-handshake-performance.0.2.0.plabpkg'
$executorArchive=Resolve-OnePackage 'org.protocol-lab.components.executor.go-tls13-executor.0.3.0.win-x64.plabpkg'
$targetArchive=Resolve-OnePackage 'org.protocol-lab.components.implementation.go-tls13.0.2.0.win-x64.plabpkg'
$scenarioRoot=Join-Path $ArtifactRoot scenario;$executorRoot=Join-Path $ArtifactRoot executor;$targetRoot=Join-Path $ArtifactRoot target
$scenarioManifest=Expand-Package $scenarioArchive $scenarioRoot;$executorManifest=Expand-Package $executorArchive $executorRoot;$targetManifest=Expand-Package $targetArchive $targetRoot
$authority=Get-Content (Join-Path $scenarioRoot 'authority-lock.json') -Raw|ConvertFrom-Json
if($authority.commit-ne '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574'){throw 'Authority commit mismatch.'}
foreach($id in @('tls.record.throughput','tls.record.coverage')){if($executorManifest.providedTestExecutors[0].scenarios-notcontains $id){throw "Executor missing $id."};if($targetManifest.providedImplementations[0].scenarios-notcontains $id){throw "Target missing $id."}}

$results=@()
foreach($cell in @(
    [pscustomobject]@{scenario='tls.record.throughput';variant='tls1.3-record';profile='tls-smoke';port=18443;duration=5;warmup=1;timeout=5;total=$null},
    [pscustomobject]@{scenario='tls.record.coverage';variant='tls1.3-record-coverage';profile='tls-diagnostic';port=18444;duration=10;warmup=0;timeout=15;total=1}
)){
    $cellRoot=Join-Path $ArtifactRoot ($cell.scenario-replace '\.','-');New-Item -ItemType Directory -Force $cellRoot|Out-Null
    $env:PLAB_LISTEN_ADDRESS="127.0.0.1:$($cell.port)";$env:PLAB_SCENARIO_ID=$cell.scenario
    $env:PLAB_TLS_CERT_FILE=Join-Path $targetRoot 'certs/leaf.pem';$env:PLAB_TLS_KEY_FILE=Join-Path $targetRoot 'certs/leaf-key.pem'
    $target=Start-Process -FilePath (Join-Path $targetRoot 'bin/win-x64/go-tls13.exe') -RedirectStandardOutput (Join-Path $cellRoot 'target.stdout.log') -RedirectStandardError (Join-Path $cellRoot 'target.stderr.log') -WindowStyle Hidden -PassThru
    try{
        $targetStdout=Join-Path $cellRoot 'target.stdout.log'
        $ready=$false;for($attempt=0;$attempt-lt 50;$attempt++){if((Test-Path $targetStdout)-and((Get-Content $targetStdout -Raw)-match '"eventName":"ready"')){$ready=$true;break};Start-Sleep -Milliseconds 100}
        if(-not $ready){throw "Target did not become ready for $($cell.scenario)."}
        $executorArtifacts=Join-Path $cellRoot 'executor-artifacts';New-Item -ItemType Directory -Force $executorArtifacts|Out-Null
        $env:PLAB_TARGET_BASE_URL="tls://127.0.0.1:$($cell.port)";$env:PLAB_ARTIFACT_DIR=$executorArtifacts;$env:PLAB_TLS_ROOT_CERTIFICATE_PATH=Join-Path $executorRoot 'certs/root.pem'
        $env:PLAB_EXECUTOR_ID='go-tls13-executor';$env:PLAB_EXECUTOR_VERSION='0.3.0';$env:PLAB_LOAD_GENERATOR_ID='go-crypto-tls13-load';$env:PLAB_LOAD_GENERATOR_VERSION='0.3.0'
        $env:PLAB_PROTOCOL='tls';$env:PLAB_PROTOCOL_VARIANT=$cell.variant;$env:PLAB_LOAD_PROFILE_ID=$cell.profile;$env:PLAB_CONNECTIONS='1';$env:PLAB_CONCURRENCY='1';$env:PLAB_DURATION_SECONDS=[string]$cell.duration;$env:PLAB_WARMUP_SECONDS=[string]$cell.warmup;$env:PLAB_REPETITION='1';$env:PLAB_REQUEST_TIMEOUT_SECONDS=[string]$cell.timeout
        if($null-ne $cell.total){$env:PLAB_TOTAL_OPERATIONS=[string]$cell.total}else{Remove-Item Env:PLAB_TOTAL_OPERATIONS -ErrorAction SilentlyContinue}
        $execution=Start-Process -FilePath (Join-Path $executorRoot 'bin/win-x64/go-tls13-executor.exe') -RedirectStandardOutput (Join-Path $executorArtifacts 'load.stdout.log') -RedirectStandardError (Join-Path $executorArtifacts 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
        if($execution.ExitCode-ne 0){throw "$($cell.scenario) executor failed with exit code $($execution.ExitCode)."}
        $result=Get-Content (Join-Path $executorArtifacts 'tls-executor-result.json') -Raw|ConvertFrom-Json
        if($result.scenarioId-ne $cell.scenario-or$result.protocolProof.tlsVersion-ne 'TLS1.3'-or$result.protocolProof.alpn-ne 'protocol-lab-tls'-or$result.protocolProof.keyExchangeGroup-ne 'X25519'-or$result.protocolProof.cipherSuite-ne 'TLS_AES_128_GCM_SHA256'-or$result.protocolProof.didResume-or$result.protocolProof.earlyDataAttempted-or$result.metrics.completedOperations-lt 1-or$result.metrics.failedOperations-ne 0-or$result.metrics.timedOutOperations-ne 0){throw "$($cell.scenario) exact validity gate failed."}
        if($cell.scenario-eq 'tls.record.coverage'){
            if($result.protocolProof.tlsProfileId-ne 'plab-tls13-aes128gcm-p256-server-auth-v2'-or$result.protocolProof.certificateProfile-ne 'plab-single-leaf-p256-server-v2'){throw 'Coverage TLS v2 profile identity mismatch.'}
        }elseif($result.protocolProof.tlsProfileId-ne 'plab-tls13-p256-v1'-or$result.protocolProof.certificateProfile-ne 'plab-single-leaf-p256-v1'){throw 'Throughput TLS v1 profile identity mismatch.'}
        $payload=Get-Content (Join-Path $executorArtifacts 'payload-hash.json') -Raw|ConvertFrom-Json
        if($cell.scenario-eq 'tls.record.throughput'){if($payload.cases.Count-ne 1-or$payload.cases[0].applicationDataBytes-ne 1048576-or$payload.cases[0].payloadSha256-ne 'bf63d8a95fcc2e64619813aae35fdcbe871fdd9264caa3f365eb3aed0f679129'){throw 'Throughput payload proof mismatch.'}}
        else{$coverage=Get-Content (Join-Path $executorArtifacts 'record-coverage.json') -Raw|ConvertFrom-Json;if(-not $coverage.allSixCasesComplete-or$coverage.cases.Count-ne 6-or$result.metrics.totalTransferredBytes-ne 2230272){throw 'Coverage matrix proof mismatch.'}}
        $results+=[pscustomobject]@{scenarioId=$cell.scenario;completedOperations=[int]$result.metrics.completedOperations;failedOperations=[int]$result.metrics.failedOperations;timedOutOperations=[int]$result.metrics.timedOutOperations;totalTransferredBytes=[long]$result.metrics.totalTransferredBytes;evidenceRoot=$cellRoot}
    }finally{if($null-ne $target-and-not $target.HasExited){Stop-Process -Id $target.Id -Force}}
}
[pscustomobject]@{authorityCommit=$authority.commit;scenarioPackageSha256=(Get-FileHash $scenarioArchive -Algorithm SHA256).Hash.ToLowerInvariant();executorPackageSha256=(Get-FileHash $executorArchive -Algorithm SHA256).Hash.ToLowerInvariant();targetPackageSha256=(Get-FileHash $targetArchive -Algorithm SHA256).Hash.ToLowerInvariant();cells=$results}|ConvertTo-Json -Depth 8
