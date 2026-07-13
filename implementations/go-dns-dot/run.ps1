[CmdletBinding()]
param([int]$Port=18530)
$ErrorActionPreference='Stop'
$env:PLAB_DOT_LISTEN="127.0.0.1:$Port"
$rid=if($IsWindows){'win-x64'}elseif($IsLinux){'linux-x64'}else{throw 'go-dns-dot supports Windows and Linux only.'}
$name=if($IsWindows){'go-dns-dot.exe'}else{'go-dns-dot'}
& (Join-Path $PSScriptRoot "bin/$rid/$name")
