# Go WebTransport executor

Native `webtransport-go` v0.11.1 client executor for
`webtransport.session-bidi-echo` and `webtransport.session-datagram-echo`. It
dials the worker endpoint while retaining the contract authority
`webtransport.plab.test`, proves HTTP/3 WebTransport session establishment,
then executes only the selected data path: either one deterministic 65,536-byte
bidirectional-stream echo or 32 deterministic 256-byte datagram echoes. It
verifies exact bytes and the selected payload hash and emits fail-closed,
scenario-specific normalized artifacts.
