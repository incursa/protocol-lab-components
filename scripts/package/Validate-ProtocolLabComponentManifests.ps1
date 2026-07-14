[CmdletBinding()]
param(
    [string]$Root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
)

$ErrorActionPreference = 'Stop'

function Test-IsIgnoredPath {
    param([Parameter(Mandatory)][string]$Path)

    return $Path -match '[\\/](artifacts|packages|bin|obj)[\\/]'
}

function Test-RelativePackagePath {
    param([AllowNull()][string]$Path)

    if ([string]::IsNullOrWhiteSpace($Path)) {
        return $false
    }

    if ([System.IO.Path]::IsPathRooted($Path)) {
        return $false
    }

    if ($Path -match '^[A-Za-z][A-Za-z0-9+.-]*:') {
        return $false
    }

    return -not ($Path -split '[\\/]' | Where-Object { $_ -eq '..' })
}

function Test-Token {
    param([AllowNull()][string]$Value)

    return -not [string]::IsNullOrWhiteSpace($Value) -and $Value -match '^[A-Za-z0-9][A-Za-z0-9_.:-]*$'
}

function Test-StringArray {
    param([AllowNull()]$Value)

    if ($null -eq $Value -or $Value -is [string]) {
        return $false
    }

    $items = @($Value)
    return $items.Count -gt 0 -and -not ($items | Where-Object { [string]::IsNullOrWhiteSpace([string]$_) })
}

function Test-AllowedProperties {
    param(
        [Parameter(Mandatory)][object]$Value,
        [Parameter(Mandatory)][string[]]$Allowed,
        [Parameter(Mandatory)][string]$Context
    )

    foreach ($propertyName in $Value.PSObject.Properties.Name) {
        if ($propertyName -notin $Allowed) {
            $errors.Add("${Context}: unsupported public property '$propertyName'.")
        }
    }
}

$publicManifestFiles = Get-ChildItem -LiteralPath $Root -Recurse -Filter 'protocol-lab-package.json' |
    Where-Object { -not (Test-IsIgnoredPath -Path $_.FullName) } |
    Sort-Object FullName

$internalManifestFiles = Get-ChildItem -LiteralPath $Root -Recurse -Filter 'protocol-lab.internal.json' |
    Where-Object { -not (Test-IsIgnoredPath -Path $_.FullName) } |
    Sort-Object FullName

if (-not $publicManifestFiles) {
    throw "No protocol-lab-package.json files found under $Root."
}

$errors = New-Object System.Collections.Generic.List[string]
$packageIds = @{}
$componentIds = @{}
$publicManifestDirectories = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
$internalManifestDirectories = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)

foreach ($file in $internalManifestFiles) {
    [void]$internalManifestDirectories.Add($file.DirectoryName)
}

$publicAllowedProperties = @(
    'schemaVersion',
    'packageId',
    'packageVersion',
    'kind',
    'displayName',
    'entryManifests',
    'providedImplementations',
    'providedTestExecutors',
    'providedScenarios',
    'providedSuites',
    'providedLoadProfiles',
    'providedSpecificationCoverage',
    'dependencies'
)

$legacyExecutionProperties = @(
    'packageKind',
    'version',
    'component',
    'entrypoints',
    'requirements',
    'provenance',
    'capabilities'
)

foreach ($file in $publicManifestFiles) {
    [void]$publicManifestDirectories.Add($file.DirectoryName)

    try {
        $manifest = Get-Content -LiteralPath $file.FullName -Raw | ConvertFrom-Json
    }
    catch {
        $errors.Add("$($file.FullName): invalid JSON: $($_.Exception.Message)")
        continue
    }

    foreach ($propertyName in $manifest.PSObject.Properties.Name) {
        if ($propertyName -notin $publicAllowedProperties) {
            $errors.Add("$($file.FullName): property '$propertyName' is not part of the public protocol-lab-package-v2 manifest.")
        }
    }

    foreach ($propertyName in $legacyExecutionProperties) {
        if ($manifest.PSObject.Properties.Name.Contains($propertyName)) {
            $errors.Add("$($file.FullName): execution/legacy property '$propertyName' belongs outside protocol-lab-package.json.")
        }
    }

    foreach ($required in @('schemaVersion', 'packageId', 'packageVersion', 'kind', 'entryManifests')) {
        if (-not $manifest.PSObject.Properties.Name.Contains($required)) {
            $errors.Add("$($file.FullName): missing required property '$required'.")
        }
    }

    if ($manifest.schemaVersion -and $manifest.schemaVersion -ne 'protocol-lab-package-v2') {
        $errors.Add("$($file.FullName): schemaVersion must be 'protocol-lab-package-v2'.")
    }

    if ($manifest.packageId -and -not (Test-Token -Value ([string]$manifest.packageId))) {
        $errors.Add("$($file.FullName): packageId '$($manifest.packageId)' is not a valid package token.")
    }

    if ($manifest.packageVersion -and [string]$manifest.packageVersion -notmatch '^\d+\.\d+\.\d+([+-][0-9A-Za-z.-]+)?$') {
        $errors.Add("$($file.FullName): packageVersion '$($manifest.packageVersion)' is not a semantic version.")
    }

    if ($manifest.packageId) {
        $packageId = [string]$manifest.packageId
        if ($packageIds.ContainsKey($packageId)) {
            $errors.Add("$($file.FullName): duplicate packageId '$packageId' also used by $($packageIds[$packageId]).")
        }
        else {
            $packageIds[$packageId] = $file.FullName
        }
    }

    if ($manifest.kind -and $manifest.kind -notin @('implementation', 'test-executor', 'scenario-pack', 'toolchain')) {
        $errors.Add("$($file.FullName): kind must be 'implementation', 'test-executor', 'scenario-pack', or 'toolchain'.")
    }

    if ($null -eq $manifest.entryManifests -or $manifest.entryManifests -is [string]) {
        $errors.Add("$($file.FullName): entryManifests must be an array.")
    }
    else {
        $entryManifests = @($manifest.entryManifests)
        if ($manifest.kind -ne 'toolchain' -and $entryManifests.Count -eq 0) {
            $errors.Add("$($file.FullName): entryManifests must contain at least one package-relative manifest path.")
        }

        foreach ($entryManifest in $entryManifests) {
            $entryPath = [string]$entryManifest
            if (-not (Test-RelativePackagePath -Path $entryPath)) {
                $errors.Add("$($file.FullName): entry manifest path '$entryPath' must be package-relative and must not traverse upward.")
                continue
            }

            if ($manifest.kind -eq 'implementation' -and $entryPath -notmatch '^implementations/') {
                $errors.Add("$($file.FullName): implementation entry manifest '$entryPath' must be under implementations/.")
            }
            elseif ($manifest.kind -eq 'test-executor' -and $entryPath -notmatch '^test-executors/') {
                $errors.Add("$($file.FullName): test-executor entry manifest '$entryPath' must be under test-executors/.")
            }

            $entryFullPath = Join-Path $file.DirectoryName $entryPath
            if (-not (Test-Path -LiteralPath $entryFullPath -PathType Leaf)) {
                $errors.Add("$($file.FullName): entry manifest '$entryPath' does not exist beside the package manifest.")
            }
        }
    }

    if ($manifest.kind -eq 'implementation') {
        if ($null -eq $manifest.providedImplementations -or $manifest.providedImplementations -is [string] -or @($manifest.providedImplementations).Count -eq 0) {
            $errors.Add("$($file.FullName): implementation packages must declare at least one providedImplementations entry.")
        }
        else {
            foreach ($provided in @($manifest.providedImplementations)) {
                $providedId = [string]$provided.implementationId
                if (-not (Test-Token -Value $providedId)) {
                    $errors.Add("$($file.FullName): providedImplementations entry is missing a valid implementationId.")
                }
                elseif ($componentIds.ContainsKey($providedId)) {
                    $errors.Add("$($file.FullName): duplicate provided component id '$providedId' also used by $($componentIds[$providedId]).")
                }
                else {
                    $componentIds[$providedId] = $file.FullName
                }

                if (-not (Test-StringArray -Value $provided.protocols)) {
                    $errors.Add("$($file.FullName): provided implementation '$providedId' must declare one or more protocols.")
                }

                if (-not (Test-StringArray -Value $provided.scenarios)) {
                    $errors.Add("$($file.FullName): provided implementation '$providedId' must declare one or more scenario IDs.")
                }
            }
        }

        if ($manifest.PSObject.Properties.Name.Contains('providedTestExecutors')) {
            $errors.Add("$($file.FullName): implementation packages must not declare providedTestExecutors.")
        }
    }
    elseif ($manifest.kind -eq 'test-executor') {
        if ($null -eq $manifest.providedTestExecutors -or $manifest.providedTestExecutors -is [string] -or @($manifest.providedTestExecutors).Count -eq 0) {
            $errors.Add("$($file.FullName): test-executor packages must declare at least one providedTestExecutors entry.")
        }
        else {
            foreach ($provided in @($manifest.providedTestExecutors)) {
                $providedId = [string]$provided.testExecutorId
                if (-not (Test-Token -Value $providedId)) {
                    $errors.Add("$($file.FullName): providedTestExecutors entry is missing a valid testExecutorId.")
                }
                elseif ($componentIds.ContainsKey($providedId)) {
                    $errors.Add("$($file.FullName): duplicate provided component id '$providedId' also used by $($componentIds[$providedId]).")
                }
                else {
                    $componentIds[$providedId] = $file.FullName
                }

                if (-not (Test-StringArray -Value $provided.protocols)) {
                    $errors.Add("$($file.FullName): provided test executor '$providedId' must declare one or more protocols.")
                }

                if (-not (Test-StringArray -Value $provided.scenarios)) {
                    $errors.Add("$($file.FullName): provided test executor '$providedId' must declare one or more scenario IDs.")
                }

                if (-not (Test-StringArray -Value $provided.tests)) {
                    $errors.Add("$($file.FullName): provided test executor '$providedId' must declare one or more test IDs.")
                }

                if ($provided.PSObject.Properties.Name.Contains('checkIds') -and -not (Test-StringArray -Value $provided.checkIds)) {
                    $errors.Add("$($file.FullName): provided test executor '$providedId' checkIds must contain one or more exact check IDs when declared.")
                }
            }
        }

        if ($manifest.PSObject.Properties.Name.Contains('providedImplementations')) {
            $errors.Add("$($file.FullName): test-executor packages must not declare providedImplementations.")
        }
    }
    elseif ($manifest.kind -eq 'scenario-pack') {
        if (($null -eq $manifest.providedScenarios -or @($manifest.providedScenarios).Count -eq 0) -and
            ($null -eq $manifest.providedSuites -or @($manifest.providedSuites).Count -eq 0)) {
            $errors.Add("$($file.FullName): scenario-pack packages must declare providedScenarios or providedSuites.")
        }

        foreach ($provided in @($manifest.providedScenarios)) {
            $providedId = [string]$provided.scenarioId
            Test-AllowedProperties `
                -Value $provided `
                -Allowed @('scenarioId', 'displayName', 'protocols') `
                -Context "$($file.FullName): providedScenarios entry '$providedId'"

            if (-not (Test-Token -Value $providedId)) {
                $errors.Add("$($file.FullName): providedScenarios entry is missing a valid scenarioId.")
            }

            if (-not (Test-StringArray -Value $provided.protocols)) {
                $errors.Add("$($file.FullName): provided scenario '$providedId' must declare one or more protocols.")
            }
        }

        $providedSuiteEntries = if ($manifest.PSObject.Properties.Name.Contains('providedSuites')) { @($manifest.providedSuites) } else { @() }
        foreach ($provided in $providedSuiteEntries) {
            $providedId = [string]$provided.suiteId
            Test-AllowedProperties `
                -Value $provided `
                -Allowed @('suiteId', 'displayName', 'protocols', 'purpose', 'resultKind', 'testExecutors') `
                -Context "$($file.FullName): providedSuites entry '$providedId'"

            if (-not (Test-Token -Value $providedId)) {
                $errors.Add("$($file.FullName): providedSuites entry is missing a valid suiteId.")
            }

            if ($provided.PSObject.Properties.Name.Contains('protocols') -and -not (Test-StringArray -Value $provided.protocols)) {
                $errors.Add("$($file.FullName): provided suite '$providedId' protocols must contain one or more protocol IDs when declared.")
            }

            if ($provided.PSObject.Properties.Name.Contains('testExecutors') -and -not (Test-StringArray -Value $provided.testExecutors)) {
                $errors.Add("$($file.FullName): provided suite '$providedId' testExecutors must contain one or more exact executor IDs when declared.")
            }
        }

        $providedLoadProfileEntries = if ($manifest.PSObject.Properties.Name.Contains('providedLoadProfiles')) { @($manifest.providedLoadProfiles) } else { @() }
        foreach ($provided in $providedLoadProfileEntries) {
            $providedId = [string]$provided.loadProfileId
            Test-AllowedProperties `
                -Value $provided `
                -Allowed @('loadProfileId', 'displayName', 'protocols') `
                -Context "$($file.FullName): providedLoadProfiles entry '$providedId'"

            if (-not (Test-Token -Value $providedId)) {
                $errors.Add("$($file.FullName): providedLoadProfiles entry is missing a valid loadProfileId.")
            }

            if (-not (Test-StringArray -Value $provided.protocols)) {
                $errors.Add("$($file.FullName): provided load profile '$providedId' must declare one or more protocols.")
            }
        }

        $specificationCoverageEntries = if ($manifest.PSObject.Properties.Name.Contains('providedSpecificationCoverage')) {
            @($manifest.providedSpecificationCoverage)
        }
        else {
            @()
        }
        foreach ($coverage in $specificationCoverageEntries) {
            Test-AllowedProperties `
                -Value $coverage `
                -Allowed @('catalogId', 'catalogVersion', 'catalogPath', 'mappingPaths', 'profilePaths') `
                -Context "$($file.FullName): providedSpecificationCoverage entry"

            if (-not (Test-Token -Value ([string]$coverage.catalogId))) {
                $errors.Add("$($file.FullName): providedSpecificationCoverage entry is missing a valid catalogId.")
            }
            if ([string]::IsNullOrWhiteSpace([string]$coverage.catalogVersion)) {
                $errors.Add("$($file.FullName): providedSpecificationCoverage entry is missing catalogVersion.")
            }

            $coveragePaths = @([string]$coverage.catalogPath) + @($coverage.mappingPaths) + @($coverage.profilePaths)
            if (-not (Test-StringArray -Value $coverage.mappingPaths)) {
                $errors.Add("$($file.FullName): specification coverage '$($coverage.catalogId)' must declare one or more mappingPaths.")
            }
            foreach ($coveragePathValue in $coveragePaths) {
                $coveragePath = [string]$coveragePathValue
                if (-not (Test-RelativePackagePath -Path $coveragePath) -or $coveragePath -notmatch '^specifications/') {
                    $errors.Add("$($file.FullName): specification coverage path '$coveragePath' must be package-relative and under specifications/.")
                    continue
                }
                if ($coveragePath -notin @($manifest.entryManifests)) {
                    $errors.Add("$($file.FullName): specification coverage path '$coveragePath' must also appear in entryManifests.")
                }
                if (-not (Test-Path -LiteralPath (Join-Path $file.DirectoryName $coveragePath) -PathType Leaf)) {
                    $errors.Add("$($file.FullName): specification coverage path '$coveragePath' does not exist beside the package manifest.")
                }
            }
        }
    }

    if ($manifest.PSObject.Properties.Name.Contains('dependencies')) {
        foreach ($dependencyPropertyName in $manifest.dependencies.PSObject.Properties.Name) {
            if ($dependencyPropertyName -ne 'requiredCapabilities') {
                $errors.Add("$($file.FullName): public dependencies property '$dependencyPropertyName' belongs in protocol-lab.internal.json.")
            }
        }
    }
}

foreach ($file in $publicManifestFiles) {
    if (-not $internalManifestDirectories.Contains($file.DirectoryName)) {
        $errors.Add("$($file.FullName): protocol-lab-package.json must be paired with protocol-lab.internal.json in the same component directory.")
    }
}

$internalPublicFields = @(
    'packageId',
    'packageVersion',
    'kind',
    'displayName',
    'entryManifests',
    'providedImplementations',
    'providedTestExecutors',
    'providedScenarios',
    'providedSuites',
    'providedLoadProfiles',
    'providedSpecificationCoverage'
)

foreach ($file in $internalManifestFiles) {
    if (-not $publicManifestDirectories.Contains($file.DirectoryName)) {
        $errors.Add("$($file.FullName): protocol-lab.internal.json must be paired with protocol-lab-package.json in the same component directory.")
    }

    try {
        $manifest = Get-Content -LiteralPath $file.FullName -Raw | ConvertFrom-Json
    }
    catch {
        $errors.Add("$($file.FullName): invalid JSON: $($_.Exception.Message)")
        continue
    }

    foreach ($propertyName in $internalPublicFields) {
        if ($manifest.PSObject.Properties.Name.Contains($propertyName)) {
            $errors.Add("$($file.FullName): public package property '$propertyName' belongs in protocol-lab-package.json.")
        }
    }

    foreach ($required in @('schemaVersion', 'environments', 'dependencies')) {
        if (-not $manifest.PSObject.Properties.Name.Contains($required)) {
            $errors.Add("$($file.FullName): missing required property '$required'.")
        }
    }

    if ($manifest.schemaVersion -and $manifest.schemaVersion -ne 'protocol-lab-internal-execution-v1') {
        $errors.Add("$($file.FullName): schemaVersion must be 'protocol-lab-internal-execution-v1'.")
    }

    if ($null -eq $manifest.environments -or $manifest.environments -is [string] -or @($manifest.environments).Count -eq 0) {
        $errors.Add("$($file.FullName): environments must contain at least one execution environment.")
    }
    else {
        foreach ($environment in @($manifest.environments)) {
            $environmentLabel = "$($environment.os)/$($environment.arch)"
            if ([string]::IsNullOrWhiteSpace([string]$environment.os)) {
                $errors.Add("$($file.FullName): environment is missing os.")
            }

            if ([string]::IsNullOrWhiteSpace([string]$environment.arch)) {
                $errors.Add("$($file.FullName): environment '$environmentLabel' is missing arch.")
            }

            if ($null -eq $environment.entrypoint) {
                $errors.Add("$($file.FullName): environment '$environmentLabel' is missing entrypoint.")
                continue
            }

            if ([string]::IsNullOrWhiteSpace([string]$environment.entrypoint.kind)) {
                $errors.Add("$($file.FullName): environment '$environmentLabel' entrypoint is missing kind.")
            }
            elseif ($environment.entrypoint.kind -notin @('process', 'pwsh', 'bash')) {
                $errors.Add("$($file.FullName): environment '$environmentLabel' entrypoint kind '$($environment.entrypoint.kind)' is not supported.")
            }

            $entrypointPath = [string]$environment.entrypoint.path
            if (-not (Test-RelativePackagePath -Path $entrypointPath)) {
                $errors.Add("$($file.FullName): environment '$environmentLabel' entrypoint path '$entrypointPath' must be package-relative and must not traverse upward.")
            }

            if ($null -eq $environment.entrypoint.arguments -or $environment.entrypoint.arguments -is [string]) {
                $errors.Add("$($file.FullName): environment '$environmentLabel' entrypoint arguments must be an array.")
            }

            if ([string]::IsNullOrWhiteSpace([string]$environment.entrypoint.workingDirectory)) {
                $errors.Add("$($file.FullName): environment '$environmentLabel' entrypoint is missing workingDirectory.")
            }
            elseif (-not (Test-RelativePackagePath -Path ([string]$environment.entrypoint.workingDirectory))) {
                $errors.Add("$($file.FullName): environment '$environmentLabel' workingDirectory '$($environment.entrypoint.workingDirectory)' must be package-relative and must not traverse upward.")
            }
        }
    }

    if ($null -eq $manifest.dependencies) {
        $errors.Add("$($file.FullName): dependencies must describe execution requirements.")
    }
    else {
        foreach ($requiredBoolean in @('requiresDotNet', 'requiresDocker', 'requiresPwsh', 'requiresBash', 'requiresGo')) {
            if (-not $manifest.dependencies.PSObject.Properties.Name.Contains($requiredBoolean)) {
                $errors.Add("$($file.FullName): dependencies is missing '$requiredBoolean'.")
            }
            elseif ($manifest.dependencies.$requiredBoolean -isnot [bool]) {
                $errors.Add("$($file.FullName): dependencies.$requiredBoolean must be a boolean.")
            }
        }

        if (-not $manifest.dependencies.PSObject.Properties.Name.Contains('requiredCapabilities')) {
            $errors.Add("$($file.FullName): dependencies is missing 'requiredCapabilities'.")
        }
        elseif ($manifest.dependencies.requiredCapabilities -is [string]) {
            $errors.Add("$($file.FullName): dependencies.requiredCapabilities must be an array.")
        }
    }
}

if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Error $_ }
    throw "Protocol Lab component manifest validation failed with $($errors.Count) error(s)."
}

Write-Host "Validated $($publicManifestFiles.Count) public Protocol Lab package manifest(s) and $($internalManifestFiles.Count) internal execution manifest(s)."
