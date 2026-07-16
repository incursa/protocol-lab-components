# BIND 9 DoT recursive resolver

Package version `0.1.2` runs BIND 9.20.24 as the selected recursive resolver for `dns.dot.resolver.interoperability.query.a`.

A second BIND process provides the isolated `plab.test.` fixture authority on loopback only. The resolver forwards only that zone, and an HTTP control shim executes authenticated `rndc flush` before every executor operation. The fixture process is support infrastructure, not a measured resolver implementation.
