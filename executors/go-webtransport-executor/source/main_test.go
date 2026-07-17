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
