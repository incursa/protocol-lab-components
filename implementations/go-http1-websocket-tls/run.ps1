[CmdletBinding()]
param([Parameter(ValueFromRemainingArguments=$true)][string[]]$RemainingArguments)
$env:PLAB_TLS_CERTIFICATE_PATH = Join-Path $PSScriptRoot 'certs/leaf.pem'
$env:PLAB_TLS_PRIVATE_KEY_PATH = Join-Path $PSScriptRoot 'certs/leaf-key.pem'
$binary = Join-Path $PSScriptRoot 'bin/win-x64/go-http1-websocket-tls.exe'
& $binary @RemainingArguments
exit $LASTEXITCODE
