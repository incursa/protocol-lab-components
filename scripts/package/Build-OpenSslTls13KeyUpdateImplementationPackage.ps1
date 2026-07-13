[CmdletBinding()]
param(
    [ValidateSet('win-x64','linux-x64')][string]$RuntimeIdentifier='win-x64',
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),
    [switch]$AllowDirtySource
)

$ErrorActionPreference='Stop'
$Root=[IO.Path]::GetFullPath($Root);$OutputRoot=[IO.Path]::GetFullPath($OutputRoot)
$componentName='openssl-tls13-key-update';$componentRoot=Join-Path $Root "implementations/$componentName"
$build=& (Join-Path $PSScriptRoot 'Invoke-OpenSslTls13KeyUpdateBuild.ps1') -Kind implementation -RuntimeIdentifier $RuntimeIdentifier -Root $Root
$staging=Join-Path $OutputRoot "$componentName/$RuntimeIdentifier";$packageRoot=Join-Path $staging package
Remove-Item -LiteralPath $staging -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force (Join-Path $packageRoot "bin/$RuntimeIdentifier"),(Join-Path $packageRoot implementations),(Join-Path $packageRoot certs),(Join-Path $packageRoot 'third-party-licenses/openssl')|Out-Null
$binaryName=if($RuntimeIdentifier-eq'win-x64'){'openssl-tls13-key-update.exe'}else{'openssl-tls13-key-update'}
Copy-Item -LiteralPath $build.binary -Destination (Join-Path $packageRoot "bin/$RuntimeIdentifier/$binaryName")
if($RuntimeIdentifier-eq'win-x64'){Copy-Item -LiteralPath (Join-Path $build.buildRoot 'libcrypto-3-x64__.dll'),(Join-Path $build.buildRoot 'libssl-3-x64__.dll') -Destination (Join-Path $packageRoot "bin/$RuntimeIdentifier")}
Copy-Item -LiteralPath (Join-Path $componentRoot protocol-lab-package.json),(Join-Path $componentRoot toolchain.json),(Join-Path $componentRoot THIRD-PARTY-NOTICES.md) -Destination $packageRoot
Copy-Item -LiteralPath (Join-Path $componentRoot 'implementations/openssl-tls13-key-update.yaml') -Destination (Join-Path $packageRoot implementations)
Copy-Item -LiteralPath (Join-Path $componentRoot 'certs/leaf.pem'),(Join-Path $componentRoot 'certs/leaf-key.pem') -Destination (Join-Path $packageRoot certs)
Copy-Item -LiteralPath (Join-Path $Root LICENSE) -Destination (Join-Path $packageRoot 'third-party-licenses/openssl/LICENSE-APACHE-2.0.txt')
$internal=Get-Content (Join-Path $componentRoot protocol-lab.internal.json) -Raw|ConvertFrom-Json
$os=if($RuntimeIdentifier-eq'win-x64'){'windows'}else{'linux'};$internal.environments=@([ordered]@{os=$os;arch='x64';entrypoint=[ordered]@{kind='process';path="bin/$RuntimeIdentifier/$binaryName";arguments=@();workingDirectory='.'}})
$internal|ConvertTo-Json -Depth 20|Set-Content (Join-Path $packageRoot protocol-lab.internal.json) -Encoding utf8NoBOM
& (Join-Path $PSScriptRoot Build-ProtocolLabComponentPackage.ps1) -Root $Root -OutputRoot $OutputRoot -ComponentPath $packageRoot -SourceComponentPath $componentRoot -BuildConfiguration Release -RuntimeIdentifier $RuntimeIdentifier -ArtifactSuffix $RuntimeIdentifier -PreparedPackageRoot -AllowDirtySource:$AllowDirtySource
