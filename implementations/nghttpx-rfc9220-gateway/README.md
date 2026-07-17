# nghttpx RFC 9220 WebSocket Gateway

This component packages nghttpx 1.69.0 as an HTTP/3 reverse-proxy target for the six canonical ProtocolLab RFC 9220 scenarios. The tested implementation is the nghttpx gateway. A small package-owned HTTP/1.1 WebSocket backend provides deterministic echo behavior and proof events; it is fixture infrastructure, not a separately cataloged implementation.

The build follows the upstream nghttp2 HTTP/3 recipe with immutable source commits for nghttp2, ngtcp2, nghttp3, and AWS-LC. nghttpx documents its HTTP/3 support as experimental, so retained results remain diagnostic and non-rankable until the lab topology and repetition floors are satisfied.

## Local build

```powershell
docker build -t incursa-protocol-lab-nghttpx-rfc9220-gateway:0.1.0 .\implementations\nghttpx-rfc9220-gateway
```

The container listens on UDP 4433 with the package-local `websocket.plab.test` certificate and exposes `/websocket-proof` through RFC 9220 Extended CONNECT.
