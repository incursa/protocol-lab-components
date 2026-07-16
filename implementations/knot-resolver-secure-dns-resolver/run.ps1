[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$image=if($env:PLAB_SECURE_DNS_IMAGE){$env:PLAB_SECURE_DNS_IMAGE}else{'incursa-protocol-lab-knot-resolver-secure-dns-resolver:0.1.5'}
$dotPort=if($env:PLAB_SECURE_DNS_PORT){[int]$env:PLAB_SECURE_DNS_PORT}else{20566}
$dotControlPort=if($env:PLAB_RESOLVER_CONTROL_PORT){[int]$env:PLAB_RESOLVER_CONTROL_PORT}else{$dotPort+1}
$doh2Port=if($env:PLAB_DOH2_PORT){[int]$env:PLAB_DOH2_PORT}else{$dotPort+2}
$doh2ControlPort=if($env:PLAB_DOH2_RESOLVER_CONTROL_PORT){[int]$env:PLAB_DOH2_RESOLVER_CONTROL_PORT}else{$doh2Port+1}
Push-Location $PSScriptRoot
try {
  if($env:PLAB_SKIP_BUILD-ne'true'){& docker build --pull -t $image docker;if($LASTEXITCODE-ne 0){throw 'Docker build failed.'}}
  if($env:PLAB_PROOF_ONLY-eq'true'){& docker run --rm --entrypoint /usr/bin/knot-resolver $image --version;exit $LASTEXITCODE}
  & docker run --rm -p "${dotPort}:853/tcp" -p "${dotControlPort}:444/tcp" -p "${doh2Port}:443/tcp" -p "${doh2ControlPort}:444/tcp" $image
  exit $LASTEXITCODE
} finally {Pop-Location}
