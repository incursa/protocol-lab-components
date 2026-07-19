[CmdletBinding()]
param(
    [string]$PackageRoot = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls13-chacha20-packages'),
    [string]$ArtifactRoot = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls13-chacha20-extracted-smoke')
)
$ErrorActionPreference = 'Stop'
$PackageRoot = [IO.Path]::GetFullPath($PackageRoot)
$ArtifactRoot = [IO.Path]::GetFullPath($ArtifactRoot)
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
if (-not $ArtifactRoot.StartsWith([IO.Path]::GetFullPath((Join-Path $repoRoot 'artifacts')), [StringComparison]::OrdinalIgnoreCase)) { throw 'ChaCha20 smoke artifacts must remain under this worktree artifacts directory.' }

function Resolve-OnePackage([string]$Pattern) {
    $matches = @(Get-ChildItem -LiteralPath $PackageRoot -Filter $Pattern -File)
    if ($matches.Count -ne 1) { throw "Expected one package matching $Pattern, observed $($matches.Count)." }
    return $matches[0].FullName
}
function Expand-Package([string]$Archive, [string]$Destination) {
    New-Item -ItemType Directory -Force $Destination | Out-Null
    [IO.Compression.ZipFile]::ExtractToDirectory($Archive, $Destination)
    $manifest = Get-Content (Join-Path $Destination 'protocol-lab-package.json') -Raw | ConvertFrom-Json
    if ($manifest.schemaVersion -ne 'protocol-lab-package-v2') { throw "$Archive is not package-v2." }
    return $manifest
}

if (Test-Path $ArtifactRoot) { Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force }
New-Item -ItemType Directory -Force $ArtifactRoot | Out-Null
$scenarioArchive = Resolve-OnePackage 'org.protocol-lab.components.scenario.tls13-handshake-performance.0.2.2.plabpkg'
$executorArchive = Resolve-OnePackage 'org.protocol-lab.components.executor.go-utls-tls13-chacha20-executor.0.1.0.win-x64.plabpkg'
$targetArchive = Resolve-OnePackage 'org.protocol-lab.components.implementation.go-tls13-chacha20.0.1.0.win-x64.plabpkg'
$scenarioRoot = Join-Path $ArtifactRoot scenario
$executorRoot = Join-Path $ArtifactRoot executor
$targetRoot = Join-Path $ArtifactRoot target
$null = Expand-Package $scenarioArchive $scenarioRoot
$executorManifest = Expand-Package $executorArchive $executorRoot
$targetManifest = Expand-Package $targetArchive $targetRoot
$authority = Get-Content (Join-Path $scenarioRoot 'authority-lock.json') -Raw | ConvertFrom-Json
if ($authority.commit -ne 'd5b78d7c07ef0e8a600e92887da2aa150ab89a60') { throw 'Authority commit mismatch.' }
if ($authority.files.'scenarios/tls/handshake/full-chacha20.yaml' -ne 'ca502190c313fcef152381f87d7a1e47be9dc998d1f49869e23eaa3444ef802f') { throw 'ChaCha20 scenario authority hash mismatch.' }
if ($executorManifest.providedTestExecutors[0].scenarios -notcontains 'tls.handshake.full.chacha20' -or $targetManifest.providedImplementations[0].scenarios -notcontains 'tls.handshake.full.chacha20') { throw 'Exact ChaCha20 package claim missing.' }
foreach ($license in @('utls-LICENSE.txt','brotli-LICENSE.txt','klauspost-compress-LICENSE.txt','golang-x-crypto-LICENSE.txt','golang-x-sys-LICENSE.txt')) {
    if (-not (Test-Path (Join-Path $executorRoot "third-party/$license") -PathType Leaf)) { throw "Third-party license missing: $license" }
}

$envNames = @('PLAB_LISTEN_ADDRESS','PLAB_SCENARIO_ID','PLAB_TLS_CERT_FILE','PLAB_TLS_KEY_FILE','PLAB_TARGET_BASE_URL','PLAB_ARTIFACT_DIR','PLAB_TLS_ROOT_CERTIFICATE_PATH','PLAB_EXECUTOR_ID','PLAB_EXECUTOR_VERSION','PLAB_LOAD_GENERATOR_ID','PLAB_LOAD_GENERATOR_VERSION','PLAB_PROTOCOL','PLAB_PROTOCOL_VARIANT','PLAB_LOAD_PROFILE_ID')
$saved = @{}
foreach ($name in $envNames) { $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
$target = $null
try {
    $port = 18447
    $env:PLAB_LISTEN_ADDRESS = "127.0.0.1:$port"
    $env:PLAB_SCENARIO_ID = 'tls.handshake.full.chacha20'
    $env:PLAB_TLS_CERT_FILE = Join-Path $targetRoot 'certs/leaf.pem'
    $env:PLAB_TLS_KEY_FILE = Join-Path $targetRoot 'certs/leaf-key.pem'
    $targetStdout = Join-Path $ArtifactRoot 'target.stdout.log'
    $targetStderr = Join-Path $ArtifactRoot 'target.stderr.log'
    $target = Start-Process -FilePath (Join-Path $targetRoot 'bin/win-x64/go-tls13-chacha20.exe') -RedirectStandardOutput $targetStdout -RedirectStandardError $targetStderr -WindowStyle Hidden -PassThru
    $ready = $false
    for ($attempt = 0; $attempt -lt 50; $attempt++) {
        if ((Test-Path $targetStdout) -and ((Get-Content $targetStdout -Raw) -match '"eventName":"ready"')) { $ready = $true; break }
        Start-Sleep -Milliseconds 100
    }
    if (-not $ready) { throw 'ChaCha20 target did not become ready.' }
    $targetReady = Get-Content $targetStdout -Raw | ConvertFrom-Json
    if ($targetReady.tlsVersion -ne 'TLS1.3' -or $targetReady.cipherSuite -ne 'TLS_CHACHA20_POLY1305_SHA256' -or $targetReady.keyExchangeGroup -ne 'X25519' -or $targetReady.alpn -ne 'protocol-lab-tls') { throw 'Target readiness identity mismatch.' }

    $executorArtifacts = Join-Path $ArtifactRoot 'executor-artifacts'
    New-Item -ItemType Directory -Force $executorArtifacts | Out-Null
    $env:PLAB_TARGET_BASE_URL = "tls://127.0.0.1:$port"
    $env:PLAB_ARTIFACT_DIR = $executorArtifacts
    $env:PLAB_TLS_ROOT_CERTIFICATE_PATH = Join-Path $executorRoot 'certs/root.pem'
    $env:PLAB_EXECUTOR_ID = 'go-utls-tls13-chacha20-executor'
    $env:PLAB_EXECUTOR_VERSION = '0.1.0'
    $env:PLAB_LOAD_GENERATOR_ID = 'go-utls-tls13-chacha20-load'
    $env:PLAB_LOAD_GENERATOR_VERSION = '0.1.0'
    $env:PLAB_PROTOCOL = 'tls'
    $env:PLAB_PROTOCOL_VARIANT = 'tls1.3-full-chacha20'
    $env:PLAB_LOAD_PROFILE_ID = 'tls-smoke'
    $executorBinary = Join-Path $executorRoot 'bin/win-x64/go-utls-tls13-chacha20-executor.exe'
    $run = Start-Process -FilePath $executorBinary -ArgumentList '--validation-only' -RedirectStandardOutput (Join-Path $executorArtifacts 'load.stdout.log') -RedirectStandardError (Join-Path $executorArtifacts 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
    if ($run.ExitCode -ne 0) { throw "ChaCha20 executor failed with exit code $($run.ExitCode)." }
    foreach ($required in @('validation.json','client-hello-proof.json','protocol-proof.json','tls-negotiation.json','result.json','tls-executor-result.json','tls-load-summary.json','executor-identity.json','load-generator-identity.json','load.stdout.log','load.stderr.log')) {
        if (-not (Test-Path (Join-Path $executorArtifacts $required))) { throw "Required artifact missing: $required" }
    }
    $result = Get-Content (Join-Path $executorArtifacts 'tls-executor-result.json') -Raw | ConvertFrom-Json
    $proof = $result.protocolProof
    $hello = Get-Content (Join-Path $executorArtifacts 'client-hello-proof.json') -Raw | ConvertFrom-Json
    if ($result.executor.id -ne 'go-utls-tls13-chacha20-executor' -or $result.loadGenerator.id -ne 'go-utls-tls13-chacha20-load' -or $result.loadGenerator.engineModule -ne 'github.com/refraction-networking/utls' -or $result.loadGenerator.engineModuleVersion -ne 'v1.8.2') { throw 'Executor or generator identity mismatch.' }
    if (@($hello.cipherSuites).Count -ne 1 -or $hello.cipherSuites[0] -ne 'TLS_CHACHA20_POLY1305_SHA256' -or @($hello.supportedVersions).Count -ne 1 -or $hello.supportedVersions[0] -ne 'TLS1.3' -or @($hello.supportedCurves).Count -ne 1 -or $hello.supportedCurves[0] -ne 'X25519' -or @($hello.keyShareGroups).Count -ne 1 -or $hello.keyShareGroups[0] -ne 'X25519' -or $hello.sessionTicketOffered -or $hello.pskOffered -or $hello.earlyDataOffered) { throw 'Exact uTLS ClientHello gate failed.' }
    if ($proof.tlsVersion -ne 'TLS1.3' -or $proof.cipherSuite -ne 'TLS_CHACHA20_POLY1305_SHA256' -or $proof.keyExchangeGroup -ne 'X25519' -or $proof.alpn -ne 'protocol-lab-tls' -or $proof.didResume -or $proof.sessionStateOffered -or $proof.earlyDataAttempted -or $proof.applicationDataBytesSent -ne 0 -or $proof.applicationDataBytesReceived -ne 0) { throw 'Exact ChaCha20 negotiation gate failed.' }
    if ($proof.certificateDerSha256 -ne 'cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e' -or $proof.certificateSpkiSha256 -ne '407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0' -or $proof.sentCertificateCount -ne 1 -or $proof.trustAnchorSent) { throw 'Certificate gate failed.' }
    if ($result.metrics.completedOperations -ne 1 -or $result.metrics.failedOperations -ne 0 -or $result.metrics.timedOutOperations -ne 0) { throw 'Outcome gate failed.' }

    $unsupportedIDs = @('tls.handshake.full','tls.handshake.resumed','tls.handshake.full.tls12','tls.handshake.mutual-auth','tls.early-data.accepted','tls.early-data.rejected','tls.key-update.diagnostic','tls.record.throughput','tls.record.coverage')
    foreach ($unsupported in $unsupportedIDs) {
        $dir = Join-Path $ArtifactRoot ('unsupported-' + ($unsupported -replace '\.','-'))
        New-Item -ItemType Directory -Force $dir | Out-Null
        $env:PLAB_SCENARIO_ID = $unsupported
        $env:PLAB_ARTIFACT_DIR = $dir
        $unsupportedRun = Start-Process -FilePath $executorBinary -RedirectStandardOutput (Join-Path $dir 'load.stdout.log') -RedirectStandardError (Join-Path $dir 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
        if ($unsupportedRun.ExitCode -ne 3) { throw "$unsupported did not exit unsupported." }
        $unsupportedResult = Get-Content (Join-Path $dir 'unsupported.json') -Raw | ConvertFrom-Json
        if ($unsupportedResult.status -ne 'unsupported' -or $unsupportedResult.scenarioId -ne $unsupported) { throw "$unsupported unsupported evidence mismatch." }
    }
    $unknownRoot = Join-Path $ArtifactRoot unknown
    New-Item -ItemType Directory -Force $unknownRoot | Out-Null
    $env:PLAB_SCENARIO_ID = 'tls.unknown'
    $env:PLAB_ARTIFACT_DIR = $unknownRoot
    $unknownRun = Start-Process -FilePath $executorBinary -RedirectStandardOutput (Join-Path $unknownRoot 'load.stdout.log') -RedirectStandardError (Join-Path $unknownRoot 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
    if ($unknownRun.ExitCode -ne 2) { throw 'Unknown identity did not fail closed.' }

    [pscustomobject]@{
        authorityCommit = $authority.commit
        scenarioPackageSha256 = (Get-FileHash $scenarioArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        executorPackageSha256 = (Get-FileHash $executorArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        targetPackageSha256 = (Get-FileHash $targetArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        loadEngine = 'github.com/refraction-networking/utls@v1.8.2'
        cipherSuite = $proof.cipherSuite
        keyExchangeGroup = $proof.keyExchangeGroup
        completedOperations = [int]$result.metrics.completedOperations
        failedOperations = [int]$result.metrics.failedOperations
        timedOutOperations = [int]$result.metrics.timedOutOperations
        unsupportedScenarioIds = $unsupportedIDs
        evidenceRoot = $ArtifactRoot
    } | ConvertTo-Json -Depth 8
}
finally {
    if ($null -ne $target -and -not $target.HasExited) { Stop-Process -Id $target.Id -Force }
    foreach ($name in $envNames) { [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process') }
}
