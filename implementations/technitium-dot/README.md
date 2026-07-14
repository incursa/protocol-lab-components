# Technitium DoT feasibility decision

Working identity: org.protocol-lab.components.implementation.technitium-dot.

Closed by exact-contract mismatch. A source/config-only wrapper around Technitium DNS Server 15.4 was built from immutable linux/amd64 image digest sha256:b2b6eeeae5057880c7403da426907ccd83070b5c7a1ecfb12135d98b9f4a0b9e and source commit d0484b6c1e7439cdc53d67d81e9c876cda2ad756. The existing DoT executor rejected the live target because Technitium did not negotiate the required dot ALPN and selected TLS_AES_256_GCM_SHA384 instead of TLS_AES_128_GCM_SHA256. Technitium exposes neither control in its server settings API.

No package is registered. The GPL-3.0-only image was acquired directly for local audit; binary or derived-image redistribution remains unapproved.

