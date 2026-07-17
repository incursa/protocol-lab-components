[CmdletBinding()]
param(
    [switch]$PlanOnly,
    [switch]$ProofOnly
)

$ErrorActionPreference = "Stop"

$image = if ($env:PLAB_IMAGE) { $env:PLAB_IMAGE } else { "incursa-protocol-lab-h2o-http3:0.1.0" }
$port = if ($env:PLAB_HTTP_PORT) { [int]$env:PLAB_HTTP_PORT } else { 5448 }
$skipBuild = $env:PLAB_SKIP_BUILD -eq "1"
$artifactRoot = if ($env:PLAB_ARTIFACT_ROOT) { $env:PLAB_ARTIFACT_ROOT } else { "artifacts/h2o-http3" }

New-Item -ItemType Directory -Force -Path $artifactRoot | Out-Null
$commands = @()
if (-not $skipBuild) {
    $commands += "docker build --pull -f docker/H2O.Dockerfile -t $image docker"
}
$commands += "docker run --rm --entrypoint h2o $image -v"
$commands += "docker run --rm -p ${port}:8443/tcp -p ${port}:8443/udp $image"
$commands | Set-Content -NoNewline -Path (Join-Path $artifactRoot "command.txt")

if ($PlanOnly -or $env:PLAB_PLAN_ONLY -eq "1") {
    @{ status = "planned"; image = $image; port = $port } | ConvertTo-Json -Compress | Set-Content -NoNewline -Path (Join-Path $artifactRoot "result.json")
    exit 0
}

if (-not $skipBuild) {
    docker build --pull -f docker/H2O.Dockerfile -t $image docker
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

$versionOutput = & docker run --rm --entrypoint h2o $image -v 2>&1
$versionOutput | Set-Content -Path (Join-Path $artifactRoot "h2o-version.txt")
if ($LASTEXITCODE -ne 0 -or ($versionOutput -join "`n") -notmatch "h2o") {
    throw "Selected image '$image' did not identify the H2O server."
}

if ($ProofOnly -or $env:PLAB_PROOF_ONLY -eq "1") {
    @{ status = "proved"; image = $image; versionPath = (Join-Path $artifactRoot "h2o-version.txt") } | ConvertTo-Json -Compress | Set-Content -NoNewline -Path (Join-Path $artifactRoot "result.json")
    exit 0
}

docker run --rm -p "${port}:8443/tcp" -p "${port}:8443/udp" $image
exit $LASTEXITCODE
