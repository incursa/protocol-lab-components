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
    'openssl-s-server'=@{packageId='org.protocol-lab.components.implementation.openssl-s-server';version='3.3.0';commit='4cb31128b5790819dfeea2739fbde265f71a10a2';tagObject='24e7fcf7aff2caadbdee879f615c63981ed132dc';archiveSha='53e66b043322a606abf0087e7699a0e033a37fa13feb9742df35c3a33b18fb02';license='Apache-2.0';image='incursa-protocol-lab-openssl-s-server:0.1.1'}
    'gnutls-serv'=@{packageId='org.protocol-lab.components.implementation.gnutls-serv';version='3.8.9';commit='477a733247460b94cd2b37a10579c27ca6fc196f';tagObject='011bda1be01e4a47224adb3cbc32fcb06cba7be1';archiveSha='69e113d802d1670c4d5ac1b99040b1f2d5c7c05daec5003813c049b5184820ed';license='GPL-3.0-or-later';image='incursa-protocol-lab-gnutls-serv:0.1.1'}
}

foreach($componentName in @('openssl-s-server','gnutls-serv')){
    $componentRoot=Join-Path $Root "implementations/$componentName"
    $public=Get-Content (Join-Path $componentRoot 'protocol-lab-package.json') -Raw|ConvertFrom-Json
    $internal=Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw|ConvertFrom-Json
    $toolchain=Get-Content (Join-Path $componentRoot 'toolchain.json') -Raw|ConvertFrom-Json
    $entryPath=Join-Path $componentRoot "implementations/$componentName.yaml"
    $entry=Get-Content $entryPath -Raw
    $expected=$expectations[$componentName]
    if($public.packageId-ne$expected.packageId-or$public.packageVersion-ne'0.1.1'-or$public.kind-ne'implementation'){throw "$componentName public identity mismatch."}
    if(@($public.providedImplementations[0].scenarios).Count-ne 1-or$public.providedImplementations[0].scenarios[0]-ne'tls.handshake.full'){throw "$componentName must declare only tls.handshake.full."}
    if($internal.runtime.requiredVersion-ne$expected.version-or$internal.runtime.upstreamCommit-ne$expected.commit-or$internal.runtime.acquisitionMode-ne'docker-source-build'){throw "$componentName internal provenance mismatch."}
    if(-not$internal.dependencies.requiresDocker-or@($internal.dependencies.requiredCapabilities|Where-Object{$_.name-eq'docker'-and$_.value-eq'true'}).Count-ne 1){throw "$componentName must require Docker and no host executable capability."}
    if(@($internal.dependencies.requiredCapabilities|Where-Object{$_.name-in@('openssl','gnutls-serv')}).Count-ne 0){throw "$componentName retained a host executable capability gate."}
    $image=@($internal.dependencies.images)[0]
    if($image.image-ne$expected.image-or$image.sourceCommit-ne$expected.commit-or$image.sourceTagObject-ne$expected.tagObject-or$image.sourceArchiveSha256-ne$expected.archiveSha-or$image.baseImageIndexDigest-ne'debian@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818'){throw "$componentName image provenance mismatch."}
    if($toolchain.implementation.version-ne'0.1.1'-or$toolchain.implementation.upstreamVersion-ne$expected.version-or$toolchain.implementation.sourceArchiveSha256-ne$expected.archiveSha-or$toolchain.implementation.license-ne$expected.license-or$toolchain.implementation.binaryBundled-or$toolchain.implementation.acquisitionMode-ne'docker-source-build-exact-version'){throw "$componentName toolchain provenance or license mismatch."}
    if($entry-notmatch'(?m)^targetKind: docker$'-or$entry-notmatch("(?m)^image: "+[regex]::Escape($expected.image)+'$')-or$entry-notmatch'(?m)^dockerfile: docker/Dockerfile$'-or$entry-notmatch'(?m)^buildContext: \.$'){throw "$componentName entry manifest is not a package-local Docker target."}
    foreach($row in $allRows){if($entry-notmatch("(?m)^  "+[regex]::Escape($row)+':')){throw "$componentName is missing scenarioCoverage for $row."}}
    if($entry-notmatch'(?m)^  tls\.handshake\.full: \{status: supported,' ){throw "$componentName full-handshake support is missing."}
    foreach($row in $allRows|Where-Object{$_-ne'tls.handshake.full'}){if($entry-notmatch("(?m)^  "+[regex]::Escape($row)+': \{status: unsupported, reason: .+\}$')){throw "$componentName must give an unsupported reason for $row."}}
    foreach($required in @('README.md','THIRD-PARTY-NOTICES.md','certs/leaf.pem','certs/leaf-key.pem','run.sh','docker/Dockerfile')){if(-not(Test-Path (Join-Path $componentRoot $required)-PathType Leaf)){throw "$componentName missing $required."}}
    $leafText=((Get-Content (Join-Path $componentRoot 'certs/leaf.pem') -Raw).Trim()-replace"`r`n","`n")
    $canonicalText=((Get-Content (Join-Path $Root 'implementations/go-tls13/certs/leaf.pem') -Raw).Trim()-replace"`r`n","`n")
    if($leafText-ne$canonicalText){throw "$componentName certificate differs from the canonical TLS leaf."}
}

$opensslPlan=& (Join-Path $Root 'implementations/openssl-s-server/run.ps1') -PlanOnly|ConvertFrom-Json
if($opensslPlan.implementationId-ne'openssl-s-server'-or$opensslPlan.packageVersion-ne'0.1.1'-or$opensslPlan.upstreamVersion-ne'3.3.0'-or$opensslPlan.scenarioId-ne'tls.handshake.full'-or$opensslPlan.image-ne'incursa-protocol-lab-openssl-s-server:0.1.1'-or$opensslPlan.containerPort-ne 8443){throw 'OpenSSL plan smoke identity mismatch.'}
foreach($control in @('tls1.3','TLS_AES_128_GCM_SHA256','X25519','ecdsa_secp256r1_sha256','protocol-lab-tls','tickets-disabled')){if(@($opensslPlan.controls)-notcontains$control){throw "OpenSSL plan missing $control."}}

$opensslDockerfile=Get-Content (Join-Path $Root 'implementations/openssl-s-server/docker/Dockerfile') -Raw
foreach($control in @('53e66b043322a606abf0087e7699a0e033a37fa13feb9742df35c3a33b18fb02','4cb31128b5790819dfeea2739fbde265f71a10a2','/usr/share/licenses/openssl/LICENSE.txt','-tls1_3','TLS_AES_128_GCM_SHA256','X25519','ecdsa_secp256r1_sha256','protocol-lab-tls','-no_ticket')){if($opensslDockerfile-notmatch[regex]::Escape($control)){throw "OpenSSL Docker target missing $control."}}

$gnutlsScript=Get-Content (Join-Path $Root 'implementations/gnutls-serv/run.sh') -Raw
foreach($control in @('TLS1.3','TLS_AES_128_GCM_SHA256','X25519','SIGN-ECDSA-SECP256R1-SHA256','fatal-sni','fatal-alpn','tickets-disabled','--plan-only')){if($gnutlsScript-notmatch[regex]::Escape($control)){throw "GnuTLS wrapper missing $control."}}
$gnutlsDockerfile=Get-Content (Join-Path $Root 'implementations/gnutls-serv/docker/Dockerfile') -Raw
foreach($control in @('69e113d802d1670c4d5ac1b99040b1f2d5c7c05daec5003813c049b5184820ed','477a733247460b94cd2b37a10579c27ca6fc196f','/usr/share/licenses/gnutls/COPYING','VERS-TLS1.3','AES-128-GCM','GROUP-X25519','SIGN-ECDSA-SECP256R1-SHA256','--alpn-fatal','--sni-hostname-fatal','--noticket')){if($gnutlsDockerfile-notmatch[regex]::Escape($control)){throw "GnuTLS Docker target missing $control."}}

[pscustomobject]@{status='passed';packages=@('openssl-s-server','gnutls-serv');packageVersion='0.1.1';targetKind='docker';supportedScenarios=@('tls.handshake.full');coverageRows=$allRows.Count}|ConvertTo-Json -Depth 4
