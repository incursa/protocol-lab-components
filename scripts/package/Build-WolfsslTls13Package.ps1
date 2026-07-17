[CmdletBinding()]
param(
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference='Stop'
$componentRoot=Join-Path $Root 'implementations/wolfssl-tls13'
$public=Get-Content (Join-Path $componentRoot 'protocol-lab-package.json') -Raw|ConvertFrom-Json
$internal=Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw|ConvertFrom-Json
$entry=Get-Content (Join-Path $componentRoot 'implementations/wolfssl-tls13.yaml') -Raw
$dockerfile=Get-Content (Join-Path $componentRoot 'docker/Dockerfile') -Raw
$entrypoint=Get-Content (Join-Path $componentRoot 'docker/entrypoint.sh') -Raw
if($public.packageId-ne'org.protocol-lab.components.implementation.wolfssl-tls13'-or$public.packageVersion-ne'0.1.0'){throw 'wolfssl-tls13 public identity mismatch.'}
if(-not$internal.dependencies.requiresDocker-or$entry-notmatch'(?m)^targetKind: docker$'){throw 'wolfssl-tls13 must remain a Docker target.'}
foreach($control in @('2f4ef3d4fd387a9b3191d36a6316d69116c46ff69bb9583b6c82b36d7b8ca114','ac01707f552c611fbd135cc723b2682b3e7f80f2','GPL-3.0-or-later','TLS13-AES128-GCM-SHA256','-t','protocol-lab-tls','tls.plab.test','-T')){if(($dockerfile+$entry+$entrypoint)-notmatch[regex]::Escape($control)){throw "wolfssl-tls13 package missing $control."}}
$leaf=((Get-Content (Join-Path $componentRoot 'certs/leaf.pem') -Raw).Trim()-replace"`r`n","`n")
$canonical=((Get-Content (Join-Path $Root 'implementations/go-tls13/certs/leaf.pem') -Raw).Trim()-replace"`r`n","`n")
if($leaf-ne$canonical){throw 'wolfssl-tls13 certificate differs from the canonical TLS leaf.'}
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath 'implementations/wolfssl-tls13' -RuntimeIdentifier linux-x64 -IncludeReadme -AllowDirtySource:$AllowDirtySource
