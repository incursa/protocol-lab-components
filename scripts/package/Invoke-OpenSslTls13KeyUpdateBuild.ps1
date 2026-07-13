[CmdletBinding()]
param(
    [Parameter(Mandatory)][ValidateSet('implementation','executor')][string]$Kind,
    [Parameter(Mandatory)][ValidateSet('win-x64','linux-x64')][string]$RuntimeIdentifier,
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OpenSslRoot='C:\Strawberry\c',
    [string]$LinuxImage='alpine@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412'
)

$ErrorActionPreference='Stop'
$Root=[IO.Path]::GetFullPath($Root)
$component=if($Kind-eq'implementation'){'implementations/openssl-tls13-key-update'}else{'executors/openssl-tls13-key-update-executor'}
$sourceRelative="$component/source/main.c"
$binaryBase=if($Kind-eq'implementation'){'openssl-tls13-key-update'}else{'openssl-tls13-key-update-executor'}
$buildRoot=Join-Path $Root "$component/source/build/$RuntimeIdentifier"
New-Item -ItemType Directory -Force $buildRoot|Out-Null

if($RuntimeIdentifier-eq'win-x64'){
    $gcc=Join-Path $OpenSslRoot 'bin/gcc.exe'
    if(-not(Test-Path $gcc)){throw "Pinned Windows GCC was not found at $gcc."}
    $opensslVersion=& (Join-Path $OpenSslRoot 'bin/openssl.exe') version
    if($LASTEXITCODE-ne0-or$opensslVersion-notmatch'^OpenSSL 3\.3\.0 '){throw "Expected OpenSSL 3.3.0, observed '$opensslVersion'."}
    foreach($dll in @('libcrypto-3-x64__.dll','libssl-3-x64__.dll')){if(-not(Test-Path (Join-Path $OpenSslRoot "bin/$dll"))){throw "Required OpenSSL runtime $dll is missing."}}
    $output=Join-Path $buildRoot "$binaryBase.exe"
    & $gcc -std=c11 -O2 -Wall -Wextra -Wpedantic -Werror "-I$(Join-Path $OpenSslRoot include)" (Join-Path $Root $sourceRelative) "-L$(Join-Path $OpenSslRoot lib)" -lssl -lcrypto -lws2_32 -o $output
    if($LASTEXITCODE-ne0){throw "$Kind Windows build failed."}
    Copy-Item -LiteralPath (Join-Path $OpenSslRoot 'bin/libcrypto-3-x64__.dll'),(Join-Path $OpenSslRoot 'bin/libssl-3-x64__.dll') -Destination $buildRoot -Force
    & $output --self-test|Out-Host
    if($LASTEXITCODE-ne0){throw "$Kind Windows self-test failed."}
    return [pscustomobject]@{binary=$output;buildRoot=$buildRoot;engineVersion='3.3.0';linkage='dynamic-package-local'}
}

$mountRoot=$Root.Replace('\','/')
$outputRelative="$component/source/build/linux-x64/$binaryBase"
$compile="apk add --no-cache build-base openssl-dev=3.5.7-r0 openssl-libs-static=3.5.7-r0 >/dev/null && gcc -std=c11 -O2 -Wall -Wextra -Wpedantic -Werror -static '$sourceRelative' -lssl -lcrypto -pthread -o '$outputRelative' && '$outputRelative' --self-test"
& docker run --rm -v "${mountRoot}:/src" -w /src $LinuxImage sh -lc $compile|Out-Host
if($LASTEXITCODE-ne0){throw "$Kind Linux build or self-test failed."}
$output=Join-Path $Root $outputRelative
if(-not(Test-Path $output)){throw "$Kind Linux binary was not materialized."}
return [pscustomobject]@{binary=$output;buildRoot=$buildRoot;engineVersion='3.5.7';linkage='static'}
