# HTTP/2 Performance Smoke and Diagnostic Scenario Pack

Package version `0.2.1` snapshots the already-committed public `http2.core.plaintext`, `http2.core.json`, `http2-performance-smoke`, `http2-smoke`, and `http2-diagnostic` contracts from `protocol-lab` commit `f8117f3967c35f91baa9a19277d52f9e2c4a0c85`.

The `http2-performance-smoke` suite is a non-publishable implementation bring-up selector for the existing plaintext and JSON scenarios with the existing `http2-smoke` profile. The package deliberately excludes `http2.streaming.response` because the target/executor vertical does not support it. It also excludes draft contracts, the comparison profile, executor code, implementation code, and evidence. `authority-lock.json` records and validates the exact public-source SHA-256 values.
