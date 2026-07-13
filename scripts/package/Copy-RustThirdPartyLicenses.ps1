[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$ManifestPath,
    [Parameter(Mandatory)][string]$Destination,
    [string]$Toolchain='stable-x86_64-pc-windows-gnu'
)

$ErrorActionPreference='Stop'
$ManifestPath=[IO.Path]::GetFullPath($ManifestPath)
$Destination=[IO.Path]::GetFullPath($Destination)
if(Test-Path $Destination){Remove-Item -LiteralPath $Destination -Recurse -Force}
New-Item -ItemType Directory -Force $Destination|Out-Null

$metadataText=& cargo "+$Toolchain" metadata --locked --format-version 1 --manifest-path $ManifestPath
if($LASTEXITCODE-ne 0){throw 'cargo metadata failed while collecting third-party license material.'}
$metadata=$metadataText|ConvertFrom-Json
$entries=@()
foreach($package in @($metadata.packages|Sort-Object name,version)){
    $packageRoot=Split-Path -Parent ([string]$package.manifest_path)
    $licenseFiles=@(
        Get-ChildItem -LiteralPath $packageRoot -File -ErrorAction SilentlyContinue |
            Where-Object {$_.Name-match '^(LICENSE|LICENCE|COPYING|NOTICE|COPYRIGHT)'} |
            Sort-Object Name
    )
    $relativeFiles=@()
    if($licenseFiles.Count-gt 0){
        $packageDestination=Join-Path $Destination ("{0}-{1}"-f $package.name,$package.version)
        New-Item -ItemType Directory -Force $packageDestination|Out-Null
        foreach($licenseFile in $licenseFiles){
            Copy-Item -LiteralPath $licenseFile.FullName -Destination (Join-Path $packageDestination $licenseFile.Name)
            $relativeFiles+=("{0}/{1}"-f (Split-Path -Leaf $packageDestination),$licenseFile.Name)
        }
    }
    $entries+=[ordered]@{
        name=[string]$package.name
        version=[string]$package.version
        license=[string]$package.license
        source=[string]$package.source
        licenseFiles=$relativeFiles
    }
}
$index=[ordered]@{schemaVersion='protocol-lab.third-party-licenses.v1';packages=$entries}
$index|ConvertTo-Json -Depth 8|Set-Content (Join-Path $Destination 'index.json') -Encoding utf8NoBOM

$critical=@('rustls','rustls-rustcrypto','rustls-pemfile')
foreach($name in $critical){
    $entry=@($entries|Where-Object {$_.name-eq$name})
    if($entry.Count-ne 1-or$entry[0].licenseFiles.Count-eq 0){throw "Required license material was not found for $name."}
}
