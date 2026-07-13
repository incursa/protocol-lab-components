[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$rid=if($IsWindows){'win-x64'}elseif($IsLinux){'linux-x64'}else{throw 'go-dns-doh2 supports Windows and Linux only.'}
$name=if($IsWindows){'go-dns-doh2.exe'}else{'go-dns-doh2'}
& (Join-Path $PSScriptRoot "bin/$rid/$name")
exit $LASTEXITCODE
