package dnsclassic

import "testing"

func TestCanonicalFixtureHashes(t *testing.T) {
	tests := []struct {
		v []byte
		n int
		h string
	}{{aQuery, 27, aQueryHash}, {aResponse, 43, aResponseHash}, {largeQuery, 45, largeQueryHash}, {truncatedResponse, 45, truncatedHash}, {largeResponse, 630, largeResponseHash}}
	for _, x := range tests {
		if len(x.v) != x.n || hash(x.v) != x.h {
			t.Fatalf("fixture mismatch len=%d hash=%s", len(x.v), hash(x.v))
		}
	}
	if got := frame(largeResponse)[:2]; got[0] != 0x02 || got[1] != 0x76 {
		t.Fatalf("prefix=%x", got)
	}
}
func TestTargetSchemesFailClosed(t *testing.T) {
	if _, e := normalizeTarget("udp://127.0.0.1:53", "udp"); e != nil {
		t.Fatal(e)
	}
	if _, e := normalizeTarget("tcp://127.0.0.1:53", "udp"); e == nil {
		t.Fatal("UDP accepted TCP")
	}
	if _, e := normalizeTarget("udp://127.0.0.1:53", "tcp"); e == nil {
		t.Fatal("TCP accepted UDP")
	}
}
func TestEveryCommittedDNSIdentityRecognized(t *testing.T) {
	if len(allDNS) != 13 {
		t.Fatalf("identity count=%d", len(allDNS))
	}
}
