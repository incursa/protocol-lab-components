# Unbound DoT resolver feasibility decision

Working identity: `org.protocol-lab.components.implementation.unbound-dot-resolver`.

The resolver-specific public contract and executor are now present, but the native Unbound 1.22.0 DoT service does not negotiate the required `dot` ALPN token. A local digest-pinned image smoke reached the authenticated TLS service and failed closed on the exact ALPN check. A TLS-terminating adapter would substitute the protocol implementation under test, so it is not an acceptable workaround.

Do not publish or admit an Unbound DoT package unless upstream Unbound gains native `dot` ALPN support or the public contract changes for standards-backed reasons. This decision does not satisfy the resolver DoT live-diversity floor.
