[CmdletBinding()]
param([Parameter(ValueFromRemainingArguments=$true)][string[]]$RemainingArguments)
$binary = Join-Path $PSScriptRoot 'bin/win-x64/go-http1-websocket-executor.exe'
& $binary @RemainingArguments
exit $LASTEXITCODE
