$ErrorActionPreference='Stop'
& (Join-Path $PSScriptRoot 'bin/win-x64/go-dns-doh3.exe') @args
exit $LASTEXITCODE
