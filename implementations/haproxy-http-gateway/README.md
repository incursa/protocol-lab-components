# HAProxy HTTP gateway package

This package exposes three separately selectable HAProxy 3.2 gateway identities:
HTTP/1.1, cleartext HTTP/2 prior knowledge, and HTTP/3. Each listener reverse
proxies to a loopback BusyBox HTTP fixture in the same container. HAProxy is
therefore measured in its gateway role and must not be compared as an origin
server implementation.

The HAProxy and BusyBox images are digest pinned. The HTTP/3 listener uses
HAProxy's QUIC-enabled official 3.2.21 image and a repository-owned test-only
certificate. The package intentionally advertises only canonical rows exercised
by the corresponding ProtocolLab executors.
