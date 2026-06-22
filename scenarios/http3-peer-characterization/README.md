# HTTP/3 Peer Characterization Scenario Pack

This component packages the `http3.external.peer-characterization` diagnostic scenario and `http3-peer-characterization` suite for package-backed external HTTP/3 peer evidence.

The scenario makes quiche, ngtcp2, and similar peer wrappers visible in reports without claiming support for the official `http3.payload.*` benchmark scenarios.

## Build

Validate manifests:

```powershell
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
```

Build the scenario package:

```powershell
pwsh ./scripts/package/Build-Http3PeerCharacterizationScenarioPackage.ps1
```

## Packaged Scenarios

- `http3.external.peer-characterization`

## Packaged Suites

- `http3-peer-characterization`
