package main

import "testing"

func TestCanonicalFixtureHashesLengthsAndSemantics(t *testing.T) {
	if len(canonicalQuery) != 27 || hash(canonicalQuery) != queryHash {
		t.Fatal("canonical query mismatch")
	}
	if len(canonicalResponse) != 43 || hash(canonicalResponse) != responseHash {
		t.Fatal("canonical response mismatch")
	}
	proof, err := validateDNSFrame(frame(canonicalResponse))
	if err != nil {
		t.Fatal(err)
	}
	if proof.Transport != "doq" || proof.RuntimeMessageID != 0 || proof.CanonicalHashNormalization != "identity" || proof.ResponseNormalizedSHA256 != responseHash {
		t.Fatalf("proof=%+v", proof)
	}
}

func TestKnownUnsupportedInventoryIsExact(t *testing.T) {
	expected := []string{"dns.classic.tcp.query.a", "dns.classic.udp-truncated-tcp-retry", "dns.classic.udp.query.a", "dns.doh2.query.a", "dns.doh2.interoperability.query.a", "dns.doh3.get.a", "dns.doh3.query.a", "dns.doh3.interoperability.query.a", "dns.doh3.query.aaaa", "dns.doh3.query.cname-chain", "dns.doh3.query.large-dnssec-shaped", "dns.doh3.query.nodata", "dns.doh3.query.nxdomain", "dns.dot.query.a", "dns.dot.interoperability.query.a"}
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
	for _, id := range []string{strictScenario, interopScenario} {
		if _, ok := supportedScenarios[id]; !ok {
			t.Fatalf("missing supported scenario %s", id)
		}
	}
}

func TestProtocolVariantFollowsSelectedScenario(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", strictScenario)
	if got := protocolVariant(); got != "dns-over-quic-v1" {
		t.Fatalf("strict variant=%q", got)
	}
	t.Setenv("PLAB_SCENARIO_ID", interopScenario)
	if got := protocolVariant(); got != "doq-rfc9250-interoperability" {
		t.Fatalf("interop variant=%q", got)
	}
}

func TestNormalizeTargetRequiresDistinctDoQLane(t *testing.T) {
	got, err := normalizeTarget("doq://127.0.0.1:18532")
	if err != nil || got != "127.0.0.1:18532" {
		t.Fatalf("got=%s err=%v", got, err)
	}
	for _, value := range []string{"quic://127.0.0.1:18532", "tls://127.0.0.1:18532", "https://127.0.0.1:18532", "doq://127.0.0.1", "doq://127.0.0.1:18532/dns-query", "doq://127.0.0.1:18532?x=1"} {
		if _, err := normalizeTarget(value); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}

func TestDNSParserRejectsFramingAndSemanticSubstitution(t *testing.T) {
	base := frame(canonicalResponse)
	for _, index := range []int{0, 2, 4, 8, len(base) - 1} {
		value := append([]byte(nil), base...)
		value[index] ^= 1
		if _, err := validateDNSFrame(value); err == nil {
			t.Fatalf("accepted mutation at %d", index)
		}
	}
}
