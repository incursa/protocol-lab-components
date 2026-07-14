[CmdletBinding()]
param([string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path)

$ErrorActionPreference='Stop'
$Root=[IO.Path]::GetFullPath($Root)
$allRows=@(
    'tls.handshake.full','tls.handshake.resumed','tls.handshake.full.tls12','tls.handshake.full.chacha20',
    'tls.handshake.mutual-auth','tls.early-data.accepted','tls.early-data.rejected','tls.key-update.diagnostic',
    'tls.record.coverage','tls.record.throughput'
)
$expectations=@{
    'openssl-s-server'=@{packageId='org.protocol-lab.components.implementation.openssl-s-server';version='3.3.0';commit='24e7fcf7aff2caadbdee879f615c63981ed132dc';license='Apache-2.0'}
    'gnutls-serv'=@{packageId='org.protocol-lab.components.implementation.gnutls-serv';version='3.8.9';commit='011bda1be01e4a47224adb3cbc32fcb06cba7be1';license='GPL-3.0-or-later'}
}

foreach($componentName in @('openssl-s-server','gnutls-serv')){
    $componentRoot=Join-Path $Root "implementations/$componentName"
    $public=Get-Content (Join-Path $componentRoot 'protocol-lab-package.json') -Raw|ConvertFrom-Json
    $internal=Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw|ConvertFrom-Json
    $toolchain=Get-Content (Join-Path $componentRoot 'toolchain.json') -Raw|ConvertFrom-Json
    $entryPath=Join-Path $componentRoot "implementations/$componentName.yaml"
    $entry=Get-Content $entryPath -Raw
    $expected=$expectations[$componentName]
    if($public.packageId-ne$expected.packageId-or$public.packageVersion-ne'0.1.0'-or$public.kind-ne'implementation'){throw "$componentName public identity mismatch."}
    if(@($public.providedImplementations[0].scenarios).Count-ne 1-or$public.providedImplementations[0].scenarios[0]-ne'tls.handshake.full'){throw "$componentName must declare only tls.handshake.full."}
    if($internal.runtime.requiredVersion-ne$expected.version-or$internal.runtime.upstreamCommit-ne$expected.commit){throw "$componentName internal provenance mismatch."}
    if($toolchain.implementation.upstreamVersion-ne$expected.version-or$toolchain.implementation.upstreamCommit-ne$expected.commit-or$toolchain.implementation.license-ne$expected.license-or$toolchain.implementation.binaryBundled){throw "$componentName toolchain provenance or license mismatch."}
    foreach($row in $allRows){if($entry-notmatch("(?m)^  "+[regex]::Escape($row)+':')){throw "$componentName is missing scenarioCoverage for $row."}}
    if($entry-notmatch'(?m)^  tls\.handshake\.full: \{status: supported,' ){throw "$componentName full-handshake support is missing."}
    foreach($row in $allRows|Where-Object{$_-ne'tls.handshake.full'}){if($entry-notmatch("(?m)^  "+[regex]::Escape($row)+': \{status: unsupported, reason: .+\}$')){throw "$componentName must give an unsupported reason for $row."}}
    foreach($required in @('README.md','THIRD-PARTY-NOTICES.md','certs/leaf.pem','certs/leaf-key.pem','run.sh')){if(-not(Test-Path (Join-Path $componentRoot $required)-PathType Leaf)){throw "$componentName missing $required."}}
    $leafText=((Get-Content (Join-Path $componentRoot 'certs/leaf.pem') -Raw).Trim()-replace"`r`n","`n")
    $canonicalText=((Get-Content (Join-Path $Root 'implementations/go-tls13/certs/leaf.pem') -Raw).Trim()-replace"`r`n","`n")
    if($leafText-ne$canonicalText){throw "$componentName certificate differs from the canonical TLS leaf."}
}

$opensslPlan=& (Join-Path $Root 'implementations/openssl-s-server/run.ps1') -PlanOnly|ConvertFrom-Json
if($opensslPlan.implementationId-ne'openssl-s-server'-or$opensslPlan.upstreamVersion-ne'3.3.0'-or$opensslPlan.scenarioId-ne'tls.handshake.full'){throw 'OpenSSL plan smoke identity mismatch.'}
$opensslArguments=@($opensslPlan.arguments)
foreach($control in @('-tls1_3','TLS_AES_128_GCM_SHA256','X25519','ecdsa_secp256r1_sha256','protocol-lab-tls','-no_ticket')){if($opensslArguments-notcontains$control){throw "OpenSSL plan missing $control."}}

$gnutlsScript=Get-Content (Join-Path $Root 'implementations/gnutls-serv/run.sh') -Raw
foreach($control in @('VERS-TLS1.3','AES-128-GCM','GROUP-X25519','SIGN-ECDSA-SECP256R1-SHA256','--alpn-fatal','--sni-hostname-fatal','--noticket','--plan-only')){if($gnutlsScript-notmatch[regex]::Escape($control)){throw "GnuTLS wrapper missing $control."}}

[pscustomobject]@{status='passed';packages=@('openssl-s-server','gnutls-serv');supportedScenarios=@('tls.handshake.full');coverageRows=$allRows.Count}|ConvertTo-Json -Depth 4
