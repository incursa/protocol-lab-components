# Go MASQUE CONNECT-UDP executor

This executor pins `github.com/quic-go/masque-go` v0.4.0 and opens an exact
RFC 9298 CONNECT-UDP tunnel over HTTP/3. Every operation sends 32 deterministic
256-byte datagrams through the selected proxy to
`masque-echo.plab.test:4433`, validates every ordered echo, and retains the
aggregate SHA-256, protocol proof, role headers, latency samples, repetition
summaries, and raw result.

The comparison profile runs three clean 15-second measurement windows. The
numbers are real comparable observations, but they remain explicitly
non-rankable until the proxy, UDP target, and load roles are physically
separated under controlled topology policy.
