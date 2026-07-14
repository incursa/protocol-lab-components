[CmdletBinding()]
param([switch]$PlanOnly)

$ErrorActionPreference = 'Stop'
$scenario = if ([string]::IsNullOrWhiteSpace($env:PLAB_SCENARIO_ID)) { 'tls.handshake.full' } else { $env:PLAB_SCENARIO_ID.Trim() }
if ($scenario -ne 'tls.handshake.full') {
    [pscustomobject]@{schemaVersion='protocol-lab.unsupported.v1';status='unsupported';scenarioId=$scenario;implementationId='openssl-s-server';supportedScenarios=@('tls.handshake.full')} | ConvertTo-Json -Compress
    exit 3
}

$listenAddress = if (-not [string]::IsNullOrWhiteSpace($env:PLAB_LISTEN_ADDRESS)) { $env:PLAB_LISTEN_ADDRESS.Trim() } elseif (-not [string]::IsNullOrWhiteSpace($env:PLAB_TARGET_PORT)) { "127.0.0.1:$($env:PLAB_TARGET_PORT.Trim())" } else { '127.0.0.1:18461' }
$certificate = if ([string]::IsNullOrWhiteSpace($env:PLAB_TLS_CERT_FILE)) { Join-Path $PSScriptRoot 'certs/leaf.pem' } else { $env:PLAB_TLS_CERT_FILE }
$privateKey = if ([string]::IsNullOrWhiteSpace($env:PLAB_TLS_KEY_FILE)) { Join-Path $PSScriptRoot 'certs/leaf-key.pem' } else { $env:PLAB_TLS_KEY_FILE }
$tool = if ([string]::IsNullOrWhiteSpace($env:PLAB_OPENSSL_PATH)) { 'openssl' } else { $env:PLAB_OPENSSL_PATH }
$toolArguments = @('s_server','-4','-accept',$listenAddress,'-cert',$certificate,'-key',$privateKey,'-tls1_3','-ciphersuites','TLS_AES_128_GCM_SHA256','-groups','X25519','-sigalgs','ecdsa_secp256r1_sha256','-alpn','protocol-lab-tls','-no_cache','-no_ticket','-num_tickets','0','-quiet','-ign_eof')

if ($PlanOnly) {
    [pscustomobject]@{schemaVersion='protocol-lab.endpoint-plan.v1';implementationId='openssl-s-server';upstreamVersion='3.3.0';scenarioId=$scenario;listenAddress=$listenAddress;executable=$tool;arguments=$toolArguments} | ConvertTo-Json -Depth 5
    return
}

$versionOutput = @(& $tool version 2>&1)
$versionExitCode = $LASTEXITCODE
$versionLine = if ($versionOutput.Count -gt 0) { $versionOutput[0].ToString().Trim() } else { '' }
if ($versionExitCode -ne 0 -or $versionLine -notmatch '^OpenSSL 3\.3\.0(?:\s|$)') { throw "openssl-s-server requires OpenSSL 3.3.0; observed '$versionLine'." }
& $tool @toolArguments
exit $LASTEXITCODE
