[CmdletBinding()]
param(
    [string]$PackageRoot = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-endpoint-docker-packages'),
    [string]$ArtifactRoot = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path 'artifacts/tls-endpoint-docker-extracted-smoke')
)

$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$PackageRoot = [IO.Path]::GetFullPath($PackageRoot)
$ArtifactRoot = [IO.Path]::GetFullPath($ArtifactRoot)
if (-not $ArtifactRoot.StartsWith([IO.Path]::GetFullPath((Join-Path $repoRoot 'artifacts')), [StringComparison]::OrdinalIgnoreCase)) {
    throw 'TLS endpoint Docker smoke artifacts must remain under this worktree artifacts directory.'
}

function Resolve-OnePackage([string]$Pattern) {
    $matches = @(Get-ChildItem -LiteralPath $PackageRoot -Filter $Pattern -File)
    if ($matches.Count -ne 1) { throw "Expected one package matching $Pattern, observed $($matches.Count)." }
    return $matches[0].FullName
}

function Expand-Package([string]$Archive, [string]$Destination) {
    New-Item -ItemType Directory -Force -Path $Destination | Out-Null
    [IO.Compression.ZipFile]::ExtractToDirectory($Archive, $Destination)
    $manifest = Get-Content (Join-Path $Destination 'protocol-lab-package.json') -Raw | ConvertFrom-Json
    if ($manifest.schemaVersion -ne 'protocol-lab-package-v2') { throw "$Archive is not package-v2." }
    return $manifest
}

function Wait-TcpPort([int]$Port) {
    for ($attempt = 0; $attempt -lt 100; $attempt++) {
        $client = [Net.Sockets.TcpClient]::new()
        try {
            $pending = $client.ConnectAsync('127.0.0.1', $Port)
            if ($pending.Wait(100) -and $client.Connected) { return }
        }
        catch { }
        finally { $client.Dispose() }
        Start-Sleep -Milliseconds 100
    }
    throw "TLS Docker target did not listen on 127.0.0.1:$Port."
}

if (Test-Path $ArtifactRoot) { Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force }
New-Item -ItemType Directory -Force -Path $ArtifactRoot | Out-Null

$scenarioArchive = Resolve-OnePackage 'org.protocol-lab.components.scenario.tls13-handshake-performance.0.2.1.plabpkg'
$executorArchive = Resolve-OnePackage 'org.protocol-lab.components.executor.go-tls13-executor.0.3.1.win-x64.plabpkg'
$targetArchives = [ordered]@{
    'openssl-s-server' = Resolve-OnePackage 'org.protocol-lab.components.implementation.openssl-s-server.0.1.1.plabpkg'
    'gnutls-serv' = Resolve-OnePackage 'org.protocol-lab.components.implementation.gnutls-serv.0.1.1.plabpkg'
}

$scenarioRoot = Join-Path $ArtifactRoot 'scenario'
$executorRoot = Join-Path $ArtifactRoot 'executor'
$scenarioManifest = Expand-Package $scenarioArchive $scenarioRoot
$executorManifest = Expand-Package $executorArchive $executorRoot
$authority = Get-Content (Join-Path $scenarioRoot 'authority-lock.json') -Raw | ConvertFrom-Json
if ($authority.commit -ne '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574') { throw 'TLS scenario authority commit mismatch.' }
if ($scenarioManifest.providedScenarios.scenarioId -notcontains 'tls.handshake.full') { throw 'Scenario package is missing tls.handshake.full.' }
if ($executorManifest.providedTestExecutors[0].scenarios -notcontains 'tls.handshake.full') { throw 'Executor package is missing tls.handshake.full.' }

$envNames = @('PLAB_SCENARIO_ID','PLAB_TARGET_BASE_URL','PLAB_ARTIFACT_DIR','PLAB_TLS_ROOT_CERTIFICATE_PATH','PLAB_EXECUTOR_ID','PLAB_EXECUTOR_VERSION','PLAB_LOAD_GENERATOR_ID','PLAB_LOAD_GENERATOR_VERSION','PLAB_PROTOCOL','PLAB_PROTOCOL_VARIANT')
$savedEnvironment = @{}
foreach ($name in $envNames) { $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
$results = [System.Collections.Generic.List[object]]::new()
$containers = [System.Collections.Generic.List[string]]::new()

try {
    $ordinal = 0
    foreach ($implementationId in $targetArchives.Keys) {
        $ordinal++
        $targetRoot = Join-Path $ArtifactRoot "target-$implementationId"
        $targetManifest = Expand-Package $targetArchives[$implementationId] $targetRoot
        if ($targetManifest.packageVersion -ne '0.1.1' -or $targetManifest.providedImplementations[0].implementationId -ne $implementationId -or $targetManifest.providedImplementations[0].scenarios -notcontains 'tls.handshake.full') {
            throw "$implementationId extracted package identity mismatch."
        }
        $entry = Get-Content (Join-Path $targetRoot "implementations/$implementationId.yaml") -Raw
        if ($entry -notmatch '(?m)^targetKind: docker$' -or $entry -notmatch '(?m)^dockerfile: docker/Dockerfile$') { throw "$implementationId is not an extracted Docker target." }

        $image = "protocol-lab-extracted-$implementationId`:0.1.1"
        & docker build --pull -f (Join-Path $targetRoot 'docker/Dockerfile') -t $image $targetRoot
        if ($LASTEXITCODE -ne 0) { throw "$implementationId extracted Docker build failed with exit code $LASTEXITCODE." }
        $licensePath = if ($implementationId -eq 'openssl-s-server') { '/usr/share/licenses/openssl/LICENSE.txt' } else { '/usr/share/licenses/gnutls/COPYING' }
        & docker run --rm --entrypoint test $image -s $licensePath
        if ($LASTEXITCODE -ne 0) { throw "$implementationId built image is missing its upstream license text at $licensePath." }

        $port = 19460 + $ordinal
        $containerName = "plab-tls-endpoint-smoke-$implementationId-$PID"
        $containerId = (& docker run --detach --rm --name $containerName -p "${port}:8443/tcp" $image).Trim()
        if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($containerId)) { throw "$implementationId container did not start." }
        $containers.Add($containerName)
        Wait-TcpPort -Port $port

        $executorArtifacts = Join-Path $ArtifactRoot "executor-$implementationId"
        New-Item -ItemType Directory -Force -Path $executorArtifacts | Out-Null
        $env:PLAB_SCENARIO_ID = 'tls.handshake.full'
        $env:PLAB_TARGET_BASE_URL = "tls://127.0.0.1:$port"
        $env:PLAB_ARTIFACT_DIR = $executorArtifacts
        $env:PLAB_TLS_ROOT_CERTIFICATE_PATH = Join-Path $executorRoot 'certs/root.pem'
        $env:PLAB_EXECUTOR_ID = 'go-tls13-executor'
        $env:PLAB_EXECUTOR_VERSION = '0.3.1'
        $env:PLAB_LOAD_GENERATOR_ID = 'go-crypto-tls13-load'
        $env:PLAB_LOAD_GENERATOR_VERSION = '0.3.1'
        $env:PLAB_PROTOCOL = 'tls'
        $env:PLAB_PROTOCOL_VARIANT = 'tls1.3-full'
        $executorBinary = Join-Path $executorRoot 'bin/win-x64/go-tls13-executor.exe'
        $run = Start-Process -FilePath $executorBinary -ArgumentList '--validation-only' -RedirectStandardOutput (Join-Path $executorArtifacts 'load.stdout.log') -RedirectStandardError (Join-Path $executorArtifacts 'load.stderr.log') -WindowStyle Hidden -Wait -PassThru
        if ($run.ExitCode -ne 0) {
            & docker logs $containerName | Set-Content (Join-Path $executorArtifacts 'target.log')
            throw "$implementationId exact executor validation failed with exit code $($run.ExitCode)."
        }
        foreach ($required in @('validation.json','protocol-proof.json','tls-negotiation.json','result.json','executor-identity.json','load.stdout.log','load.stderr.log')) {
            if (-not (Test-Path (Join-Path $executorArtifacts $required) -PathType Leaf)) { throw "$implementationId required artifact missing: $required" }
        }
        $validation = Get-Content (Join-Path $executorArtifacts 'validation.json') -Raw | ConvertFrom-Json
        $proof = Get-Content (Join-Path $executorArtifacts 'protocol-proof.json') -Raw | ConvertFrom-Json
        if (-not $validation.passed -or $validation.fallbackDetected -or $validation.didResume -or $validation.unexpectedFailureCount -ne 0 -or $validation.timeoutCount -ne 0) { throw "$implementationId validation gate failed." }
        if ($proof.tlsVersion -ne 'TLS1.3' -or $proof.cipherSuite -ne 'TLS_AES_128_GCM_SHA256' -or $proof.keyExchangeGroup -ne 'X25519' -or $proof.alpn -ne 'protocol-lab-tls' -or $proof.didResume -or $proof.earlyDataAttempted -or $proof.applicationDataBytesSent -ne 0 -or $proof.applicationDataBytesReceived -ne 0) { throw "$implementationId exact TLS proof gate failed." }
        if ($proof.certificateDerSha256 -ne 'cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e' -or $proof.certificateSpkiSha256 -ne '407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0' -or $proof.certificateSignatureAlgorithm -ne 'ECDSA-SHA256' -or $proof.certificateNamedCurve -ne 'P-256' -or $proof.verifiedChainCount -ne 1) { throw "$implementationId canonical certificate proof gate failed." }

        $results.Add([ordered]@{
            implementationId = $implementationId
            packageSha256 = (Get-FileHash $targetArchives[$implementationId] -Algorithm SHA256).Hash.ToLowerInvariant()
            imageId = (& docker image inspect $image --format '{{.Id}}').Trim()
            tlsVersion = $proof.tlsVersion
            cipherSuite = $proof.cipherSuite
            keyExchangeGroup = $proof.keyExchangeGroup
            certificateSignatureAlgorithm = $proof.certificateSignatureAlgorithm
            alpn = $proof.alpn
            didResume = [bool]$proof.didResume
            applicationDataBytesSent = [int64]$proof.applicationDataBytesSent
            applicationDataBytesReceived = [int64]$proof.applicationDataBytesReceived
        })
        & docker rm --force $containerName | Out-Null
        [void]$containers.Remove($containerName)
    }

    $summary = [ordered]@{
        status = 'passed'
        authorityCommit = $authority.commit
        scenarioPackageSha256 = (Get-FileHash $scenarioArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        executorPackageSha256 = (Get-FileHash $executorArchive -Algorithm SHA256).Hash.ToLowerInvariant()
        results = $results.ToArray()
        evidenceRoot = $ArtifactRoot
    }
    $summary | ConvertTo-Json -Depth 8 | Set-Content (Join-Path $ArtifactRoot 'smoke-summary.json') -Encoding utf8NoBOM
    $summary | ConvertTo-Json -Depth 8
}
finally {
    foreach ($containerName in @($containers)) { & docker rm --force $containerName 2>$null | Out-Null }
    foreach ($name in $envNames) { [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process') }
}
