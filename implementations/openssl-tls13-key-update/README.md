# OpenSSL TLS 1.3 KeyUpdate target

This independent library-backed target implements only `tls.key-update.diagnostic`.
It uses OpenSSL's public TLS 1.3 KeyUpdate handling and message callback to prove
that it received exactly one client-initiated `KeyUpdate` carrying
`update_not_requested`, changed its client-traffic read generation, and completed
the exact deterministic post-update transfer without publishing traffic secrets.

Every other committed TLS identity is explicitly unsupported. Unknown identities
remain distinct configuration failures and are never substituted.
