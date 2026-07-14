# Technitium DoH2 feasibility decision

Working identity: org.protocol-lab.components.implementation.technitium-doh2.

Closed by exact-contract mismatch. A source/config-only wrapper around Technitium DNS Server 15.4 was built from immutable linux/amd64 image digest sha256:b2b6eeeae5057880c7403da426907ccd83070b5c7a1ecfb12135d98b9f4a0b9e and source commit d0484b6c1e7439cdc53d67d81e9c876cda2ad756. The initial live target selected TLS_AES_256_GCM_SHA384; an OpenSSL policy override successfully selected the required TLS_AES_128_GCM_SHA256. Exact execution still failed because Technitium does not emit the contract-required Cache-Control: no-store response header. Its response also advertises recursion until the settings API is set to recursion=Deny; that part is configurable, the header is not.

No package is registered. The GPL-3.0-only image was acquired directly for local audit; binary or derived-image redistribution remains unapproved.

