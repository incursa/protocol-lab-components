[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$ArtifactRoot = (Join-Path $Root 'artifacts/component-aware-release-tests')
)

$ErrorActionPreference = 'Stop'
$graphPath = Join-Path $Root 'release/component-graph.v1.json'
& (Join-Path $PSScriptRoot 'Test-ProtocolLabComponentReleaseGraph.ps1') -Root $Root -GraphPath $graphPath
& (Join-Path $PSScriptRoot 'Test-ProtocolLabReleaseIntents.ps1') -Root $Root -GraphPath $graphPath

function Get-ClosureDigest([string]$ComponentId) {
    return ((& (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentClosure.ps1') -Root $Root -GraphPath $graphPath -ComponentId $ComponentId | ConvertFrom-Json).componentClosureDigest)
}

$before = Get-ClosureDigest 'http2-performance-scenarios'
$unrelatedPath = Join-Path $Root 'docs/.component-aware-release-stability-test.txt'
Set-Content -LiteralPath $unrelatedPath -Value 'unrelated documentation test input' -Encoding utf8NoBOM
try { $after = Get-ClosureDigest 'http2-performance-scenarios' }
finally { Remove-Item -LiteralPath $unrelatedPath -Force -ErrorAction SilentlyContinue }
if ($before -ne $after) { throw 'An unrelated documentation file changed the modeled component closure digest.' }

$selection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/http2-performance/scenarios/http2/core/plaintext.yaml' | ConvertFrom-Json
if (@($selection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'apache-http2,caddy-http2,go-http2-executor,http2-performance-scenarios,kestrel-http2,nginx-http2') {
    throw 'Declared reverse-dependency selection did not include the complete modeled HTTP/2 cohort.'
}
$http1Selection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/http1-performance/scenarios/http1/core/plaintext.yaml' | ConvertFrom-Json
if (@($http1Selection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'apache-http1,caddy-http1,go-http1-executor,go-nethttp-http1,http1-performance-scenarios,kestrel-http1,nginx-http1,node-http1') {
    throw 'Declared reverse-dependency selection did not include the complete modeled HTTP/1 cohort.'
}
$dnsClassicSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/dns-classic-calibration/scenarios/dns/classic/query-a-udp.yaml' | ConvertFrom-Json
if (@($dnsClassicSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'bind9-classic-authority,dns-classic-calibration,go-dns-classic-authority,go-dns-tcp-executor,go-dns-udp-executor,technitium-classic-authority') {
    throw 'Declared reverse-dependency selection did not include the complete modeled classic DNS cohort.'
}
$dotSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/dns-dot-performance/scenarios/dns/dot/query-a.yaml' | ConvertFrom-Json
if (@($dotSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'bind9-dot,bind9-dot-resolver,dns-dot-performance,go-dns-dot,go-dns-dot-executor,knot-resolver-secure-dns-resolver') {
    throw 'Declared reverse-dependency selection did not include the complete modeled DoT cohort.'
}
$doh2Selection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/dns-doh2-performance/scenarios/dns/doh2/query-a.yaml' | ConvertFrom-Json
if (@($doh2Selection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'bind9-doh2,dns-doh2-performance,go-dns-doh2,go-dns-doh2-executor,knot-resolver-secure-dns-resolver,unbound-doh2-resolver') {
    throw 'Declared reverse-dependency selection did not include the complete modeled DoH2 cohort.'
}
$doh3Selection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/dns-doh3-performance/scenarios/dns/doh3/query-a.yaml' | ConvertFrom-Json
if (@($doh3Selection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'dns-doh3-performance,go-dns-doh3,go-dns-doh3-executor') {
    throw 'Declared reverse-dependency selection did not include the complete DoH3 cohort.'
}
$doqSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/dns-doq-performance/scenarios/dns/doq/query-a.yaml' | ConvertFrom-Json
if (@($doqSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'dns-doq-performance,go-dns-doq,go-dns-doq-executor') {
    throw 'Declared reverse-dependency selection did not include the complete DoQ cohort.'
}
$http3PeerSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/http3-peer-characterization/scenarios/http3/external/peer-characterization.yaml' | ConvertFrom-Json
if (@($http3PeerSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'curl-http3-client,http3-peer-characterization,lsquic-http3,neqo-http3,ngtcp2-http3,quiche-http3,xquic-http3,xquic-http3-client') {
    throw 'Declared reverse-dependency selection did not include the complete HTTP/3 peer-characterization cohort.'
}
$h3specSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/h3spec-http3-qpack/scenarios/http3/protocol/qpack-repeated-headers.yaml' | ConvertFrom-Json
if (@($h3specSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'aioquic-http3,caddy-http3,curl-http3-client,h2o-http3,h3spec-http3-qpack-executor,h3spec-http3-qpack-scenarios,kestrel-http3,nginx-http3,ngtcp2-http3,quic-go-http3,quiche-http3') {
    throw 'Declared reverse-dependency selection did not include the complete h3spec/QPACK cohort.'
}
$grpcSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/grpc-h2-performance/scenarios/grpc/h2/unary-echo.yaml' | ConvertFrom-Json
if (@($grpcSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'go-grpc-h2,go-grpc-h2-executor,grpc-cpp,grpc-dotnet,grpc-h2-performance,grpc-java-netty,grpc-js') {
    throw 'Declared reverse-dependency selection did not include the complete gRPC HTTP/2 cohort.'
}
$webtransportSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/webtransport-performance/scenarios/webtransport/session-bidi-echo.yaml' | ConvertFrom-Json
if (@($webtransportSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'aioquic-webtransport,go-webtransport-executor,webtransport-go,webtransport-performance') {
    throw 'Declared reverse-dependency selection did not include the complete WebTransport cohort.'
}
$masqueSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/masque-connect-udp-performance/scenarios/masque/connect-udp-tunnel.yaml' | ConvertFrom-Json
if (@($masqueSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'go-masque-connect-udp-executor,masque-connect-udp-performance,masque-go-connect-udp') {
    throw 'Declared reverse-dependency selection did not include the complete MASQUE CONNECT-UDP cohort.'
}

$rawQuicSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/raw-quic-transport/scenarios/quic/transport/stream-throughput.yaml' | ConvertFrom-Json
if (@($rawQuicSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'aioquic-raw,picoquic-raw,quic-go-raw,quic-go-raw-load,quiche-raw,quinn-raw,raw-quic-transport,s2n-quic-raw') {
    throw 'Declared reverse-dependency selection did not include the complete raw QUIC cohort.'
}

$rawQuicFixtureSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'implementations/aioquic-http3/certs/leaf.pem' | ConvertFrom-Json
if (@($rawQuicFixtureSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'aioquic-http3,aioquic-raw,quiche-raw') {
    throw 'Raw QUIC certificate fixture changes did not select the source package and every consuming package.'
}

$rfc9220Selection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/aioquic-rfc9220-websocket/scenarios/http3/websocket/rfc9220-extended-connect.yaml' | ConvertFrom-Json
if (@($rfc9220Selection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'aioquic-http3,aioquic-rfc9220-websocket-executor,aioquic-rfc9220-websocket-scenarios,nghttpx-rfc9220-gateway') {
    throw 'Declared reverse-dependency selection did not include every RFC9220 scenario consumer.'
}

$cleartextWebSocketSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/http1-websocket-cleartext-performance/scenarios/http1/websocket/rfc6455-cleartext-upgrade.yaml' | ConvertFrom-Json
if (@($cleartextWebSocketSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'go-http1-websocket,go-http1-websocket-executor,http1-websocket-cleartext-scenarios,jetty-websocket,node-ws-websocket,uwebsockets-websocket,websocat-http1-websocket') {
    throw 'Declared reverse-dependency selection did not include every cleartext HTTP/1 WebSocket consumer.'
}

$http2WebSocketSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/http2-websocket-performance/scenarios/http2/websocket/rfc8441-binary-echo.yaml' | ConvertFrom-Json
if (@($http2WebSocketSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'go-http2-websocket-executor,http2-websocket-scenarios,jetty-http2-websocket,kestrel-http2-websocket') {
    throw 'Declared reverse-dependency selection did not include every HTTP/2 WebSocket consumer.'
}

$http1OriginSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/http1-performance/scenarios/http1/core/plaintext.yaml' | ConvertFrom-Json
if (@($http1OriginSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'apache-http1,caddy-http1,go-http1-executor,go-nethttp-http1,http1-performance-scenarios,kestrel-http1,nginx-http1,node-http1') {
    throw 'Declared reverse-dependency selection did not include every HTTP/1 core consumer.'
}

$tlsWebSocketSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/http1-websocket-tls-performance/scenarios/http1/websocket/rfc6455-tls-binary-echo.yaml' | ConvertFrom-Json
if (@($tlsWebSocketSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'go-http1-websocket-tls,go-http1-websocket-tls-executor,http1-websocket-tls-scenarios') {
    throw 'Declared reverse-dependency selection did not include the complete TLS WebSocket cohort.'
}

$tls12Selection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'scenarios/tls13-handshake-performance/scenarios/tls/handshake/full-tls12.yaml' | ConvertFrom-Json
if (@($tls12Selection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'dotnet-sslstream-tls13,gnutls-serv,go-tls12,go-tls12-executor,go-tls13,go-tls13-chacha20,go-tls13-executor,go-tls13-mtls,go-tls13-mtls-executor,go-utls-tls13-chacha20-executor,openssl-s-server,rustls-tls13,s2n-tls13,tls-handshake-scenarios,wolfssl-tls13') {
    throw 'Declared reverse-dependency selection did not include the complete modeled TLS cohort.'
}
$certificateSelection = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'implementations/go-dns-dot/certs/root.pem' | ConvertFrom-Json
if (@($certificateSelection.selectedComponents.componentId | Sort-Object) -join ',' -ne 'go-dns-doh2,go-dns-doh2-executor,go-dns-doh3,go-dns-doh3-executor,go-dns-doq,go-dns-doq-executor,go-dns-dot') {
    throw 'Shared secure-DNS certificate changes did not select every consuming package.'
}
$unknown = & (Join-Path $PSScriptRoot 'Get-ProtocolLabComponentReleaseSelection.ps1') -Root $Root -GraphPath $graphPath -ChangedPath 'unmodeled-release-input.txt' | ConvertFrom-Json
if (-not $unknown.fullBuildDryRunRequired) { throw 'Unknown changes must require conservative full-build dry-run.' }

Remove-Item -LiteralPath $ArtifactRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $ArtifactRoot | Out-Null
$packageOutput = Join-Path $ArtifactRoot 'packages'
& (Join-Path $PSScriptRoot 'Build-Http2PerformanceScenarioPackage.ps1') -Root $Root -OutputRoot $packageOutput -AllowDirtySource
$packages = @(Get-ChildItem -LiteralPath $packageOutput -File -Filter '*.plabpkg')
if ($packages.Count -ne 1) { throw "Expected exactly one scenario package, found $($packages.Count)." }
$package = $packages[0]
$attestation = Get-Item -LiteralPath "$($package.FullName).build-attestation.json"
& (Join-Path $PSScriptRoot 'Test-ProtocolLabPackageBuildAttestation.ps1') -PackagePath $package.FullName -AttestationPath $attestation.FullName

$temporaryPayload = Join-Path $Root 'scenarios/http2-performance/.component-aware-release-immutable-test.txt'
Set-Content -LiteralPath $temporaryPayload -Value 'must change package closure' -Encoding utf8NoBOM
try {
    $collision = $null
    try { & (Join-Path $PSScriptRoot 'Build-Http2PerformanceScenarioPackage.ps1') -Root $Root -OutputRoot $packageOutput -AllowDirtySource }
    catch { $collision = $_ }
    if ($null -eq $collision -or $collision.Exception.Message -notmatch 'Immutable package collision') { throw 'Changed package bytes did not require a version advance.' }
}
finally { Remove-Item -LiteralPath $temporaryPayload -Force -ErrorAction SilentlyContinue }

$snapshot = Join-Path $ArtifactRoot 'catalog-2026.07.17-test.json'
& (Join-Path $PSScriptRoot 'New-ProtocolLabCatalogSnapshot.ps1') -CatalogVersion '2026.07.17-test' -PackagePath $package.FullName -AttestationPath $attestation.FullName -ReleaseTag 'packages/http2-performance-scenarios/v0.2.2' -OutputPath $snapshot
& (Join-Path $PSScriptRoot 'Test-ProtocolLabCatalogSnapshot.ps1') -SnapshotPath $snapshot

$invalidIntentRoot = Join-Path $ArtifactRoot 'invalid-intents'
New-Item -ItemType Directory -Force -Path $invalidIntentRoot | Out-Null
'{ "schemaVersion": "protocol-lab.release-intent.v1", "id": "invalid", "classification": "no-release", "status": "approved", "components": ["forbidden"], "reason": "test" }' | Set-Content -LiteralPath (Join-Path $invalidIntentRoot 'invalid.json') -Encoding utf8NoBOM
$intentError = $null
try { & (Join-Path $PSScriptRoot 'Test-ProtocolLabReleaseIntents.ps1') -Root $Root -GraphPath $graphPath -IntentRoot $invalidIntentRoot }
catch { $intentError = $_ }
if ($null -eq $intentError -or $intentError.Exception.Message -notmatch 'no-release intents') { throw 'Invalid no-release intent was accepted.' }

[ordered]@{
    schemaVersion = 'protocol-lab.component-aware-release-tests.v1'
    status = 'passed'
    unaffectedPackageClosureStable = $true
    reverseDependencySelection = $true
    immutableVersionEnforcement = $true
    releaseIntentEnforcement = $true
    catalogSnapshotReproducibility = $true
} | ConvertTo-Json
