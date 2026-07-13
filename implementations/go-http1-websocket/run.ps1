[CmdletBinding()]
param([Parameter(ValueFromRemainingArguments=$true)][string[]]$RemainingArguments)
$binary = Join-Path $PSScriptRoot 'bin/win-x64/go-http1-websocket.exe'
& $binary @RemainingArguments
exit $LASTEXITCODE
