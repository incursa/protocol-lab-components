[CmdletBinding()]
param(
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot=(Join-Path $Root 'artifacts/dns-classic-smoke-packages'),
    [string]$SmokeRoot=(Join-Path $Root 'artifacts/dns-classic-three-package-smoke'),
    [switch]$SkipBuild,
    [switch]$AllowDirtySource
)
$ErrorActionPreference='Stop'
if(-not[IO.Path]::IsPathRooted($OutputRoot)){$OutputRoot=Join-Path $Root $OutputRoot};if(-not[IO.Path]::IsPathRooted($SmokeRoot)){$SmokeRoot=Join-Path $Root $SmokeRoot}
Add-Type -AssemblyName System.IO.Compression.FileSystem
function Expand-Package([string]$Archive,[string]$Destination){if(Test-Path $Destination){Remove-Item $Destination -Recurse -Force};New-Item -ItemType Directory -Force $Destination|Out-Null;[IO.Compression.ZipFile]::ExtractToDirectory($Archive,$Destination);$manifest=Get-Content (Join-Path $Destination 'protocol-lab-package.json') -Raw|ConvertFrom-Json;if($manifest.schemaVersion-ne'protocol-lab-package-v2'){throw "$Archive is not package-v2"};return $manifest}
function Get-Archive([string]$Pattern){$matches=@(Get-ChildItem $OutputRoot -File -Filter $Pattern);if($matches.Count-ne 1){throw "Expected one $Pattern archive, found $($matches.Count)."};return $matches[0].FullName}
if(-not $SkipBuild){& (Join-Path $PSScriptRoot 'Build-DnsClassicCalibrationScenarioPackage.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource;& (Join-Path $PSScriptRoot 'Build-GoDnsClassicAuthorityPackage.ps1') win-x64 -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource;& (Join-Path $PSScriptRoot 'Build-GoDnsUdpExecutorPackage.ps1') win-x64 -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource;& (Join-Path $PSScriptRoot 'Build-GoDnsTcpExecutorPackage.ps1') win-x64 -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource}
if(Test-Path $SmokeRoot){Remove-Item $SmokeRoot -Recurse -Force};New-Item -ItemType Directory -Force $SmokeRoot|Out-Null
$scenarioRoot=Join-Path $SmokeRoot 'scenario';$targetRoot=Join-Path $SmokeRoot 'target';$udpRoot=Join-Path $SmokeRoot 'udp-executor';$tcpRoot=Join-Path $SmokeRoot 'tcp-executor'
$scenarioManifest=Expand-Package (Get-Archive 'org.protocol-lab.components.scenario.dns-classic-calibration.0.1.0.plabpkg') $scenarioRoot
$targetManifest=Expand-Package (Get-Archive 'org.protocol-lab.components.implementation.go-dns-classic-authority.0.1.0.win-x64.plabpkg') $targetRoot
$udpManifest=Expand-Package (Get-Archive 'org.protocol-lab.components.executor.go-dns-udp-executor.0.1.0.win-x64.plabpkg') $udpRoot
$tcpManifest=Expand-Package (Get-Archive 'org.protocol-lab.components.executor.go-dns-tcp-executor.0.1.0.win-x64.plabpkg') $tcpRoot
if($scenarioManifest.packageId-ne'org.protocol-lab.components.scenario.dns-classic-calibration'-or$targetManifest.packageId-ne'org.protocol-lab.components.implementation.go-dns-classic-authority'-or$udpManifest.packageId-ne'org.protocol-lab.components.executor.go-dns-udp-executor'-or$tcpManifest.packageId-ne'org.protocol-lab.components.executor.go-dns-tcp-executor'){throw 'Package identity mismatch.'}
$port=15373;$env:PLAB_DNS_CLASSIC_PORT=[string]$port;$targetOut=Join-Path $SmokeRoot 'target.stdout.log';$targetErr=Join-Path $SmokeRoot 'target.stderr.log';$targetExe=Join-Path $targetRoot 'bin/win-x64/go-dns-classic-authority.exe'
$target=Start-Process -FilePath $targetExe -WorkingDirectory $targetRoot -RedirectStandardOutput $targetOut -RedirectStandardError $targetErr -WindowStyle Hidden -PassThru
try{
    $ready=$false;for($i=0;$i-lt 40;$i++){Start-Sleep -Milliseconds 100;if($target.HasExited){throw "Target exited $($target.ExitCode): $(Get-Content $targetErr -Raw)"};if((Test-Path $targetOut)-and((Get-Content $targetOut -Raw)-match'"status":"ready"')){$ready=$true;break}}
    if(-not $ready){throw 'Classic DNS target did not report readiness.'}
    $cells=@(
        @{scenario='dns.classic.udp.query.a';executor='udp';id='go-dns-udp-executor';generator='go-dns-udp-load';target="udp://127.0.0.1:$port"},
        @{scenario='dns.classic.tcp.query.a';executor='tcp';id='go-dns-tcp-executor';generator='go-dns-tcp-load';target="tcp://127.0.0.1:$port"},
        @{scenario='dns.classic.udp-truncated-tcp-retry';executor='udp';id='go-dns-udp-executor';generator='go-dns-udp-load';target="udp://127.0.0.1:$port"}
    )
    $summaries=@()
    foreach($cell in $cells){
        $cellRoot=Join-Path $SmokeRoot $cell.scenario;New-Item -ItemType Directory -Force $cellRoot|Out-Null
        $env:PLAB_SCENARIO_ID=$cell.scenario;$env:PLAB_LOAD_PROFILE_ID='dns-classic-diagnostic';$env:PLAB_EXECUTOR_ID=$cell.id;$env:PLAB_EXECUTOR_VERSION='0.1.0';$env:PLAB_LOAD_GENERATOR_ID=$cell.generator;$env:PLAB_LOAD_GENERATOR_VERSION='0.1.0';$env:PLAB_CONNECTIONS='1';$env:PLAB_CONCURRENCY='1';$env:PLAB_DURATION_SECONDS='5';$env:PLAB_WARMUP_SECONDS='1';$env:PLAB_REPETITION='1'
        $exe=if($cell.executor-eq'udp'){Join-Path $udpRoot 'bin/win-x64/go-dns-udp-executor.exe'}else{Join-Path $tcpRoot 'bin/win-x64/go-dns-tcp-executor.exe'}
        $stdout=Join-Path $cellRoot 'load.stdout.log';$stderr=Join-Path $cellRoot 'load.stderr.log';& $exe --target-address $cell.target --output-dir $cellRoot 1> $stdout 2> $stderr;if($LASTEXITCODE-ne 0){throw "$($cell.scenario) executor exited $LASTEXITCODE`: $(Get-Content $stderr -Raw)"}
        $result=Get-Content (Join-Path $cellRoot 'result.json') -Raw|ConvertFrom-Json;if($result.status-ne'passed'-or$result.scenarioId-ne$cell.scenario-or$result.metrics.completedOperations-le 0-or$result.metrics.malformedOperations-ne 0-or$result.metrics.failedOperations-ne 0-or$result.metrics.timedOutOperations-ne 0){throw "$($cell.scenario) outcome gate failed."}
        $dns=$result.protocolProof.dns;if($dns.canonicalHashNormalization-ne'set-message-id-to-zero'-or-not $dns.messageIdCorrelated-or-not $dns.messageIdUniqueAmongOutstanding){throw "$($cell.scenario) ID normalization gate failed."}
        if($cell.scenario-eq'dns.classic.udp-truncated-tcp-retry'){if(-not $dns.udpTruncated-or$dns.udpAdvertisedPayloadBytes-ne 512-or$dns.udpTruncatedResponseLength-ne 45-or$dns.retryCount-ne 1-or-not $dns.retryQuestionIdentical-or-not $dns.retryMessageIdNew-or$dns.tcpResponseLength-ne 630-or$dns.tcpResponsePrefixHex-ne'0276'){throw 'Retry proof gate failed.'}}
        foreach($name in @('validation.json','protocol-proof.json','dns-wire-summary.json','result.json','dns-classic-executor-result.json','dns-load-summary.json','dns-warmup-summary.json','executor-identity.json','load-generator-identity.json','load.stdout.log','load.stderr.log')){if(-not(Test-Path (Join-Path $cellRoot $name))){throw "$($cell.scenario) missing $name"}}
        $summaries+=@{scenarioId=$cell.scenario;completedOperations=$result.metrics.completedOperations;malformedOperations=$result.metrics.malformedOperations;retryCount=$result.metrics.retryCount;failedOperations=$result.metrics.failedOperations;timedOutOperations=$result.metrics.timedOutOperations;artifactRoot=$cellRoot}
    }
    $allDns=@('dns.dot.query.a','dns.doh2.query.a','dns.doh3.get.a','dns.doh3.query.a','dns.doh3.query.aaaa','dns.doh3.query.cname-chain','dns.doh3.query.large-dnssec-shaped','dns.doh3.query.nodata','dns.doh3.query.nxdomain','dns.doq.query.a','dns.classic.udp.query.a','dns.classic.tcp.query.a','dns.classic.udp-truncated-tcp-retry')
    foreach($executor in @(@{root=$udpRoot;exe='go-dns-udp-executor.exe';id='go-dns-udp-executor';generator='go-dns-udp-load';target="udp://127.0.0.1:$port";supported=@('dns.classic.udp.query.a','dns.classic.udp-truncated-tcp-retry')},@{root=$tcpRoot;exe='go-dns-tcp-executor.exe';id='go-dns-tcp-executor';generator='go-dns-tcp-load';target="tcp://127.0.0.1:$port";supported=@('dns.classic.tcp.query.a')})){
        foreach($unsupported in @($allDns|Where-Object{$executor.supported -notcontains $_})){$unsupportedRoot=Join-Path $SmokeRoot ("unsupported/"+$executor.id+"/"+$unsupported);New-Item -ItemType Directory -Force $unsupportedRoot|Out-Null;$env:PLAB_SCENARIO_ID=$unsupported;$env:PLAB_EXECUTOR_ID=$executor.id;$env:PLAB_EXECUTOR_VERSION='0.1.0';$env:PLAB_LOAD_GENERATOR_ID=$executor.generator;$env:PLAB_LOAD_GENERATOR_VERSION='0.1.0';& (Join-Path $executor.root ("bin/win-x64/"+$executor.exe)) --target-address $executor.target --output-dir $unsupportedRoot *> $null;if($LASTEXITCODE-ne 3){throw "$($executor.id) $unsupported exit=$LASTEXITCODE"};$unsupportedResult=Get-Content (Join-Path $unsupportedRoot 'result.json') -Raw|ConvertFrom-Json;if($unsupportedResult.status-ne'unsupported'-or$unsupportedResult.scenarioId-ne$unsupported){throw "$($executor.id) $unsupported did not fail closed."}}
        $unknownRoot=Join-Path $SmokeRoot ("unknown/"+$executor.id);New-Item -ItemType Directory -Force $unknownRoot|Out-Null;$env:PLAB_SCENARIO_ID='dns.unknown.identity';& (Join-Path $executor.root ("bin/win-x64/"+$executor.exe)) --target-address $executor.target --output-dir $unknownRoot *> $null;if($LASTEXITCODE-ne 2){throw "$($executor.id) unknown identity exit=$LASTEXITCODE"}
    }
    $summaries|ConvertTo-Json -Depth 10|Set-Content (Join-Path $SmokeRoot 'smoke-summary.json') -Encoding utf8NoBOM
    $summaries|ForEach-Object{Write-Host "$($_.scenarioId): completed=$($_.completedOperations) malformed=$($_.malformedOperations) retries=$($_.retryCount) failed=$($_.failedOperations) timedOut=$($_.timedOutOperations)"}
}
finally{if($target-and-not $target.HasExited){Stop-Process -Id $target.Id -Force};Remove-Item Env:PLAB_DNS_CLASSIC_PORT -ErrorAction SilentlyContinue}
