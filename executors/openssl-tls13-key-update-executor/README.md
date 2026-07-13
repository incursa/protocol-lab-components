# OpenSSL TLS 1.3 KeyUpdate executor

This package supports exactly `tls.key-update.diagnostic` under `tls-diagnostic`.
It uses `SSL_key_update(SSL_KEY_UPDATE_NOT_REQUESTED)` and OpenSSL's message
callback to prove one genuine client-initiated KeyUpdate, target observation,
traffic-key generation transition, and exact deterministic post-update traffic.
It never enables key logging or publishes traffic-secret material.

Every other committed TLS identity returns explicit `unsupported`; unknown IDs
remain separate configuration errors.
