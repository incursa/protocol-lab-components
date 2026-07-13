[CmdletBinding()]
param([Parameter(ValueFromRemainingArguments=$true)][string[]]$RemainingArguments)
$env:PLAB_TLS_ROOT_CERTIFICATE_PATH = Join-Path $PSScriptRoot 'certs/root.pem'
$binary = Join-Path $PSScriptRoot 'bin/win-x64/go-http1-websocket-tls-executor.exe'
& $binary @RemainingArguments
exit $LASTEXITCODE
