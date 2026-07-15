package main

import "testing"

func TestCanonicalFixtures(t *testing.T) {
	if len(fixtures) != 8 {
		t.Fatalf("fixture count=%d", len(fixtures))
	}
	for _, f := range fixtures {
		if err := validateFixture(f); err != nil {
			t.Errorf("%s: %v", f.ScenarioID, err)
		}
	}
}
func TestGetValue(t *testing.T) {
	got := getValue(fixtures["dns.doh3.get.a"])
	if got != "AAAAAAABAAAAAAAABHBsYWIEdGVzdAAAAQAB" {
		t.Fatalf("GET value=%s", got)
	}
	if got[len(got)-1] == '=' {
		t.Fatal("GET value is padded")
	}
}
func TestSemanticCompressionTolerance(t *testing.T) {
	f := fixtures["dns.doh3.query.a"]
	if !semanticEqual(bytesOf(f.ResponseHex), f) {
		t.Fatal("canonical response not semantically equal")
	}
}
