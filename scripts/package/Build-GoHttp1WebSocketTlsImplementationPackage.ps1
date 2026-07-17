[CmdletBinding()]
param(
    [ValidateSet('win-x64','linux-x64')][string]$RuntimeIdentifier = 'win-x64',
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot = (Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)
$ErrorActionPreference = 'Stop'
$Root = [IO.Path]::GetFullPath($Root)
$OutputRoot = [IO.Path]::GetFullPath($OutputRoot)
$componentRoot = Join-Path $Root 'implementations/go-http1-websocket-tls'
$sourceRoot = Join-Path $componentRoot 'source'
& go -C $sourceRoot test -count=1 ./...
if ($LASTEXITCODE -ne 0) { throw 'go-http1-websocket-tls tests failed.' }
$rid = switch ($RuntimeIdentifier) {
    'win-x64' { @{os='windows';arch='x64';goOs='windows';goArch='amd64';name='go-http1-websocket-tls.exe'} }
    'linux-x64' { @{os='linux';arch='x64';goOs='linux';goArch='amd64';name='go-http1-websocket-tls'} }
}
$stagingRoot = Join-Path $OutputRoot "go-http1-websocket-tls/$RuntimeIdentifier"
$packageRoot = Join-Path $stagingRoot 'package'
$packageBin = Join-Path $packageRoot "bin/$RuntimeIdentifier"
Remove-Item -LiteralPath $stagingRoot -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $packageBin,(Join-Path $packageRoot 'implementations'),(Join-Path $packageRoot 'certs') | Out-Null
$oldGoOs=$env:GOOS; $oldGoArch=$env:GOARCH
try {
    $env:GOOS=$rid.goOs; $env:GOARCH=$rid.goArch
    & go -C $sourceRoot build -buildvcs=false -trimpath -o (Join-Path $packageBin $rid.name) .
    if ($LASTEXITCODE -ne 0) { throw 'go-http1-websocket-tls build failed.' }
} finally { $env:GOOS=$oldGoOs; $env:GOARCH=$oldGoArch }
Copy-Item (Join-Path $componentRoot 'protocol-lab-package.json') $packageRoot
Copy-Item (Join-Path $componentRoot 'implementations/go-http1-websocket-tls.yaml') (Join-Path $packageRoot 'implementations')
Copy-Item (Join-Path $componentRoot 'certs/leaf.pem') (Join-Path $packageRoot 'certs')
Copy-Item (Join-Path $componentRoot 'certs/leaf-key.pem') (Join-Path $packageRoot 'certs')
$implementationPath = Join-Path $packageRoot 'implementations/go-http1-websocket-tls.yaml'
$binaryPath = "bin/$RuntimeIdentifier/$($rid.name)"
$implementation = Get-Content $implementationPath -Raw
$implementation = $implementation -replace '(?m)^executable:.*$',("executable: " + $binaryPath)
$implementation = $implementation -replace '(?m)^entrypoint:.*$',("entrypoint: {kind: process, path: " + $binaryPath + ", arguments: [], workingDirectory: .}")
Set-Content $implementationPath $implementation -Encoding utf8NoBOM
$internal = Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw | ConvertFrom-Json
$internal.environments = @(@{os=$rid.os;arch=$rid.arch;entrypoint=@{kind='process';path="bin/$RuntimeIdentifier/$($rid.name)";arguments=@();workingDirectory='.'}})
$internal | ConvertTo-Json -Depth 20 | Set-Content (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8NoBOM
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath $packageRoot -SourceComponentPath $componentRoot -RuntimeIdentifier $RuntimeIdentifier -ArtifactSuffix $RuntimeIdentifier -PreparedPackageRoot -AllowDirtySource:$AllowDirtySource
