# Go WebTransport executor

Native `webtransport-go` v0.11.1 client executor for
`webtransport.session-bidi-echo`. It dials the worker endpoint while retaining
the contract authority `webtransport.plab.test`, proves HTTP/3 WebTransport
session establishment, opens one client-initiated bidirectional stream, sends
the deterministic 65,536-byte payload, verifies the exact echo hash, and emits
fail-closed normalized artifacts.
