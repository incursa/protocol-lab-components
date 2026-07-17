[CmdletBinding()]
param(
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference='Stop'
$componentRoot=Join-Path $Root 'implementations/s2n-tls13'
$public=Get-Content (Join-Path $componentRoot 'protocol-lab-package.json') -Raw|ConvertFrom-Json
$internal=Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw|ConvertFrom-Json
$entry=Get-Content (Join-Path $componentRoot 'implementations/s2n-tls13.yaml') -Raw
$dockerfile=Get-Content (Join-Path $componentRoot 'docker/Dockerfile') -Raw
if($public.packageId-ne'org.protocol-lab.components.implementation.s2n-tls13'-or$public.packageVersion-ne'0.1.0'){throw 's2n-tls13 public identity mismatch.'}
if(-not$internal.dependencies.requiresDocker-or$entry-notmatch'(?m)^targetKind: docker$'){throw 's2n-tls13 must remain a Docker target.'}
foreach($control in @('841f67cb1eb4ce71c829fa12ce940ae33556429c7e137be5c16461999bb7f666','f5f6c6c2ce2370de1aa3ade6899a7321d1127bb8','default_tls13','TLS_AES_128_GCM_SHA256','x25519','protocol-lab-tls','tls.plab.test','s2n_config_set_session_tickets_onoff')){if(($dockerfile+$entry+(Get-Content (Join-Path $componentRoot 'source/main.c') -Raw))-notmatch[regex]::Escape($control)){throw "s2n-tls13 package missing $control."}}
$leaf=((Get-Content (Join-Path $componentRoot 'certs/leaf.pem') -Raw).Trim()-replace"`r`n","`n")
$canonical=((Get-Content (Join-Path $Root 'implementations/go-tls13/certs/leaf.pem') -Raw).Trim()-replace"`r`n","`n")
if($leaf-ne$canonical){throw 's2n-tls13 certificate differs from the canonical TLS leaf.'}
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath 'implementations/s2n-tls13' -RuntimeIdentifier linux-x64 -IncludeReadme -AllowDirtySource:$AllowDirtySource
