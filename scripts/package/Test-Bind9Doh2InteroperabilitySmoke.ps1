[CmdletBinding()]
param(
  [string]$Root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path,
  [string]$SmokeRoot=(Join-Path $Root 'artifacts/bind9-doh2-interoperability-smoke'),
  [int]$Port=20443,
  [string]$Image='incursa-protocol-lab-bind9-doh2:0.1.0'
)
$ErrorActionPreference='Stop'
$component=Join-Path $Root 'implementations/bind9-doh2'
$executorSource=Join-Path $Root 'executors/go-dns-doh2-executor/source'
$container="plab-bind9-doh2-smoke-$PID"
New-Item -ItemType Directory -Force $SmokeRoot|Out-Null
Push-Location $component
try {
  & docker build --pull -t $Image docker
  if($LASTEXITCODE-ne0){throw 'BIND 9 DoH2 image build failed.'}
  $version=(& docker run --rm --entrypoint named $Image -V 2>&1|Out-String)
  if($LASTEXITCODE-ne0-or$version-notmatch'BIND 9.20.24'-or$version-notmatch'libnghttp2'){throw "BIND version/capability proof failed: $version"}
  & docker run --rm -d --name $container -p "${Port}:443/tcp" $Image|Out-Null
  if($LASTEXITCODE-ne0){throw 'BIND 9 DoH2 container start failed.'}
  $ready=$false
  for($i=0;$i-lt100;$i++){
    $client=[Net.Sockets.TcpClient]::new()
    try{$client.Connect('127.0.0.1',$Port);$ready=$true;break}catch{Start-Sleep -Milliseconds 100}finally{$client.Dispose()}
  }
  if(-not$ready){throw 'BIND 9 DoH2 listener did not become ready.'}
  $env:PLAB_EXECUTOR_ID='go-dns-doh2-executor'
  $env:PLAB_EXECUTOR_VERSION='0.2.0'
  $env:PLAB_LOAD_GENERATOR_ID='go-dns-doh2-load'
  $env:PLAB_LOAD_GENERATOR_VERSION='0.2.0'
  $env:PLAB_PROTOCOL='doh2'
  $env:PLAB_PROTOCOL_VARIANT='doh-h2-tls-alpn'
  $env:PLAB_SCENARIO_ID='dns.doh2.interoperability.query.a'
  Push-Location $executorSource
  try {
    & go run . --target-address "https://127.0.0.1:${Port}/dns-query" --root-certificate (Join-Path $Root 'implementations/go-dns-dot/certs/root.pem') --output-dir $SmokeRoot --validation-only
    if($LASTEXITCODE-ne0){throw "DoH2 executor validation exited $LASTEXITCODE."}
  } finally { Pop-Location }
  $proof=Get-Content (Join-Path $SmokeRoot 'protocol-proof.json') -Raw|ConvertFrom-Json
  if($proof.observedProtocol-ne'doh2'-or$proof.fallbackDetected-or$proof.http.httpVersion-ne'HTTP/2.0'-or$proof.http.responseStatus-ne200-or$proof.dns.answer-ne'192.0.2.1'-or$proof.dns.authoritativeAnswer-ne$true){throw 'BIND 9 DoH2 interoperability proof gate failed.'}
  if(-not$proof.tls.platformProvenance.goos-or-not$proof.tls.platformProvenance.goarch-or-not$proof.tls.platformProvenance.goVersion-or-not$proof.tls.accelerationProvenance.mode){throw 'TLS provenance gate failed.'}
  [pscustomobject]@{status='passed';scenarioId=$env:PLAB_SCENARIO_ID;implementationId='bind9-doh2';httpVersion=$proof.http.httpVersion;responseCacheControl=$proof.http.responseCacheControl;fallbackDetected=$proof.fallbackDetected;image=$Image;bindVersion='9.20.24'}|ConvertTo-Json|Set-Content (Join-Path $SmokeRoot 'smoke-summary.json') -Encoding utf8NoBOM
} finally {
  & docker rm -f $container 2>$null|Out-Null
  Pop-Location
}
