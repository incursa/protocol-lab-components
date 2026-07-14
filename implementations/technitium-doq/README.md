# Technitium DoQ feasibility decision

Working identity: org.protocol-lab.components.implementation.technitium-doq.

Closed by exact-contract mismatch. A live source/config-only Technitium DNS Server 15.4 DoQ wrapper was built and bootstrapped with recursion denied and the deterministic plab.test A zone. The existing DoQ executor rejected its QUIC handshake because the platform QUIC stack selected a cipher outside the required TLS_AES_128_GCM_SHA256 policy. An OpenSSL policy override cannot control the platform QUIC stack, and Technitium exposes no cipher-suite control.

Audited immutable linux/amd64 image digest: sha256:b2b6eeeae5057880c7403da426907ccd83070b5c7a1ecfb12135d98b9f4a0b9e; source commit d0484b6c1e7439cdc53d67d81e9c876cda2ad756. No package is registered; GPL-3.0-only redistribution remains unapproved.

