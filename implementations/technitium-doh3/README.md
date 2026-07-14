# Technitium DoH3 feasibility decision

Working identity: org.protocol-lab.components.implementation.technitium-doh3.

Closed by exact-contract mismatch after source/runtime audit of the shared Technitium DoH implementation. The existing DoH3 executor requires an exact Cache-Control: no-store response and TLS_AES_128_GCM_SHA256. Technitium DNS Server 15.4 does not set a DNS-response Cache-Control header, and its QUIC path uses the platform QUIC stack rather than the OpenSSL TLS policy that corrected the TCP DoH2 cipher. Registering the lane would therefore overclaim compatibility.

Audited immutable linux/amd64 image digest: sha256:b2b6eeeae5057880c7403da426907ccd83070b5c7a1ecfb12135d98b9f4a0b9e; source commit d0484b6c1e7439cdc53d67d81e9c876cda2ad756. GPL-3.0-only redistribution remains unapproved.

