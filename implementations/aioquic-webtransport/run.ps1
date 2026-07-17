[CmdletBinding()]
param([string]$Image='incursa-protocol-lab-aioquic-webtransport:0.1.3',[int]$Port=5462,[switch]$SkipBuild,[switch]$PlanOnly)
$ErrorActionPreference='Stop'
$scenario=if($env:PLAB_SCENARIO_ID){$env:PLAB_SCENARIO_ID}else{'webtransport.session-bidi-echo'}
if($scenario-notin@('webtransport.session-bidi-echo','webtransport.session-datagram-echo')){[ordered]@{schemaVersion='protocol-lab.unsupported.v1';status='unsupported';scenarioId=$scenario;implementationId='aioquic-webtransport';supportedScenarios=@('webtransport.session-bidi-echo','webtransport.session-datagram-echo')}|ConvertTo-Json -Compress;exit 3}
Push-Location $PSScriptRoot
try{
  if($PlanOnly){[ordered]@{schemaVersion='protocol-lab.endpoint-plan.v1';implementationId='aioquic-webtransport';packageVersion='0.1.3';upstreamVersion='1.3.0';scenarioId=$scenario;image=$Image;hostPort=$Port;containerPort=4433;protocol='webtransport-over-h3'}|ConvertTo-Json -Compress;return}
  if(-not $SkipBuild){& docker build --pull -f docker/Dockerfile -t $Image .;if($LASTEXITCODE-ne 0){throw 'Docker build failed.'}}
  $version=(& docker run --rm $Image --version).Trim();if($version-ne'aioquic-webtransport 0.1.3 aioquic 1.3.0'){throw "Version proof mismatch: $version"}
  & docker run --rm -p "${Port}:4433/udp" $Image
  if($LASTEXITCODE-ne 0){throw "Server failed with exit code $LASTEXITCODE."}
}finally{Pop-Location}
