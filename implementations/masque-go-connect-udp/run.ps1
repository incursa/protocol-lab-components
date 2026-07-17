[CmdletBinding()]
param([string]$Image='incursa-protocol-lab-masque-go-connect-udp:0.1.2',[int]$Port=5471,[switch]$SkipBuild,[switch]$PlanOnly)
$ErrorActionPreference='Stop'
$scenario=if($env:PLAB_SCENARIO_ID){$env:PLAB_SCENARIO_ID}else{'masque.connect-udp-tunnel'}
if($scenario-ne'masque.connect-udp-tunnel'){[ordered]@{schemaVersion='protocol-lab.unsupported.v1';status='unsupported';scenarioId=$scenario;implementationId='masque-go-connect-udp';supportedScenarios=@('masque.connect-udp-tunnel')}|ConvertTo-Json -Compress;exit 3}
Push-Location $PSScriptRoot
try{
  if($PlanOnly){[ordered]@{schemaVersion='protocol-lab.endpoint-plan.v1';implementationId='masque-go-connect-udp';packageVersion='0.1.2';upstreamVersion='v0.4.0';scenarioId=$scenario;image=$Image;hostPort=$Port;containerPort=4443;protocol='masque-connect-udp-over-h3';roles=@('proxy','udp-target')}|ConvertTo-Json -Compress;return}
  if(-not $SkipBuild){& docker build --pull -f docker/Dockerfile -t $Image .;if($LASTEXITCODE-ne 0){throw 'Docker build failed.'}}
  $version=(& docker run --rm $Image --version).Trim();if($version-ne'masque-go-connect-udp 0.1.2 masque-go v0.4.0'){throw "Version proof mismatch: $version"}
  & docker run --rm -e "PLAB_PUBLIC_PORT=$Port" -p "${Port}:4443/udp" $Image
  if($LASTEXITCODE-ne 0){throw "Server failed with exit code $LASTEXITCODE."}
}finally{Pop-Location}
