# Third-party notices

This package contains build instructions for the OpenSSL executable. It does
not include a prebuilt third-party binary in the `.plabpkg`; Docker builds the
exact upstream OpenSSL `3.3.0` source release. The annotated tag object is
`24e7fcf7aff2caadbdee879f615c63981ed132dc`, its dereferenced source commit is
`4cb31128b5790819dfeea2739fbde265f71a10a2`, and the official release archive
SHA-256 is `53e66b043322a606abf0087e7699a0e033a37fa13feb9742df35c3a33b18fb02`.
OpenSSL 3.3.0 is licensed under Apache-2.0.
The source release's `LICENSE.txt` is copied into the built target image at
`/usr/share/licenses/openssl/LICENSE.txt`.

- Project: OpenSSL
- Source: https://github.com/openssl/openssl
- Release source: https://github.com/openssl/openssl/tree/openssl-3.3.0
- Release archive: https://github.com/openssl/openssl/releases/download/openssl-3.3.0/openssl-3.3.0.tar.gz
- License: https://github.com/openssl/openssl/blob/openssl-3.3.0/LICENSE.txt

The package-local certificate and key are public ProtocolLab test fixtures and
must never be used outside isolated test environments.
