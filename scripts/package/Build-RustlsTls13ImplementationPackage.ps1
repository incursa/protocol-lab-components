[CmdletBinding()]
param(
    [ValidateSet('win-x64','linux-x64')][string]$RuntimeIdentifier='win-x64',
    [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
    [string]$OutputRoot=(Join-Path $Root 'artifacts/packages'),
    [string]$Toolchain='stable-x86_64-pc-windows-gnu',
    [switch]$AllowDirtySource
)

$ErrorActionPreference='Stop'
$Root=[IO.Path]::GetFullPath($Root);$OutputRoot=[IO.Path]::GetFullPath($OutputRoot)
$componentName='rustls-tls13';$componentRoot=Join-Path $Root "implementations/$componentName";$sourceRoot=Join-Path $componentRoot 'source'
& cargo "+$Toolchain" test --locked --manifest-path (Join-Path $sourceRoot 'Cargo.toml')
if($LASTEXITCODE-ne 0){throw 'rustls TLS 1.3 target tests failed.'}
$rid=switch($RuntimeIdentifier){
    'win-x64'{@{os='windows';arch='x64';target=$null;name='rustls-tls13.exe';source='target/release/protocol-lab-rustls-tls13-target.exe'}}
    'linux-x64'{@{os='linux';arch='x64';target='x86_64-unknown-linux-musl';name='rustls-tls13';source='target/x86_64-unknown-linux-musl/release/protocol-lab-rustls-tls13-target'}}
}
$buildArgs=@("+$Toolchain",'build','--locked','--release','--manifest-path',(Join-Path $sourceRoot 'Cargo.toml'))
if($rid.target){$buildArgs+=@('--target',$rid.target)}
$savedLinker=$env:CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_LINKER
try{
    if($RuntimeIdentifier-eq'linux-x64'-and$IsWindows){
        $sysroot=& rustc "+$Toolchain" --print sysroot
        if($LASTEXITCODE-ne 0){throw 'Unable to resolve the pinned Rust sysroot.'}
        $env:CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_LINKER=Join-Path $sysroot 'lib/rustlib/x86_64-pc-windows-gnu/bin/rust-lld.exe'
        if(-not(Test-Path $env:CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_LINKER)){throw 'rust-lld is unavailable for the Linux musl package build.'}
    }
    & cargo @buildArgs
    if($LASTEXITCODE-ne 0){throw "rustls TLS 1.3 target build failed for $RuntimeIdentifier."}
}finally{$env:CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_LINKER=$savedLinker}
$staging=Join-Path $OutputRoot "$componentName/$RuntimeIdentifier";$packageRoot=Join-Path $staging 'package'
Remove-Item -LiteralPath $staging -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force (Join-Path $packageRoot "bin/$RuntimeIdentifier"),(Join-Path $packageRoot 'implementations'),(Join-Path $packageRoot 'certs')|Out-Null
Copy-Item -LiteralPath (Join-Path $sourceRoot $rid.source) -Destination (Join-Path $packageRoot "bin/$RuntimeIdentifier/$($rid.name)")
Copy-Item -LiteralPath (Join-Path $componentRoot 'protocol-lab-package.json'),(Join-Path $componentRoot 'toolchain.json'),(Join-Path $componentRoot 'THIRD-PARTY-NOTICES.md') -Destination $packageRoot
Copy-Item -LiteralPath (Join-Path $componentRoot 'implementations/rustls-tls13.yaml') -Destination (Join-Path $packageRoot 'implementations')
Copy-Item -LiteralPath (Join-Path $sourceRoot 'Cargo.lock') -Destination $packageRoot
Copy-Item -LiteralPath (Join-Path $componentRoot 'certs/leaf.pem'),(Join-Path $componentRoot 'certs/leaf-key.pem') -Destination (Join-Path $packageRoot 'certs')
& (Join-Path $PSScriptRoot 'Copy-RustThirdPartyLicenses.ps1') -ManifestPath (Join-Path $sourceRoot 'Cargo.toml') -Destination (Join-Path $packageRoot 'third-party-licenses') -Toolchain $Toolchain
$binaryPath="bin/$RuntimeIdentifier/$($rid.name)";$entry=Join-Path $packageRoot 'implementations/rustls-tls13.yaml'
$text=Get-Content $entry -Raw;$text=$text-replace'(?m)^executable:.*$',("executable: "+$binaryPath);$text=$text-replace'(?m)^entrypoint:.*$',("entrypoint: {kind: process, path: "+$binaryPath+", arguments: [], workingDirectory: .}");Set-Content $entry $text -Encoding utf8NoBOM
$internal=Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw|ConvertFrom-Json
$internal.environments=@([ordered]@{os=$rid.os;arch=$rid.arch;entrypoint=[ordered]@{kind='process';path=$binaryPath;arguments=@();workingDirectory='.'}})
$internal|ConvertTo-Json -Depth 20|Set-Content (Join-Path $packageRoot 'protocol-lab.internal.json') -Encoding utf8NoBOM
& (Join-Path $PSScriptRoot 'Build-ProtocolLabComponentPackage.ps1') -Root $Root -OutputRoot $OutputRoot -ComponentPath $packageRoot -SourceComponentPath $componentRoot -BuildConfiguration Release -RuntimeIdentifier $RuntimeIdentifier -ArtifactSuffix $RuntimeIdentifier -PreparedPackageRoot -AllowDirtySource:$AllowDirtySource
