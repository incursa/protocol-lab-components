# Go classic DNS fixture authority

`go-dns-classic-authority@0.1.0` binds UDP and TCP on the same runner-selected local port and serves only the committed v2 A and large DNSSEC-shaped fixtures. UDP returns the 45-byte TC response for the large fixture; TCP returns the full 630-byte response. It does not resolve recursively, cache, or contact an upstream.

The target is shared implementation material, but the UDP and TCP executor packages remain independent evidence lanes.
