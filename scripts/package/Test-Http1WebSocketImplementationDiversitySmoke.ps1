[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/http1-websocket-diversity-packages'),
    [string]$SmokeRoot = (Join-Path $Root 'artifacts/http1-websocket-diversity-smoke'),
    [ValidateSet(5)][int]$DurationSeconds = 5,
    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression.FileSystem

$scenarios = @(
    'http1.websocket.rfc6455.cleartext.upgrade',
    'http1.websocket.rfc6455.cleartext.control-frames',
    'http1.websocket.rfc6455.cleartext.text-echo',
    'http1.websocket.rfc6455.cleartext.binary-echo',
    'http1.websocket.rfc6455.cleartext.close'
)
$targets = @(
    [pscustomobject]@{ id='websocat-http1-websocket'; packageId='org.protocol-lab.components.implementation.websocat-http1-websocket'; image='incursa-protocol-lab-websocat-http1-websocket:0.1.0'; dockerfile='docker/Websocat.Dockerfile'; port=18092; diagnosticOnly=$true; scenarios=@('http1.websocket.rfc6455.cleartext.upgrade','http1.websocket.rfc6455.cleartext.control-frames','http1.websocket.rfc6455.cleartext.text-echo','http1.websocket.rfc6455.cleartext.close') },
    [pscustomobject]@{ id='node-ws-websocket'; packageId='org.protocol-lab.components.implementation.node-ws-websocket'; image='incursa-protocol-lab-node-ws-websocket:0.1.0'; dockerfile='docker/NodeWs.Dockerfile'; port=18093; diagnosticOnly=$false; scenarios=$scenarios },
    [pscustomobject]@{ id='jetty-websocket'; packageId='org.protocol-lab.components.implementation.jetty-websocket'; image='incursa-protocol-lab-jetty-websocket:0.1.0'; dockerfile='docker/Jetty.Dockerfile'; port=18094; diagnosticOnly=$false; scenarios=$scenarios },
    [pscustomobject]@{ id='uwebsockets-websocket'; packageId='org.protocol-lab.components.implementation.uwebsockets-websocket'; image='incursa-protocol-lab-uwebsockets-websocket:0.1.0'; dockerfile='docker/UWebSockets.Dockerfile'; port=18095; diagnosticOnly=$false; scenarios=$scenarios }
)

if (-not $SkipBuild) {
    $resolvedRoot = [IO.Path]::GetFullPath($Root)
    $resolvedOutputRoot = [IO.Path]::GetFullPath($OutputRoot)
    if (-not $resolvedOutputRoot.StartsWith($resolvedRoot, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to clean output root outside repository: $resolvedOutputRoot"
    }
    Remove-Item -LiteralPath $resolvedOutputRoot -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null
    & (Join-Path $PSScriptRoot 'Build-Http1WebSocketCleartextScenarioPackage.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-GoHttp1WebSocketExecutorPackage.ps1') win-x64 -Root $Root -OutputRoot $OutputRoot -AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-WebsocatHttp1WebSocketPackage.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-NodeWsWebSocketPackage.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-JettyWebSocketPackage.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource
    & (Join-Path $PSScriptRoot 'Build-UWebSocketsWebSocketPackage.ps1') -Root $Root -OutputRoot $OutputRoot -AllowDirtySource
}

$scenarioPackage = Get-ChildItem $OutputRoot -File -Filter 'org.protocol-lab.components.scenario.http1-websocket-cleartext-performance.0.1.0.plabpkg' | Select-Object -First 1
$executorPackage = Get-ChildItem $OutputRoot -File -Filter 'org.protocol-lab.components.executor.go-http1-websocket-executor.0.1.0.win-x64.plabpkg' | Select-Object -First 1
if ($null -eq $scenarioPackage -or $null -eq $executorPackage) { throw 'Scenario or executor package artifact was not found.' }

Remove-Item -LiteralPath $SmokeRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $SmokeRoot | Out-Null
$scenarioRoot = Join-Path $SmokeRoot 'scenario'
$executorRoot = Join-Path $SmokeRoot 'executor'
[IO.Compression.ZipFile]::ExtractToDirectory($scenarioPackage.FullName, $scenarioRoot)
[IO.Compression.ZipFile]::ExtractToDirectory($executorPackage.FullName, $executorRoot)
& pwsh -NoLogo -NoProfile -File (Join-Path $scenarioRoot 'validate.ps1')
if ($LASTEXITCODE -ne 0) { throw 'Extracted scenario package validation failed.' }
$executor = Join-Path $executorRoot 'bin/win-x64/go-http1-websocket-executor.exe'

$saved = @{}
$variables = @('PLAB_EXECUTOR_ID','PLAB_EXECUTOR_VERSION','PLAB_LOAD_GENERATOR_ID','PLAB_LOAD_GENERATOR_VERSION','PLAB_PROTOCOL','PLAB_PROTOCOL_VARIANT','PLAB_LOAD_PROFILE_ID','PLAB_CONNECTIONS','PLAB_CONCURRENCY','PLAB_DURATION_SECONDS','PLAB_WARMUP_SECONDS','PLAB_REPETITION','PLAB_OPERATION_TIMEOUT_MILLISECONDS','PLAB_SCENARIO_ID')
foreach ($name in $variables) { $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }

$env:PLAB_EXECUTOR_ID='go-http1-websocket-executor'
$env:PLAB_EXECUTOR_VERSION='0.1.0'
$env:PLAB_LOAD_GENERATOR_ID='go-http1-websocket-load'
$env:PLAB_LOAD_GENERATOR_VERSION='0.1.0'
$env:PLAB_PROTOCOL='h1'
$env:PLAB_PROTOCOL_VARIANT='websocket-h1-cleartext-upgrade'
$env:PLAB_LOAD_PROFILE_ID='websocket-smoke'
$env:PLAB_CONNECTIONS='1'
$env:PLAB_CONCURRENCY='1'
$env:PLAB_DURATION_SECONDS=[string]$DurationSeconds
$env:PLAB_WARMUP_SECONDS='1'
$env:PLAB_REPETITION='1'
$env:PLAB_OPERATION_TIMEOUT_MILLISECONDS='5000'

$results = [Collections.Generic.List[object]]::new()
try {
    foreach ($target in $targets) {
        $targetPackage = Get-ChildItem $OutputRoot -File -Filter "$($target.packageId).0.1.0.plabpkg" | Select-Object -First 1
        if ($null -eq $targetPackage) { throw "$($target.id) package artifact was not found." }
        $targetRoot = Join-Path $SmokeRoot "targets/$($target.id)"
        [IO.Compression.ZipFile]::ExtractToDirectory($targetPackage.FullName, $targetRoot)
        & docker build --pull -f (Join-Path $targetRoot $target.dockerfile) -t $target.image (Join-Path $targetRoot 'docker')
        if ($LASTEXITCODE -ne 0) { throw "$($target.id) extracted-package Docker build failed." }

        $containerName = "plab-$($target.id)-smoke"
        & docker rm -f $containerName 2>$null | Out-Null
        $containerId = (& docker run -d --rm --name $containerName -p "$($target.port):18081/tcp" $target.image).Trim()
        if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($containerId)) { throw "$($target.id) container failed to start." }
        try {
            Start-Sleep -Seconds 1
            $running = (& docker inspect --format '{{.State.Running}}' $containerName 2>$null).Trim()
            if ($running -ne 'true') {
                throw "$($target.id) readiness failed: $(& docker logs $containerName 2>&1 | Out-String)"
            }
            foreach ($scenarioId in $target.scenarios) {
                $env:PLAB_SCENARIO_ID = $scenarioId
                $artifactRoot = Join-Path $SmokeRoot "results/$($target.id)/$($scenarioId -replace '[^a-zA-Z0-9.-]','-')"
                New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
                $stdout = Join-Path $artifactRoot 'load.stdout.log'
                $stderr = Join-Path $artifactRoot 'load.stderr.log'
                $run = Start-Process -FilePath $executor -WorkingDirectory $executorRoot -ArgumentList @('--target-url',"http://127.0.0.1:$($target.port)/websocket",'--output-dir',$artifactRoot) -RedirectStandardOutput $stdout -RedirectStandardError $stderr -WindowStyle Hidden -PassThru -Wait
                if ($run.ExitCode -ne 0) { throw "$($target.id) $scenarioId executor exit code $($run.ExitCode): $(Get-Content $stderr -Raw)" }
                $result = Get-Content (Join-Path $artifactRoot 'websocket-executor-result.json') -Raw | ConvertFrom-Json
                if ($result.status -ne 'passed' -or $result.scenarioId -ne $scenarioId -or $result.protocolProof.observedProtocol -ne 'websocket-over-h1-cleartext' -or $result.protocolProof.fallbackDetected -ne $false -or $result.metrics.completedOperations -le 0 -or $result.metrics.failedOperations -ne 0 -or $result.metrics.timedOutOperations -ne 0) {
                    throw "$($target.id) $scenarioId normalized evidence failed validation."
                }
                [void]$results.Add([ordered]@{ implementationId=$target.id; diagnosticOnly=$target.diagnosticOnly; scenarioId=$scenarioId; status='passed'; completedOperations=$result.metrics.completedOperations; fallbackDetected=$result.protocolProof.fallbackDetected; artifactRoot=$artifactRoot })
            }
        }
        finally { & docker rm -f $containerName 2>$null | Out-Null }
    }

    $packageEvidence = @($scenarioPackage, $executorPackage) + @($targets | ForEach-Object { Get-ChildItem $OutputRoot -File -Filter "$($_.packageId).0.1.0.plabpkg" | Select-Object -First 1 })
    $summary = [ordered]@{
        status='passed'
        scenarioCount=$scenarios.Count
        implementationCount=$targets.Count
        cellCount=$results.Count
        packages=@($packageEvidence | ForEach-Object { [ordered]@{ name=$_.Name; sha256=(Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant() } })
        results=$results
    }
    $summary | ConvertTo-Json -Depth 10 | Set-Content (Join-Path $SmokeRoot 'smoke-summary.json') -Encoding utf8NoBOM
    $summary
}
finally {
    foreach ($name in $variables) { [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process') }
}
