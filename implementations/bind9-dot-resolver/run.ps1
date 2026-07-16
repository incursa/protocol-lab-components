[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$image=if($env:PLAB_SECURE_DNS_IMAGE){$env:PLAB_SECURE_DNS_IMAGE}else{'incursa-protocol-lab-bind9-dot-resolver:0.1.1'}
$port=if($env:PLAB_SECURE_DNS_PORT){[int]$env:PLAB_SECURE_DNS_PORT}else{20562}
$controlPort=if($env:PLAB_RESOLVER_CONTROL_PORT){[int]$env:PLAB_RESOLVER_CONTROL_PORT}else{$port+1}
Push-Location $PSScriptRoot
try {
  if($env:PLAB_SKIP_BUILD-ne'true'){& docker build --pull -t $image docker;if($LASTEXITCODE-ne 0){throw 'Docker build failed.'}}
  if($env:PLAB_PROOF_ONLY-eq'true'){& docker run --rm --entrypoint named $image -V;exit $LASTEXITCODE}
  & docker run --rm -p "${port}:853/tcp" -p "${controlPort}:854/tcp" $image
  exit $LASTEXITCODE
} finally {Pop-Location}
