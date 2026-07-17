# Third-party notices

## wolfSSL

- Project: https://github.com/wolfSSL/wolfssl
- Version/tag: `5.9.2` / `v5.9.2-stable`
- Commit: `ac01707f552c611fbd135cc723b2682b3e7f80f2`
- Source archive SHA-256: `2f4ef3d4fd387a9b3191d36a6316d69116c46ff69bb9583b6c82b36d7b8ca114`
- License: GPL-3.0-or-later

The `.plabpkg` contains build instructions and the ProtocolLab fixture, not a
prebuilt wolfSSL binary. The Docker build obtains the complete official source
archive, verifies its hash, builds locally, and copies the upstream `COPYING`
file into the runtime image.
