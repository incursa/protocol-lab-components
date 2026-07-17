# masque-go CONNECT-UDP

This package pins `github.com/quic-go/masque-go` v0.4.0 at commit
`03179c5407a332f0509fa8e165244e6410ac165a`. It terminates RFC 9298
CONNECT-UDP over HTTP/3 and forwards only the declared
`masque-echo.plab.test:4433` target to a separately logged exact UDP echo
listener in the same diagnostic container.

The co-located roles make the wire path deterministic and package-backed, but
they do not satisfy a physically separated ranking topology. Ordinary HTTP
proxying, CONNECT-IP, arbitrary external targets, and ranking claims are
explicitly unsupported.
