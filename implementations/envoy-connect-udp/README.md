# Envoy CONNECT-UDP

This package pins Envoy v1.38.3 at tag commit
`0ebfcfe5b0484b89ca85b761da9e05ce75dbda8d` and Linux image digest
`sha256:5f7c43e1147412fdb3af578c651c67478a3df818eae89d2261e707e06c209cdb`.
Envoy terminates RFC 9298 CONNECT-UDP over HTTP/3 and routes datagrams to the
package-backed UDP echo listener at `127.0.0.1:4433`.

Envoy documents CONNECT-UDP as alpha. The proxy and target roles are
separately configured and logged but co-located in the same diagnostic
container, so the package does not claim a physically separated or rankable
topology.
