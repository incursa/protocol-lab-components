[CmdletBinding()]
param(
    [ValidateSet('win-x64','linux-x64')]
    [string]$RuntimeIdentifier='win-x64',
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference='Stop'
$Root=[IO.Path]::GetFullPath($Root)
$OutputRoot=[IO.Path]::GetFullPath($OutputRoot)
$componentName='go-tls13-mtls-executor'
$componentRoot=Join-Path $Root "executors/$componentName"
$sourceRoot=Join-Path $componentRoot 'source'
& go -C $sourceRoot test -count=1 ./...
if($LASTEXITCODE-ne 0){throw 'go-tls13-mtls-executor tests failed.'}
$rid=switch($RuntimeIdentifier){
    'win-x64'{@{os='windows';arch='x64';goOs='windows';goArch='amd64';name='go-tls13-mtls-executor.exe'}}
    'linux-x64'{@{os='linux';arch='x64';goOs='linux';goArch='amd64';name='go-tls13-mtls-executor'}}
}
$staging=Join-Path $OutputRoot "$componentName/$RuntimeIdentifier"
$packageRoot=Join-Path $staging 'package'
Remove-Item -LiteralPath $staging -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path (Join-Path $packageRoot "bin/$RuntimeIdentifier"),(Join-Path $packageRoot 'test-executors'),(Join-Path $packageRoot 'certs')|Out-Null
$oldOS=$env:GOOS;$oldArch=$env:GOARCH
try{
    $env:GOOS=$rid.goOs;$env:GOARCH=$rid.goArch
    & go -C $sourceRoot build -buildvcs=false -trimpath -o (Join-Path $packageRoot "bin/$RuntimeIdentifier/$($rid.name)") .
    if($LASTEXITCODE-ne 0){throw "go-tls13-mtls-executor build failed for $RuntimeIdentifier."}
}finally{$env:GOOS=$oldOS;$env:GOARCH=$oldArch}
Copy-Item (Join-Path $componentRoot 'protocol-lab-package.json') $packageRoot
Copy-Item (Join-Path $componentRoot 'test-executors/go-tls13-mtls-executor.yaml') (Join-Path $packageRoot 'test-executors')
Copy-Item (Join-Path $componentRoot 'toolchain.json') $packageRoot
Copy-Item (Join-Path $componentRoot 'certs/*') (Join-Path $packageRoot 'certs')
$binaryPath="bin/$RuntimeIdentifier/$($rid.name)"
$internal=Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw|ConvertFrom-Json
$internal.environments=@([ordered]@{os=$rid.os;arch=$rid.arch;entrypoint=[ordered]@{kind='process';path=$binaryPath;arguments=@();workingDirectory='.'}})
$internal|ConvertTo-Json -Depth 20|Set-Content (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8NoBOM
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath $packageRoot -SourceComponentPath $componentRoot -BuildConfiguration Release -RuntimeIdentifier $RuntimeIdentifier -ArtifactSuffix $RuntimeIdentifier -PreparedPackageRoot -AllowDirtySource:$AllowDirtySource
