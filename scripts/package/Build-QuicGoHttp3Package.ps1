[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages')
)

$ErrorActionPreference = 'Stop'
$componentPath = Join-Path $Root 'implementations/quic-go-http3'
$stagedSource = Join-Path $OutputRoot 'tmp/quic-go-http3-source'

Remove-Item -LiteralPath $stagedSource -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $stagedSource | Out-Null
Copy-Item -Path (Join-Path $componentPath '*') -Destination $stagedSource -Recurse -Force

$runtimeRoot = Join-Path $stagedSource 'runtimes/linux-x64'
New-Item -ItemType Directory -Force -Path $runtimeRoot | Out-Null

$previousGoos = $env:GOOS
$previousGoarch = $env:GOARCH
$previousCgo = $env:CGO_ENABLED
try {
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    Push-Location $stagedSource
    try {
        & go build `
            -buildvcs=false `
            -trimpath `
            -ldflags '-s -w -X main.quicGoVersion=v0.60.0' `
            -o (Join-Path $runtimeRoot 'quic-go-http3-server') `
            ./src
        if ($LASTEXITCODE -ne 0) {
            throw "quic-go HTTP/3 Linux x64 binary build failed with exit code $LASTEXITCODE."
        }
    }
    finally {
        Pop-Location
    }

    & (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') `
        -Root $Root `
        -OutputRoot $OutputRoot `
        -ComponentPath $stagedSource
}
finally {
    $env:GOOS = $previousGoos
    $env:GOARCH = $previousGoarch
    $env:CGO_ENABLED = $previousCgo
    Remove-Item -LiteralPath $stagedSource -Recurse -Force -ErrorAction SilentlyContinue
}
