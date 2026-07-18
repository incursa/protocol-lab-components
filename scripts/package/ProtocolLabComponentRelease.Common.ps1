Set-StrictMode -Version Latest

function Get-ProtocolLabComponentReleaseGraph {
    param(
        [Parameter(Mandatory)][string]$GraphPath
    )

    if (-not (Test-Path -LiteralPath $GraphPath -PathType Leaf)) {
        throw "Component release graph was not found: $GraphPath"
    }

    try {
        $graph = Get-Content -LiteralPath $GraphPath -Raw | ConvertFrom-Json -AsHashtable
    }
    catch {
        throw "Component release graph is not valid JSON: $GraphPath. $($_.Exception.Message)"
    }

    if ($graph.schemaVersion -ne 'protocol-lab.component-graph.v1') {
        throw "Unsupported component release graph schema '$($graph.schemaVersion)'."
    }

    return $graph
}

function ConvertTo-ProtocolLabRelativePath {
    param(
        [Parameter(Mandatory)][string]$Root,
        [Parameter(Mandatory)][string]$Path
    )

    $fullPath = [System.IO.Path]::GetFullPath((Join-Path $Root $Path))
    $rootPath = [System.IO.Path]::GetFullPath($Root)
    if (-not ($fullPath.StartsWith($rootPath + [System.IO.Path]::DirectorySeparatorChar) -or $fullPath -eq $rootPath)) {
        throw "Declared graph path escapes the repository root: $Path"
    }

    return [System.IO.Path]::GetRelativePath($rootPath, $fullPath).Replace('\', '/')
}

function Test-ProtocolLabPathMatchesDeclaration {
    param(
        [Parameter(Mandatory)][string]$ChangedPath,
        [Parameter(Mandatory)][string]$DeclaredPath
    )

    $normalizedChanged = $ChangedPath.Replace('\', '/').TrimStart('./')
    $normalizedDeclared = $DeclaredPath.Replace('\', '/').Trim('/').TrimStart('./')
    return $normalizedChanged -eq $normalizedDeclared -or $normalizedChanged.StartsWith("$normalizedDeclared/", [System.StringComparison]::OrdinalIgnoreCase)
}

function Get-ProtocolLabDeclaredFiles {
    param(
        [Parameter(Mandatory)][string]$Root,
        [Parameter(Mandatory)][AllowEmptyCollection()][object[]]$Paths,
        [switch]$PackagePayload
    )

    $files = @{}
    foreach ($path in @($Paths)) {
        $relative = ConvertTo-ProtocolLabRelativePath -Root $Root -Path ([string]$path)
        $absolute = Join-Path $Root $relative
        if (-not (Test-Path -LiteralPath $absolute)) {
            throw "Declared component closure path does not exist: $relative"
        }

        $candidates = if ((Get-Item -LiteralPath $absolute).PSIsContainer) {
            @(Get-ChildItem -LiteralPath $absolute -Recurse -File -Force)
        }
        else {
            @(Get-Item -LiteralPath $absolute -Force)
        }

        foreach ($file in $candidates) {
            $fileRelative = [System.IO.Path]::GetRelativePath($Root, $file.FullName).Replace('\', '/')
            if ($PackagePayload) {
                $parts = $fileRelative -split '/'
                if ($parts -contains 'artifacts' -or $parts -contains 'packages' -or $parts -contains 'bin' -or $parts -contains 'obj' -or
                    $parts -contains 'package.protocol-lab.json' -or $parts[-1] -eq 'README.md') {
                    continue
                }
            }

            $files[$fileRelative] = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        }
    }

    $records = @($files.GetEnumerator() | Sort-Object Key | ForEach-Object {
        [ordered]@{ path = $_.Key; sha256 = $_.Value }
    })
    Write-Output -NoEnumerate $records
}

function Get-ProtocolLabSha256Text {
    param([Parameter(Mandatory)][string]$Text)
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($Text)
    try {
        return ([System.Security.Cryptography.SHA256]::HashData($bytes) | ForEach-Object ToString x2) -join ''
    }
    finally {
        [System.Array]::Clear($bytes, 0, $bytes.Length)
    }
}

function ConvertTo-ProtocolLabCanonicalJson {
    param([Parameter(Mandatory)]$Value)
    return ($Value | ConvertTo-Json -Depth 32 -Compress)
}

function Get-ProtocolLabComponentClosure {
    param(
        [Parameter(Mandatory)][hashtable]$Graph,
        [Parameter(Mandatory)][string]$ComponentId,
        [Parameter(Mandatory)][string]$Root
    )

    $component = @($Graph.components | Where-Object { $_.id -eq $ComponentId })
    if ($component.Count -ne 1) {
        throw "Component '$ComponentId' is not uniquely declared in the component graph."
    }
    $component = $component[0]
    $sharedById = @{}
    foreach ($shared in @($Graph.sharedInputs)) { $sharedById[[string]$shared.id] = $shared }

    $payload = @(Get-ProtocolLabDeclaredFiles -Root $Root -Paths @($component.inputs.payload) -PackagePayload)
    $recipe = @(Get-ProtocolLabDeclaredFiles -Root $Root -Paths @($component.inputs.buildRecipe))
    $fixtures = @(Get-ProtocolLabDeclaredFiles -Root $Root -Paths @($component.inputs.fixtures))
    $templatePaths = @()
    foreach ($templateId in @($component.inputs.templates)) {
        $template = @($Graph.templates | Where-Object { $_.id -eq $templateId })
        if ($template.Count -ne 1) { throw "Component '$ComponentId' references unknown template '$templateId'." }
        $templatePaths += @($template[0].paths)
    }
    $templates = @(Get-ProtocolLabDeclaredFiles -Root $Root -Paths $templatePaths)
    $shared = @()
    foreach ($sharedId in @($component.inputs.shared)) {
        if (-not $sharedById.ContainsKey([string]$sharedId)) { throw "Component '$ComponentId' references unknown shared input '$sharedId'." }
        $shared += @(Get-ProtocolLabDeclaredFiles -Root $Root -Paths @($sharedById[[string]$sharedId].paths))
    }
    $shared = @($shared | Sort-Object path)

    $payloadDigest = Get-ProtocolLabSha256Text (ConvertTo-ProtocolLabCanonicalJson $payload)
    $recipeDigest = Get-ProtocolLabSha256Text (ConvertTo-ProtocolLabCanonicalJson $recipe)
    $closure = [ordered]@{
        schemaVersion = 'protocol-lab.component-closure.v1'
        component = [ordered]@{
            id = $component.id
            packageId = $component.packageId
            packageRoot = $component.packageRoot
            kind = $component.kind
        }
        payload = $payload
        shared = $shared
        buildRecipe = $recipe
        fixtures = $fixtures
        templates = $templates
        contracts = @($component.inputs.contracts | Sort-Object id, repository, version)
        toolchains = @($component.inputs.toolchains | Sort-Object id, version)
        dependencies = @($component.dependsOn | Sort-Object componentId, kind)
    }
    $closureJson = ConvertTo-ProtocolLabCanonicalJson $closure
    return [ordered]@{
        componentId = $component.id
        packageId = $component.packageId
        componentTreeDigest = $payloadDigest
        buildRecipeDigest = $recipeDigest
        componentClosureDigest = Get-ProtocolLabSha256Text $closureJson
        closure = $closure
    }
}
