# Third-party notices

This package contains build instructions for the GnuTLS `gnutls-serv`
executable. It does not include a prebuilt third-party binary in the
`.plabpkg`; Docker builds the exact upstream GnuTLS `3.8.9` source release.
The annotated tag object is `011bda1be01e4a47224adb3cbc32fcb06cba7be1`, its
dereferenced source commit is `477a733247460b94cd2b37a10579c27ca6fc196f`,
and the official release archive SHA-256 is
`69e113d802d1670c4d5ac1b99040b1f2d5c7c05daec5003813c049b5184820ed`.
`gnutls-serv` reports that it is licensed under GPL-3.0-or-later.
The source release's `COPYING` license text is copied into the built target
image at `/usr/share/licenses/gnutls/COPYING`.

- Project: GnuTLS
- Source: https://gitlab.com/gnutls/gnutls
- Release source: https://gitlab.com/gnutls/gnutls/-/tree/3.8.9
- Release archive: https://www.gnupg.org/ftp/gcrypt/gnutls/v3.8/gnutls-3.8.9.tar.xz
- License: https://gitlab.com/gnutls/gnutls/-/blob/3.8.9/COPYING

The package-local certificate and key are public ProtocolLab test fixtures and
must never be used outside isolated test environments.
