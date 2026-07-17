package main

import "testing"

func TestPayloadSetIdentity(t *testing.T) {
	payloads := makePayloadSet()
	if len(payloads) != datagramCount {
		t.Fatalf("datagram count = %d", len(payloads))
	}
	for i, payload := range payloads {
		if len(payload) != payloadBytesPerDatagram {
			t.Fatalf("payload %d length = %d", i, len(payload))
		}
	}
	if got := hashPayloadSet(payloads); got != payloadSetSHA256 {
		t.Fatalf("payload set hash = %s", got)
	}
}

func TestNormalizeTargetPreservesLogicalAuthority(t *testing.T) {
	actual, template, err := normalizeTarget("https://10.10.99.85:5471")
	if err != nil {
		t.Fatal(err)
	}
	if actual != "10.10.99.85:5471" {
		t.Fatalf("actual = %s", actual)
	}
	if template != "https://masque-proxy.plab.test:5471/.well-known/masque/udp/{target_host}/{target_port}/" {
		t.Fatalf("template = %s", template)
	}
}
