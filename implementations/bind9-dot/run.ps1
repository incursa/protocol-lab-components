[CmdletBinding()]
param([string]$Image='incursa-protocol-lab-bind9-dot:0.1.0',[int]$Port=20530,[switch]$SkipBuild,[switch]$PlanOnly,[switch]$ProofOnly,[string]$OutputRoot='artifacts/bind9-dot')
$ErrorActionPreference='Stop'
$out=if([IO.Path]::IsPathRooted($OutputRoot)){$OutputRoot}else{Join-Path $PSScriptRoot $OutputRoot};New-Item -ItemType Directory -Force $out|Out-Null
$build=@('build','--pull','-t',$Image,'docker');$proof=@('run','--rm','--entrypoint','named',$Image,'-V');$run=@('run','--rm','-p',"${Port}:853/tcp",$Image)
@('docker '+($build-join' '),'docker '+($proof-join' '),'docker '+($run-join' '))|Set-Content (Join-Path $out 'command.txt')
if($PlanOnly){@{status='planned';image=$Image;protocol='dot'}|ConvertTo-Json|Set-Content (Join-Path $out 'result.json');return}
Push-Location $PSScriptRoot
try{if(-not $SkipBuild){& docker @build;if($LASTEXITCODE-ne 0){throw 'Docker build failed.'}};$v=(& docker @proof 2>&1|Out-String).Trim();$v|Set-Content (Join-Path $out 'version.txt');if($v-notmatch [regex]::Escape('BIND 9.20.24')){throw "Version proof failed: $v"};if($ProofOnly){@{status='proved';version=$v}|ConvertTo-Json|Set-Content (Join-Path $out 'result.json');return};& docker @run}finally{Pop-Location}

