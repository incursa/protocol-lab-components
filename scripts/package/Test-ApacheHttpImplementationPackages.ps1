[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/apache-http-packages'),
    [string]$SmokeRoot = (Join-Path $Root 'artifacts/apache-http-package-smoke'),
    [switch]$SkipBuild,
    [switch]$AllowDirtySource
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression.FileSystem
$Root = [System.IO.Path]::GetFullPath($Root)
$OutputRoot = if ([System.IO.Path]::IsPathRooted($OutputRoot)) { [System.IO.Path]::GetFullPath($OutputRoot) } else { [System.IO.Path]::GetFullPath((Join-Path $Root $OutputRoot)) }
$SmokeRoot = if ([System.IO.Path]::IsPathRooted($SmokeRoot)) { [System.IO.Path]::GetFullPath($SmokeRoot) } else { [System.IO.Path]::GetFullPath((Join-Path $Root $SmokeRoot)) }

function Get-OnePackage {
    param([Parameter(Mandatory)][string]$Pattern)
    $matches = @(Get-ChildItem -LiteralPath $OutputRoot -File -Filter $Pattern)
    if ($matches.Count -ne 1) { throw "Expected one package matching '$Pattern', found $($matches.Count)." }
    return $matches[0]
}

function Expand-Package {
    param([Parameter(Mandatory)][System.IO.FileInfo]$Package, [Parameter(Mandatory)][string]$Destination)
    Remove-Item -LiteralPath $Destination -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force -Path $Destination | Out-Null
    [System.IO.Compression.ZipFile]::ExtractToDirectory($Package.FullName, $Destination)
    return (Get-Content -LiteralPath (Join-Path $Destination 'protocol-lab-package.json') -Raw | ConvertFrom-Json)
}

function Wait-TargetPort {
    param([Parameter(Mandatory)][int]$Port, [Parameter(Mandatory)][System.Diagnostics.Process]$Process, [Parameter(Mandatory)][string]$StderrPath)
    $deadline = (Get-Date).AddSeconds(30)
    while ((Get-Date) -lt $deadline) {
        if ($Process.HasExited) {
            $stderr = if (Test-Path -LiteralPath $StderrPath) { Get-Content -LiteralPath $StderrPath -Raw } else { '' }
            throw "Target exited early with $($Process.ExitCode): $stderr"
        }
        $client = [System.Net.Sockets.TcpClient]::new()
        try {
            $task = $client.ConnectAsync('127.0.0.1', $Port)
            if ($task.Wait(250) -and $client.Connected) { return }
        }
        catch { }
        finally { $client.Dispose() }
        Start-Sleep -Milliseconds 150
    }
    throw "Target did not listen on port $Port before the readiness deadline."
}

function Stop-Target {
    param([AllowNull()][System.Diagnostics.Process]$Process, [string]$ContainerName)
    if (-not [string]::IsNullOrWhiteSpace($ContainerName)) { & docker rm --force $ContainerName 2>$null | Out-Null }
    if ($null -ne $Process -and -not $Process.HasExited) {
        if (-not $Process.WaitForExit(5000)) { Stop-Process -Id $Process.Id -Force -ErrorAction SilentlyContinue }
    }
}

function Assert-ExecutorResult {
    param(
        [Parameter(Mandatory)][string]$ArtifactRoot,
        [Parameter(Mandatory)][string]$ExecutorId,
        [Parameter(Mandatory)][string]$RequestedVersion,
        [Parameter(Mandatory)][AllowEmptyString()][string]$ExecutionVariant
    )
    foreach ($name in @('validation.json', 'result.json', 'protocol-proof.json', 'executor-identity.json')) {
        if (-not (Test-Path -LiteralPath (Join-Path $ArtifactRoot $name) -PathType Leaf)) { throw "$ExecutorId did not produce $name." }
    }
    $result = Get-Content -LiteralPath (Join-Path $ArtifactRoot 'result.json') -Raw | ConvertFrom-Json
    $proof = Get-Content -LiteralPath (Join-Path $ArtifactRoot 'protocol-proof.json') -Raw | ConvertFrom-Json
    if ($result.passed -ne $true -or $result.executorId -ne $ExecutorId -or $result.fallbackDetected -ne $false -or $result.unexpectedFailureCount -ne 0 -or $result.timeoutCount -ne 0) {
        throw "$ExecutorId validation result did not pass the zero-failure/no-fallback gate."
    }
    if (@($proof.observedProtocolVersions) -notcontains $RequestedVersion -or $proof.requestedProtocolVersion -ne $RequestedVersion -or $proof.fallbackDetected -ne $false) {
        throw "$ExecutorId protocol proof did not establish exact $RequestedVersion."
    }
    if (-not [string]::IsNullOrWhiteSpace($ExecutionVariant) -and $proof.executionVariant -ne $ExecutionVariant) {
        throw "$ExecutorId execution variant '$($proof.executionVariant)' did not match '$ExecutionVariant'."
    }
    foreach ($check in @($result.checks)) {
        if ($check.passed -ne $true -or $check.observedPayloadSha256 -ne $check.expectedPayloadSha256 -or $check.observedPayloadLength -ne $check.expectedPayloadLength) {
            throw "$ExecutorId scenario '$($check.scenarioId)' failed exact payload validation."
        }
    }
    return $result
}

if (-not $SkipBuild) {
    & (Join-Path $PSScriptRoot 'Build-Http1PerformanceScenarioPackage.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-GoHttp1ExecutorPackage.ps1') win-x64 -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-ApacheHttp1Package.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-Http2PerformanceScenarioPackage.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-GoHttp2ExecutorPackage.ps1') win-x64 -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-ApacheHttp2Package.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource:$AllowDirtySource
}

$packages = [ordered]@{
    http1Scenario = Get-OnePackage 'org.protocol-lab.components.scenario.http1-performance.0.1.0.plabpkg'
    http1Executor = Get-OnePackage 'org.protocol-lab.components.executor.go-http1-executor.0.3.0.win-x64.plabpkg'
    http1Target = Get-OnePackage 'org.protocol-lab.components.implementation.apache-http1.0.1.0.plabpkg'
    http2Scenario = Get-OnePackage 'org.protocol-lab.components.scenario.http2-performance.0.2.0.plabpkg'
    http2Executor = Get-OnePackage 'org.protocol-lab.components.executor.go-http2-executor.0.3.0.win-x64.plabpkg'
    http2Target = Get-OnePackage 'org.protocol-lab.components.implementation.apache-http2.0.1.0.plabpkg'
}

Remove-Item -LiteralPath $SmokeRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $SmokeRoot | Out-Null
$roots = @{}
foreach ($entry in $packages.GetEnumerator()) {
    $destination = Join-Path $SmokeRoot $entry.Key
    $manifest = Expand-Package -Package $entry.Value -Destination $destination
    if ($manifest.schemaVersion -ne 'protocol-lab-package-v2') { throw "$($entry.Key) is not package-v2." }
    $roots[$entry.Key] = $destination
}
& pwsh -NoLogo -NoProfile -File (Join-Path $roots.http1Scenario 'validate.ps1')
if ($LASTEXITCODE -ne 0) { throw 'Extracted HTTP/1 scenario package validation failed.' }
& pwsh -NoLogo -NoProfile -File (Join-Path $roots.http2Scenario 'validate.ps1')
if ($LASTEXITCODE -ne 0) { throw 'Extracted HTTP/2 scenario package validation failed.' }

$savedEnvironment = @{}
$environmentNames = @('PLAB_EXECUTOR_ID', 'PLAB_EXECUTOR_VERSION', 'PLAB_PROTOCOL', 'PLAB_PROTOCOL_VARIANT')
foreach ($name in $environmentNames) { $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
$summaries = @()
try {
    $http1Port = Get-Random -Minimum 22000 -Maximum 28000
    $http1Container = "plab-apache-http1-smoke-$PID"
    $http1Out = Join-Path $SmokeRoot 'apache-http1.stdout.log'
    $http1Err = Join-Path $SmokeRoot 'apache-http1.stderr.log'
    $http1Process = Start-Process -FilePath (Get-Command pwsh).Source -ArgumentList @('-NoLogo', '-NoProfile', '-File', (Join-Path $roots.http1Target 'run.ps1'), '-Port', $http1Port, '-ContainerName', $http1Container) -WorkingDirectory $roots.http1Target -RedirectStandardOutput $http1Out -RedirectStandardError $http1Err -WindowStyle Hidden -PassThru
    try {
        Wait-TargetPort -Port $http1Port -Process $http1Process -StderrPath $http1Err
        $env:PLAB_EXECUTOR_ID = 'go-http1-executor'; $env:PLAB_EXECUTOR_VERSION = '0.3.0'; $env:PLAB_PROTOCOL = 'h1'; Remove-Item Env:PLAB_PROTOCOL_VARIANT -ErrorAction SilentlyContinue
        $artifactRoot = Join-Path $SmokeRoot 'http1-executor-artifacts'; New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
        $run = Start-Process -FilePath (Join-Path $roots.http1Executor 'bin/win-x64/go-http1-executor.exe') -ArgumentList @('--target-url', "http://127.0.0.1:$http1Port", '--output-dir', $artifactRoot, '--validation-only') -WorkingDirectory $roots.http1Executor -RedirectStandardOutput (Join-Path $artifactRoot 'executor.stdout.log') -RedirectStandardError (Join-Path $artifactRoot 'executor.stderr.log') -WindowStyle Hidden -PassThru -Wait
        if ($run.ExitCode -ne 0) { throw "HTTP/1 executor exited $($run.ExitCode): $(Get-Content (Join-Path $artifactRoot 'executor.stderr.log') -Raw)" }
        $result = Assert-ExecutorResult -ArtifactRoot $artifactRoot -ExecutorId 'go-http1-executor' -RequestedVersion 'HTTP/1.1' -ExecutionVariant ''
        $summaries += [ordered]@{ implementationId = 'apache-http1'; variant = 'http1.1'; status = 'passed'; scenarios = @($result.checks.scenarioId); artifactRoot = $artifactRoot }
    }
    finally { Stop-Target -Process $http1Process -ContainerName $http1Container }

    $http2Port = Get-Random -Minimum 28001 -Maximum 34000
    $http2Container = "plab-apache-http2-smoke-$PID"
    $http2Out = Join-Path $SmokeRoot 'apache-http2.stdout.log'
    $http2Err = Join-Path $SmokeRoot 'apache-http2.stderr.log'
    $http2Process = Start-Process -FilePath (Get-Command pwsh).Source -ArgumentList @('-NoLogo', '-NoProfile', '-File', (Join-Path $roots.http2Target 'run.ps1'), '-Variant', 'h2c', '-Port', $http2Port, '-ContainerName', $http2Container) -WorkingDirectory $roots.http2Target -RedirectStandardOutput $http2Out -RedirectStandardError $http2Err -WindowStyle Hidden -PassThru
    try {
        Wait-TargetPort -Port $http2Port -Process $http2Process -StderrPath $http2Err
        $env:PLAB_EXECUTOR_ID = 'go-http2-executor'; $env:PLAB_EXECUTOR_VERSION = '0.3.0'; $env:PLAB_PROTOCOL = 'h2'; $env:PLAB_PROTOCOL_VARIANT = 'http2-h2c-prior-knowledge'
        $artifactRoot = Join-Path $SmokeRoot 'http2-h2c-executor-artifacts'; New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
        $run = Start-Process -FilePath (Join-Path $roots.http2Executor 'bin/win-x64/go-http2-executor.exe') -ArgumentList @('--target-url', "http://127.0.0.1:$http2Port", '--output-dir', $artifactRoot, '--validation-only') -WorkingDirectory $roots.http2Executor -RedirectStandardOutput (Join-Path $artifactRoot 'executor.stdout.log') -RedirectStandardError (Join-Path $artifactRoot 'executor.stderr.log') -WindowStyle Hidden -PassThru -Wait
        if ($run.ExitCode -ne 0) { throw "HTTP/2 executor exited $($run.ExitCode): $(Get-Content (Join-Path $artifactRoot 'executor.stderr.log') -Raw)" }
        $result = Assert-ExecutorResult -ArtifactRoot $artifactRoot -ExecutorId 'go-http2-executor' -RequestedVersion 'HTTP/2.0' -ExecutionVariant 'http2-h2c-prior-knowledge'
        $summaries += [ordered]@{ implementationId = 'apache-http2'; variant = 'http2-h2c-prior-knowledge'; status = 'passed'; scenarios = @($result.checks.scenarioId); artifactRoot = $artifactRoot }
    }
    finally { Stop-Target -Process $http2Process -ContainerName $http2Container }

    $tlsPort = Get-Random -Minimum 34001 -Maximum 40000
    $tlsContainer = "plab-apache-http2-tls-smoke-$PID"
    $tlsOut = Join-Path $SmokeRoot 'apache-http2-tls.stdout.log'
    $tlsErr = Join-Path $SmokeRoot 'apache-http2-tls.stderr.log'
    $tlsProcess = Start-Process -FilePath (Get-Command pwsh).Source -ArgumentList @('-NoLogo', '-NoProfile', '-File', (Join-Path $roots.http2Target 'run.ps1'), '-Variant', 'tls-alpn', '-Port', $tlsPort, '-ContainerName', $tlsContainer) -WorkingDirectory $roots.http2Target -RedirectStandardOutput $tlsOut -RedirectStandardError $tlsErr -WindowStyle Hidden -PassThru
    try {
        Wait-TargetPort -Port $tlsPort -Process $tlsProcess -StderrPath $tlsErr
        $handler = [System.Net.Http.HttpClientHandler]::new()
        $handler.SslProtocols = [System.Security.Authentication.SslProtocols]::Tls13
        $handler.ServerCertificateCustomValidationCallback = [System.Net.Http.HttpClientHandler]::DangerousAcceptAnyServerCertificateValidator
        $client = [System.Net.Http.HttpClient]::new($handler)
        try {
            $request = [System.Net.Http.HttpRequestMessage]::new([System.Net.Http.HttpMethod]::Get, "https://127.0.0.1:$tlsPort/plaintext")
            $request.Version = [Version]'2.0'
            $request.VersionPolicy = [System.Net.Http.HttpVersionPolicy]::RequestVersionExact
            $response = $client.SendAsync($request).GetAwaiter().GetResult()
            $body = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
            if ($response.StatusCode -ne [System.Net.HttpStatusCode]::OK -or $response.Version -ne [Version]'2.0' -or $response.Content.Headers.ContentType.MediaType -ne 'text/plain' -or $body -ne 'Hello, World!') {
                throw 'Apache HTTP/2 TLS/ALPN local protocol smoke did not establish exact HTTP/2 and payload semantics.'
            }
            $summaries += [ordered]@{
                implementationId = 'apache-http2'
                variant = 'http2-tls-alpn'
                status = 'local-protocol-smoke-passed'
                observedProtocolVersion = $response.Version.ToString()
                tlsPolicy = 'tls13-with-package-test-certificate-diagnostic-bypass'
                executorStatus = 'unavailable'
                unavailableReason = 'go-http2-executor@0.3.0 rejects HTTPS targets; no exact general HTTP/2 TLS executor package exists.'
            }
        }
        finally { $client.Dispose(); $handler.Dispose() }
    }
    finally { Stop-Target -Process $tlsProcess -ContainerName $tlsContainer }
}
finally {
    foreach ($name in $environmentNames) { [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process') }
}

$summary = [ordered]@{
    schemaVersion = 'protocol-lab.apache-http-package-smoke.v1'
    status = 'passed'
    packages = @($packages.GetEnumerator() | ForEach-Object { [ordered]@{ role = $_.Key; path = $_.Value.FullName; sha256 = (Get-FileHash -LiteralPath $_.Value.FullName -Algorithm SHA256).Hash.ToLowerInvariant() } })
    results = $summaries
    tlsAlpn = [ordered]@{ implementationStatus = 'supported'; executorStatus = 'unavailable'; reason = 'go-http2-executor@0.3.0 rejects HTTPS targets; no exact general HTTP/2 TLS executor package exists.' }
}
$summary | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath (Join-Path $SmokeRoot 'smoke-summary.json') -Encoding utf8NoBOM
$summary
