package main

import "testing"

func TestPayloadIdentity(t *testing.T) {
	payload := makePayload()
	if len(payload) != payloadBytes {
		t.Fatalf("payload length = %d", len(payload))
	}
	if got := hash(payload); got != payloadSHA256 {
		t.Fatalf("payload hash = %s", got)
	}
}

func TestDatagramPayloadSetIdentity(t *testing.T) {
	payloads := makeDatagramPayloadSet()
	if len(payloads) != datagramCount {
		t.Fatalf("datagram count = %d", len(payloads))
	}
	for index, payload := range payloads {
		if len(payload) != payloadBytesPerDatagram {
			t.Fatalf("datagram %d length = %d", index, len(payload))
		}
	}
	if got := hashPayloadSet(payloads); got != payloadSetSHA256 {
		t.Fatalf("payload set hash = %s", got)
	}
}

func TestNormalizeTargetPreservesLogicalAuthority(t *testing.T) {
	actual, logical, err := normalizeTarget("https://10.10.99.85:5461")
	if err != nil {
		t.Fatal(err)
	}
	if actual != "10.10.99.85:5461" {
		t.Fatalf("actual = %s", actual)
	}
	if logical != "https://webtransport.plab.test:5461/webtransport/echo" {
		t.Fatalf("logical = %s", logical)
	}
}
