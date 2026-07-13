# Go DoH3 local fixture authority

Independent pinned `quic-go` HTTP/3 origin serving only the seven public deterministic DNS fixtures. It permits exact POST and the canonical unpadded-base64url GET binding, disables cache/upstream/recursion, and supports QUIC v1 with fresh TLS 1.3 and `h3` ALPN only.
