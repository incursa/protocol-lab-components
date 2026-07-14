[CmdletBinding()]
param(
    [Parameter(Mandatory)][ValidateSet('implementation','executor')][string]$Kind,
    [Parameter(Mandatory)][ValidateSet('win-x64','linux-x64')][string]$RuntimeIdentifier,
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OpenSslRoot='C:\Strawberry\c',
    [string]$OpenSslSourceUrl='https://github.com/openssl/openssl/releases/download/openssl-3.3.0/openssl-3.3.0.tar.gz',
    [string]$OpenSslSourceSha256='53e66b043322a606abf0087e7699a0e033a37fa13feb9742df35c3a33b18fb02',
    [string]$LinuxImage='alpine@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412'
)

$ErrorActionPreference='Stop'
$Root=[IO.Path]::GetFullPath($Root)
$component=if($Kind-eq'implementation'){'implementations/openssl-tls13-key-update'}else{'executors/openssl-tls13-key-update-executor'}
$sourceRelative="$component/source/main.c"
$binaryBase=if($Kind-eq'implementation'){'openssl-tls13-key-update'}else{'openssl-tls13-key-update-executor'}
$buildRoot=Join-Path $Root "$component/source/build/$RuntimeIdentifier"
New-Item -ItemType Directory -Force $buildRoot|Out-Null

if($RuntimeIdentifier-eq'win-x64'-and$IsWindows){
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
if($RuntimeIdentifier-eq'win-x64'){
    $targetBuildRoot=Join-Path $Root 'implementations/openssl-tls13-key-update/source/build/win-x64'
    $executorBuildRoot=Join-Path $Root 'executors/openssl-tls13-key-update-executor/source/build/win-x64'
    $targetOutput=Join-Path $targetBuildRoot 'openssl-tls13-key-update.exe'
    $executorOutput=Join-Path $executorBuildRoot 'openssl-tls13-key-update-executor.exe'

    if(-not($global:ProtocolLabOpenSslCrossBuildCompleted-and(Test-Path $targetOutput)-and(Test-Path $executorOutput))){
        New-Item -ItemType Directory -Force $targetBuildRoot,$executorBuildRoot|Out-Null
        $crossBuild=@"
set -eu
apk add --no-cache build-base perl mingw-w64-gcc curl tar >/dev/null
mkdir -p /src/artifacts/toolchains/openssl-3.3.0-mingw64-static
cd /src/artifacts/toolchains/openssl-3.3.0-mingw64-static
curl -fsSLo openssl-3.3.0.tar.gz '$OpenSslSourceUrl'
echo '$OpenSslSourceSha256  openssl-3.3.0.tar.gz' | sha256sum -c -
rm -rf openssl-3.3.0
tar -xzf openssl-3.3.0.tar.gz
cd openssl-3.3.0
./Configure mingw64 no-shared no-tests no-asm --cross-compile-prefix=x86_64-w64-mingw32- >/dev/null
make -j`$(nproc) build_sw >/dev/null
cd /src
x86_64-w64-mingw32-gcc -std=c11 -O2 -Wall -Wextra -Wpedantic -Werror -Iartifacts/toolchains/openssl-3.3.0-mingw64-static/openssl-3.3.0/include implementations/openssl-tls13-key-update/source/main.c artifacts/toolchains/openssl-3.3.0-mingw64-static/openssl-3.3.0/libssl.a artifacts/toolchains/openssl-3.3.0-mingw64-static/openssl-3.3.0/libcrypto.a -lws2_32 -lcrypt32 -static -o implementations/openssl-tls13-key-update/source/build/win-x64/openssl-tls13-key-update.exe
x86_64-w64-mingw32-gcc -std=c11 -O2 -Wall -Wextra -Wpedantic -Werror -Iartifacts/toolchains/openssl-3.3.0-mingw64-static/openssl-3.3.0/include executors/openssl-tls13-key-update-executor/source/main.c artifacts/toolchains/openssl-3.3.0-mingw64-static/openssl-3.3.0/libssl.a artifacts/toolchains/openssl-3.3.0-mingw64-static/openssl-3.3.0/libcrypto.a -lws2_32 -lcrypt32 -static -o executors/openssl-tls13-key-update-executor/source/build/win-x64/openssl-tls13-key-update-executor.exe
"@
        & docker run --rm -v "${mountRoot}:/src" -w /src $LinuxImage sh -lc $crossBuild|Out-Host
        if($LASTEXITCODE-ne0){throw 'Windows OpenSSL cross-build failed.'}
        $global:ProtocolLabOpenSslCrossBuildCompleted=$true
    }

    $output=if($Kind-eq'implementation'){$targetOutput}else{$executorOutput}
    if(-not(Test-Path $output)){throw "$Kind Windows binary was not materialized."}
    return [pscustomobject]@{binary=$output;buildRoot=$buildRoot;engineVersion='3.3.0';linkage='static-cross-compiled'}
}

$outputRelative="$component/source/build/linux-x64/$binaryBase"
$compile="apk add --no-cache build-base openssl-dev=3.5.7-r0 openssl-libs-static=3.5.7-r0 >/dev/null && gcc -std=c11 -O2 -Wall -Wextra -Wpedantic -Werror -static '$sourceRelative' -lssl -lcrypto -pthread -o '$outputRelative' && '$outputRelative' --self-test"
& docker run --rm -v "${mountRoot}:/src" -w /src $LinuxImage sh -lc $compile|Out-Host
if($LASTEXITCODE-ne0){throw "$Kind Linux build or self-test failed."}
$output=Join-Path $Root $outputRelative
if(-not(Test-Path $output)){throw "$Kind Linux binary was not materialized."}
return [pscustomobject]@{binary=$output;buildRoot=$buildRoot;engineVersion='3.5.7';linkage='static'}
