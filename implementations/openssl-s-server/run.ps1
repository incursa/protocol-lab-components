[CmdletBinding()]
param(
    [string]$Image = 'incursa-protocol-lab-openssl-s-server:0.1.1',
    [int]$Port = 18461,
    [switch]$SkipBuild,
    [switch]$PlanOnly,
    [switch]$ProofOnly
)

$ErrorActionPreference = 'Stop'
$scenario = if ([string]::IsNullOrWhiteSpace($env:PLAB_SCENARIO_ID)) { 'tls.handshake.full' } else { $env:PLAB_SCENARIO_ID.Trim() }
if ($scenario -ne 'tls.handshake.full') {
    [pscustomobject]@{schemaVersion='protocol-lab.unsupported.v1';status='unsupported';scenarioId=$scenario;implementationId='openssl-s-server';supportedScenarios=@('tls.handshake.full')} | ConvertTo-Json -Compress
    exit 3
}
if ($PlanOnly) {
    [pscustomobject]@{schemaVersion='protocol-lab.endpoint-plan.v1';implementationId='openssl-s-server';packageVersion='0.1.1';upstreamVersion='3.3.0';scenarioId=$scenario;image=$Image;hostPort=$Port;containerPort=8443;controls=@('tls1.3','TLS_AES_128_GCM_SHA256','X25519','ecdsa_secp256r1_sha256','protocol-lab-tls','tickets-disabled')} | ConvertTo-Json -Depth 5
    return
}
Push-Location $PSScriptRoot
try {
    if (-not $SkipBuild) {
        & docker build --pull -f docker/Dockerfile -t $Image .
        if ($LASTEXITCODE -ne 0) { throw "OpenSSL s_server image build failed with exit code $LASTEXITCODE." }
    }
    $versionLine = (& docker run --rm --entrypoint openssl $Image version | Select-Object -First 1).Trim()
    if ($LASTEXITCODE -ne 0 -or $versionLine -notmatch '^OpenSSL 3\.3\.0(?:\s|$)') { throw "Expected OpenSSL 3.3.0, observed '$versionLine'." }
    if ($ProofOnly) {
        [pscustomobject]@{status='proved';image=$Image;upstreamVersion='3.3.0'} | ConvertTo-Json -Compress
        return
    }
    & docker run --rm -p "${Port}:8443/tcp" $Image
    exit $LASTEXITCODE
}
finally { Pop-Location }
