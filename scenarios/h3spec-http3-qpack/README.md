# h3spec HTTP/3 and QPACK Scenario Pack

This component packages the focused scenario and suite declarations used to run the `h3spec-http3-qpack` test executor through the live ProtocolLab controller.

The pack is declarative. It does not provide a target implementation or the h3spec executable. The suite is intentionally bound to `h3spec-http3-qpack` so package-backed scheduling does not fall back to `managed-httpclient-h3-load`.

## Package

- Package ID: `org.protocol-lab.components.scenario.h3spec-http3-qpack`
- Package version: `0.1.2`
- Suite ID: `h3spec-http3-qpack-focused`
- Scenarios: `http3.core.status`, `http3.headers.response-headers-50x32`, `http3.protocol.qpack-repeated-headers`
- Test executor: `h3spec-http3-qpack`

## Build

```powershell
pwsh ./scripts/package/Build-H3SpecHttp3QpackScenarioPackage.ps1
```
