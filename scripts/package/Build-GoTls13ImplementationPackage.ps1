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
$componentName='go-tls13'
$componentRoot=Join-Path $Root "implementations/$componentName"
$sourceRoot=Join-Path $componentRoot 'source'
& go -C $sourceRoot test -count=1 ./...
if($LASTEXITCODE-ne 0){throw 'go-tls13 tests failed.'}
$rid=switch($RuntimeIdentifier){
    'win-x64'{@{os='windows';arch='x64';goOs='windows';goArch='amd64';name='go-tls13.exe'}}
    'linux-x64'{@{os='linux';arch='x64';goOs='linux';goArch='amd64';name='go-tls13'}}
}
$staging=Join-Path $OutputRoot "$componentName/$RuntimeIdentifier"
$packageRoot=Join-Path $staging 'package'
Remove-Item -LiteralPath $staging -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path (Join-Path $packageRoot "bin/$RuntimeIdentifier"),(Join-Path $packageRoot 'implementations'),(Join-Path $packageRoot 'certs')|Out-Null
$oldOS=$env:GOOS;$oldArch=$env:GOARCH
try{
    $env:GOOS=$rid.goOs;$env:GOARCH=$rid.goArch
    & go -C $sourceRoot build -buildvcs=false -trimpath -o (Join-Path $packageRoot "bin/$RuntimeIdentifier/$($rid.name)") .
    if($LASTEXITCODE-ne 0){throw "go-tls13 build failed for $RuntimeIdentifier."}
}finally{$env:GOOS=$oldOS;$env:GOARCH=$oldArch}
Copy-Item (Join-Path $componentRoot 'protocol-lab-package.json') $packageRoot
Copy-Item (Join-Path $componentRoot 'implementations/go-tls13.yaml') (Join-Path $packageRoot 'implementations')
Copy-Item (Join-Path $componentRoot 'toolchain.json') $packageRoot
Copy-Item (Join-Path $componentRoot 'certs/*') (Join-Path $packageRoot 'certs')
$binaryPath="bin/$RuntimeIdentifier/$($rid.name)"
$implementationPath=Join-Path $packageRoot 'implementations/go-tls13.yaml'
$implementation=Get-Content $implementationPath -Raw
$implementation=$implementation -replace '(?m)^executable:.*$',("executable: " + $binaryPath)
$implementation=$implementation -replace '(?m)^entrypoint:.*$',("entrypoint: {kind: process, path: " + $binaryPath + ", arguments: [], workingDirectory: .}")
Set-Content $implementationPath $implementation -Encoding utf8NoBOM
$internal=Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw|ConvertFrom-Json
$internal.environments=@([ordered]@{os=$rid.os;arch=$rid.arch;entrypoint=[ordered]@{kind='process';path=$binaryPath;arguments=@();workingDirectory='.'}})
$internal|ConvertTo-Json -Depth 20|Set-Content (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8NoBOM
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath $packageRoot -SourceComponentPath $componentRoot -BuildConfiguration Release -RuntimeIdentifier $RuntimeIdentifier -ArtifactSuffix $RuntimeIdentifier -PreparedPackageRoot -AllowDirtySource:$AllowDirtySource
