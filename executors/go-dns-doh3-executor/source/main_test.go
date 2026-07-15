package main

import "testing"

func TestCanonicalFixtures(t *testing.T) {
	if len(fixtures) != 8 {
		t.Fatalf("count=%d", len(fixtures))
	}
	for _, f := range fixtures {
		if err := validateFixture(f); err != nil {
			t.Errorf("%s: %v", f.ScenarioID, err)
		}
	}
}
func TestSupportedScenarioOrder(t *testing.T) {
	ids := sortedScenarioIDs()
	if len(ids) != 8 || ids[0] != "dns.doh3.get.a" || ids[1] != "dns.doh3.interoperability.query.a" || ids[7] != "dns.doh3.query.nxdomain" {
		t.Fatalf("ids=%v", ids)
	}
}
func TestTargetNormalization(t *testing.T) {
	got, err := normalizeTarget("https://127.0.0.1:18533")
	if err != nil || got != "https://127.0.0.1:18533/dns-query" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}
