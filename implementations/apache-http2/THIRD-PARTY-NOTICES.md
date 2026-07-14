# Third-party notices

This package is a source-and-configuration-only wrapper. It does not contain
Apache HTTP Server, `mod_http2`, nghttp2, OpenSSL, or Ubuntu container binaries.
At execution time it acquires the digest-pinned Canonical `ubuntu/apache2`
image recorded in `toolchain.json`.

- Upstream: Apache HTTP Server with `mod_http2`
- Runtime package: Ubuntu `apache2` `2.4.58-1ubuntu8.11`
- HTTP/2 dependency: Ubuntu `libnghttp2-14` `1.59.0-1ubuntu0.2`
- Primary license: Apache License 2.0 (with separately licensed dependencies)
- Image: `ubuntu/apache2@sha256:6563a8f98ce5469715962cf217335ec73842e56abb3720094a15f2b6747b87bc`

The package builder includes the repository's Apache License 2.0 text as
`third-party/apache-http-server-LICENSE.txt`. The acquired image retains its
complete Debian-format copyright inventory under `/usr/share/doc/*/copyright`
and common license texts under `/usr/share/common-licenses/`.

The wrapper changes only Apache module enablement, configuration, certificates,
and static fixtures. It does not patch, rebuild, or replace upstream httpd or
`mod_http2` code.
