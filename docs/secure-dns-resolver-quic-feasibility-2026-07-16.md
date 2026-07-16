# Secure DNS Resolver QUIC Feasibility Decision

Date: 2026-07-16

This decision closes the implementation-coverage wishlist evaluation of Knot
Resolver and Technitium DNS Server in recursive-resolver roles for DNS over
QUIC (DoQ) and DNS over HTTPS/3 (DoH3). It does not reduce any live-coverage
floor and does not treat an authoritative-server scenario as resolver proof.

## Exact public-contract boundary

`REQ-PL-SECDNS-0014` and the public fixture catalog deliberately limit the
initial recursive-resolver surface to DoT and DoH2. The resolver scenarios
require RD in the query, RA set and AA clear in the response, a cache flush
before each measured operation, no cache reuse, a runner-provided local fixture
authority as the only upstream, no external upstream, and resolver plus
upstream proof artifacts. The existing DoQ and DoH3 scenarios instead select
the authoritative-server role. Substituting those scenarios would mislabel the
implementation role and comparison cohort.

## Candidate decisions

| Candidate | Upstream capability | Current ProtocolLab decision |
| --- | --- | --- |
| Knot Resolver DoQ | Knot Resolver 6.4 documents a native RFC 9250 `doq` listener; the feature is marked beta. | Blocked from package declaration because no resolver-role DoQ scenario, fixture, cache-control artifact contract, or suite exists. The authoritative DoQ scenario is not a substitute. Re-evaluate if the public contract adds resolver DoQ. |
| Knot Resolver DoH3 | Knot Resolver 6.4 documents `doh2` for DNS over HTTPS and does not document a native DoH3 listener. | Unsupported by the selected native resolver. A reverse proxy would substitute the measured HTTP/3/QUIC transport and is prohibited. |
| Technitium resolver DoQ | Technitium DNS Server supports hosted DoQ, including recursive service. | Blocked because no resolver-role DoQ public contract exists. The existing Technitium authoritative audit also found that its platform QUIC stack could not be constrained to the exact required cipher cohort; changing DNS role does not correct that transport mismatch. Redistribution approval for the GPL-3.0-only image also remains absent. |
| Technitium resolver DoH3 | Technitium DNS Server supports HTTP/3 for its hosted DoH service, including recursive service. | Blocked because no resolver-role DoH3 public contract exists. The existing authoritative audit also found an absent required DNS-response cache header and uncontrollable QUIC cipher selection; changing DNS role does not correct those transport mismatches. Redistribution approval for the GPL-3.0-only image also remains absent. |

## Re-entry conditions

A candidate can be reopened only through a versioned public resolver-role
contract for the exact transport, followed by package-local proof of role,
cache flush, local-only upstream, DNS wire semantics, transport negotiation,
and artifacts. Knot Resolver DoQ additionally needs an explicit decision about
admitting a beta upstream feature. Technitium additionally needs license and
redistribution approval plus exact transport-policy proof. Until then these
lanes remain visible, repository-backed unsupported decisions and do not count
toward any implementation floor.

## Sources

- Knot Resolver 6.4 encrypted DNS configuration: <https://www.knot-resolver.cz/documentation/latest/config-network-server-tls.html>
- Knot Resolver 6.4 release notes: <https://www.knot-resolver.cz/documentation/v6.4.0/NEWS.html>
- Technitium DoQ and HTTP/3 configuration: <https://blog.technitium.com/2023/02/configuring-dns-over-quic-and-https3.html>
- Public resolver-role requirement: `protocol-lab/specs/requirements/SPEC-PL-SECURE-DNS-PERFORMANCE.json`
- Public fixture catalog: `protocol-lab/docs/protocol-lab/secure-dns-fixture-catalog.md`
- Existing Technitium exact-contract decisions: `implementations/technitium-doq/README.md` and `implementations/technitium-doh3/README.md`
