[CmdletBinding()]
param(
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$LinuxImage='alpine@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412'
)

$ErrorActionPreference='Stop'
$Root=[IO.Path]::GetFullPath($Root)
foreach($kind in @('implementation','executor')){
    & (Join-Path $PSScriptRoot Invoke-OpenSslTls13KeyUpdateBuild.ps1) -Kind $kind -RuntimeIdentifier win-x64 -Root $Root|Out-Null
}
$mountRoot=$Root.Replace('\','/')
$command=@'
apk add --no-cache build-base openssl-dev=3.5.7-r0 >/dev/null
mkdir -p artifacts/key-update-strict
gcc -std=c11 -O1 -g -Wall -Wextra -Wpedantic -Werror -fanalyzer implementations/openssl-tls13-key-update/source/main.c -lssl -lcrypto -pthread -o artifacts/key-update-strict/target
gcc -std=c11 -O1 -g -Wall -Wextra -Wpedantic -Werror -fanalyzer executors/openssl-tls13-key-update-executor/source/main.c -lssl -lcrypto -pthread -o artifacts/key-update-strict/executor
artifacts/key-update-strict/target --self-test
artifacts/key-update-strict/executor --self-test
'@
& docker run --rm -v "${mountRoot}:/src" -w /src $LinuxImage sh -lc $command
if($LASTEXITCODE-ne0){throw 'OpenSSL KeyUpdate Linux GCC analyzer strict source gate failed.'}
Write-Output 'OpenSSL TLS KeyUpdate source strict gates passed for target and executor.'
