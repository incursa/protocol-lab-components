# HTTP/2 Performance Smoke and Diagnostic Scenario Pack

This package snapshots the already-committed public `http2.core.plaintext`, `http2.core.json`, `http2-smoke`, and `http2-diagnostic` contracts from `protocol-lab` commit `a4dcd74e5c8907907ccc58808da92d2b920b2fbc`.

It deliberately excludes `http2.streaming.response` because the target/executor vertical does not support it. It also excludes draft contracts, the comparison profile, executor code, implementation code, and evidence. `authority-lock.json` records and validates the exact public-source SHA-256 values.
