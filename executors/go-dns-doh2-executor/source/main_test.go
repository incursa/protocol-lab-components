package main

import "testing"

func TestCanonicalFixtureHashesAndLengths(t *testing.T) {
	if len(canonicalQuery) != 27 || hash(canonicalQuery) != queryHash {
		t.Fatal("canonical query mismatch")
	}
	if len(canonicalResponse) != 43 || hash(canonicalResponse) != responseHash {
		t.Fatal("canonical response mismatch")
	}
	proof, err := validateDNS(canonicalResponse)
	if err != nil {
		t.Fatal(err)
	}
	if proof.ResponseNormalizedSHA256 != responseHash || proof.ResponseMessageID != 0 {
		t.Fatalf("proof=%+v", proof)
	}
	if len(resolverQuery) != 27 || hash(resolverQuery) != resolverQueryHash || len(resolverResponse) != 43 || hash(resolverResponse) != resolverResponseHash {
		t.Fatal("resolver fixture mismatch")
	}
}
func TestKnownUnsupportedInventoryIsExact(t *testing.T) {
	expected := []string{"dns.classic.tcp.query.a", "dns.classic.udp-truncated-tcp-retry", "dns.classic.udp.query.a", "dns.doh3.get.a", "dns.doh3.query.a", "dns.doh3.query.aaaa", "dns.doh3.query.cname-chain", "dns.doh3.query.large-dnssec-shaped", "dns.doh3.query.nodata", "dns.doh3.query.nxdomain", "dns.doh3.interoperability.query.a", "dns.doq.query.a", "dns.doq.interoperability.query.a", "dns.dot.query.a", "dns.dot.interoperability.query.a"}
	if len(knownUnsupported) != len(expected) {
		t.Fatalf("count=%d", len(knownUnsupported))
	}
	for _, id := range expected {
		if _, ok := knownUnsupported[id]; !ok {
			t.Fatalf("missing %s", id)
		}
	}
}
func TestStrictAndInteroperabilityScenariosAreAdmitted(t *testing.T) {
	for _, id := range []string{strictScenario, interopScenario, resolverScenario} {
		if _, ok := supportedScenarios[id]; !ok {
			t.Fatalf("missing supported scenario %s", id)
		}
	}
}
func TestProtocolVariantFollowsSelectedScenario(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", strictScenario)
	if got := protocolVariant(); got != "doh-h2-tls-alpn" {
		t.Fatalf("strict variant=%q", got)
	}
	t.Setenv("PLAB_SCENARIO_ID", interopScenario)
	if got := protocolVariant(); got != "doh-h2-rfc8484-interoperability" {
		t.Fatalf("interop variant=%q", got)
	}
	if tlsProfileID() != interopTLSProfile || selectedCertificateProfile() != interopCertProfile {
		t.Fatal("interoperability TLS profiles were not selected")
	}
	t.Setenv("PLAB_SCENARIO_ID", resolverScenario)
	if got := protocolVariant(); got != "doh-h2-rfc8484-recursive-resolver" {
		t.Fatalf("resolver variant=%q", got)
	}
	if selectedFixtureID() != resolverFixtureID {
		t.Fatalf("resolver fixture=%q", selectedFixtureID())
	}
	proof, err := validateDNS(resolverResponse)
	if err != nil || !proof.RecursionDesired || !proof.RecursionAvailable || proof.AuthoritativeAnswer {
		t.Fatalf("resolver proof=%+v err=%v", proof, err)
	}
}
func TestNormalizeTargetRejectsFallbackAndWrongPath(t *testing.T) {
	got, err := normalizeTarget("https://127.0.0.1:18531")
	if err != nil || got != "https://127.0.0.1:18531/dns-query" {
		t.Fatalf("got=%s err=%v", got, err)
	}
	for _, value := range []string{"http://127.0.0.1:18531", "tls://127.0.0.1:853", "https://127.0.0.1:18531/other", "https://127.0.0.1:18531/dns-query?dns=x"} {
		if _, err := normalizeTarget(value); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}
func TestResponseSemanticParserRejectsSubstitution(t *testing.T) {
	for _, index := range []int{0, 2, 6, len(canonicalResponse) - 1} {
		value := append([]byte(nil), canonicalResponse...)
		value[index] ^= 1
		if _, err := validateDNS(value); err == nil {
			t.Fatalf("accepted mutation at %d", index)
		}
	}
}
