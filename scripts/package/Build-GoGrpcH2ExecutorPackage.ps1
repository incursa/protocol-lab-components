[CmdletBinding()]
param(
    [ValidateSet('win-x64','linux-x64')][string]$RuntimeIdentifier='win-x64',
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),
    [switch]$SkipSmoke,
    [switch]$AllowDirtySource
)
$ErrorActionPreference='Stop'
$Root=[IO.Path]::GetFullPath($Root); $OutputRoot=[IO.Path]::GetFullPath($OutputRoot)
$componentName='go-grpc-h2-executor'; $componentRoot=Join-Path $Root "executors/$componentName"; $sourceRoot=Join-Path $componentRoot 'source'
$rid=if($RuntimeIdentifier -eq 'win-x64'){@{os='windows';arch='x64';goos='windows';goarch='amd64';name='go-grpc-h2-executor.exe'}}else{@{os='linux';arch='x64';goos='linux';goarch='amd64';name='go-grpc-h2-executor'}}
if(-not $SkipSmoke){& go -C $sourceRoot test -count=1 .; if($LASTEXITCODE-ne 0){throw 'go test failed.'}}
$staging=Join-Path $OutputRoot "$componentName/$RuntimeIdentifier"; $packageRoot=Join-Path $staging 'package'; Remove-Item -LiteralPath $staging -Recurse -Force -ErrorAction SilentlyContinue
$bin=Join-Path $packageRoot "bin/$RuntimeIdentifier"; New-Item -ItemType Directory -Force -Path $bin,(Join-Path $packageRoot 'test-executors'),(Join-Path $packageRoot 'certs'),(Join-Path $packageRoot 'third-party')|Out-Null
$oldOS=$env:GOOS; $oldArch=$env:GOARCH; try{$env:GOOS=$rid.goos;$env:GOARCH=$rid.goarch;& go -C $sourceRoot build -buildvcs=false -trimpath -o (Join-Path $bin $rid.name) .;if($LASTEXITCODE-ne 0){throw 'go build failed.'}}finally{$env:GOOS=$oldOS;$env:GOARCH=$oldArch}
Copy-Item (Join-Path $componentRoot 'protocol-lab-package.json') $packageRoot; Copy-Item (Join-Path $componentRoot 'test-executors/go-grpc-h2-executor.yaml') (Join-Path $packageRoot 'test-executors'); Copy-Item (Join-Path $componentRoot 'toolchain.json') $packageRoot; Copy-Item (Join-Path $componentRoot 'certs/root.pem') (Join-Path $packageRoot 'certs'); Copy-Item (Join-Path $componentRoot 'third-party/golang-x-net-LICENSE.txt') (Join-Path $packageRoot 'third-party')
$internal=Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw|ConvertFrom-Json; $internal.environments=@([ordered]@{os=$rid.os;arch=$rid.arch;entrypoint=[ordered]@{kind='process';path="bin/$RuntimeIdentifier/$($rid.name)";arguments=@();workingDirectory='.'}}); $internal.dependencies.requiresPwsh=$false;$internal.dependencies.requiresBash=$false;$internal.dependencies.requiresGo=$false;$internal.dependencies.requiredCapabilities=@();$internal|ConvertTo-Json -Depth 20|Set-Content (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8NoBOM
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath $packageRoot -SourceComponentPath $componentRoot -ArtifactSuffix $RuntimeIdentifier -BuildConfiguration Release -RuntimeIdentifier $RuntimeIdentifier -PreparedPackageRoot -AllowDirtySource:$AllowDirtySource
