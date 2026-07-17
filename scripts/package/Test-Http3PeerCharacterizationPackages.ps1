[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
)

$ErrorActionPreference = 'Stop'

foreach ($component in @('quiche-http3', 'ngtcp2-http3')) {
    $componentRoot = Join-Path $Root "implementations/$component"
    $package = Get-Content (Join-Path $componentRoot 'protocol-lab-package.json') -Raw | ConvertFrom-Json
    $internal = Get-Content (Join-Path $componentRoot 'protocol-lab.internal.json') -Raw | ConvertFrom-Json
    $manifestText = Get-Content (Join-Path $componentRoot "implementations/$component.yaml") -Raw

    if ($package.providedImplementations.scenarios -notcontains 'http3.external.peer-characterization') {
        throw "$component does not provide the HTTP/3 peer-characterization scenario."
    }

    if ($manifestText -notmatch '(?m)^\s*- http3\.external\s*$') {
        throw "$component does not declare the http3.external workload family."
    }

    if ($manifestText -notmatch '(?m)^\s*- h3ExternalPeer\s*$') {
        throw "$component does not declare the h3ExternalPeer capability."
    }

    if ($manifestText -notmatch '/tmp/www/index\.html') {
        throw "$component does not materialize the scenario's root-path fixture."
    }

    $hostRuntimeGates = @($internal.dependencies.requiredCapabilities | Where-Object name -notin @('docker'))
    if ($hostRuntimeGates.Count -ne 0) {
        throw "$component retains a host runtime capability gate even though its peer runtime is Docker-packaged."
    }
}

Write-Host 'Validated Docker-backed HTTP/3 peer-characterization package contracts.'
