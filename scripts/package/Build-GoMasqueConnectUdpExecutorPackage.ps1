[CmdletBinding()]
param([ValidateSet('win-x64','linux-x64')][string]$RuntimeIdentifier='win-x64',[string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,[string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),[switch]$AllowDirtySource)
$ErrorActionPreference='Stop'
$Root=[IO.Path]::GetFullPath($Root)
$componentRoot=Join-Path $Root 'executors/go-masque-connect-udp-executor'
$sourceRoot=Join-Path $componentRoot 'source'
& go -C $sourceRoot test -count=1 ./...
if($LASTEXITCODE-ne 0){throw 'MASQUE executor tests failed.'}
$rid=switch($RuntimeIdentifier){'win-x64'{@{os='windows';arch='x64';goOs='windows';name='go-masque-connect-udp-executor.exe'}}'linux-x64'{@{os='linux';arch='x64';goOs='linux';name='go-masque-connect-udp-executor'}}}
$stage=Join-Path $OutputRoot "go-masque-connect-udp-executor/$RuntimeIdentifier"
$packageRoot=Join-Path $stage 'package'
Remove-Item -LiteralPath $stage -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force (Join-Path $packageRoot "bin/$RuntimeIdentifier"),(Join-Path $packageRoot 'test-executors'),(Join-Path $packageRoot 'certs'),(Join-Path $packageRoot 'third-party')|Out-Null
$oldOS=$env:GOOS;$oldArch=$env:GOARCH;$oldCgo=$env:CGO_ENABLED
try{$env:GOOS=$rid.goOs;$env:GOARCH='amd64';$env:CGO_ENABLED='0';& go -C $sourceRoot build -buildvcs=false -trimpath -o (Join-Path $packageRoot "bin/$RuntimeIdentifier/$($rid.name)") .;if($LASTEXITCODE-ne 0){throw 'MASQUE executor build failed.'}}
finally{$env:GOOS=$oldOS;$env:GOARCH=$oldArch;$env:CGO_ENABLED=$oldCgo}
Copy-Item (Join-Path $componentRoot 'protocol-lab-package.json'),(Join-Path $componentRoot 'toolchain.json'),(Join-Path $componentRoot 'protocol-lab.internal.json') $packageRoot
Copy-Item (Join-Path $componentRoot 'test-executors/go-masque-connect-udp-executor.yaml') (Join-Path $packageRoot 'test-executors')
Copy-Item (Join-Path $componentRoot 'certs/root.pem') (Join-Path $packageRoot 'certs')
$moduleCache=(& go env GOMODCACHE).Trim()
$licenses=@(
  @{source='github.com/quic-go/masque-go@v0.4.0/LICENSE';target='masque-go-LICENSE.txt'},
  @{source='github.com/quic-go/quic-go@v0.60.0/LICENSE';target='quic-go-LICENSE.txt'},
  @{source='github.com/dunglas/httpsfv@v1.1.0/LICENSE';target='httpsfv-LICENSE.txt'},
  @{source='github.com/quic-go/qpack@v0.6.0/LICENSE.md';target='qpack-LICENSE.md'},
  @{source='github.com/yosida95/uritemplate/v3@v3.0.2/LICENSE';target='uritemplate-LICENSE.txt'}
)
foreach($license in $licenses){Copy-Item (Join-Path $moduleCache $license.source) (Join-Path $packageRoot "third-party/$($license.target)")}
$internal=Get-Content (Join-Path $packageRoot 'protocol-lab.internal.json') -Raw|ConvertFrom-Json
$internal.environments=@([ordered]@{os=$rid.os;arch=$rid.arch;entrypoint=[ordered]@{kind='process';path="bin/$RuntimeIdentifier/$($rid.name)";arguments=@();workingDirectory='.'}})
$internal|ConvertTo-Json -Depth 20|Set-Content (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8NoBOM
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath $packageRoot -SourceComponentPath $componentRoot -BuildConfiguration Release -RuntimeIdentifier $RuntimeIdentifier -ArtifactSuffix $RuntimeIdentifier -PreparedPackageRoot -AllowDirtySource:$AllowDirtySource
