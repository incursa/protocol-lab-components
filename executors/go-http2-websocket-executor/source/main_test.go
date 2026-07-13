package main

import (
	"bytes"
	"testing"
)

func TestMaskedFrameCarriesMaskAndRoundTrips(t *testing.T) {
	key, encoded, err := maskedFrame(0x1, []byte(textPayload))
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 4 || encoded[1]&0x80 == 0 {
		t.Fatal("client frame was not masked")
	}
	frame, ok, err := parseFrame(encoded)
	if err != nil || !ok {
		t.Fatalf("parse: ok=%t err=%v", ok, err)
	}
	if !frame.Masked || frame.Opcode != 0x1 || !bytes.Equal(frame.Payload, []byte(textPayload)) {
		t.Fatalf("unexpected frame: %#v", frame)
	}
}

func TestParseFrameRejectsFragmentationAndRSV(t *testing.T) {
	for _, first := range []byte{0x01, 0xC1} {
		if _, _, err := parseFrame([]byte{first, 0}); err == nil {
			t.Fatalf("accepted first byte 0x%x", first)
		}
	}
}

func TestStrictOrderingRequiresExactOneThroughOneHundred(t *testing.T) {
	values := make([]int, 100)
	for index := range values {
		values[index] = index + 1
	}
	if !strictOrder(values) {
		t.Fatal("exact order rejected")
	}
	values[50], values[51] = values[51], values[50]
	if strictOrder(values) {
		t.Fatal("out-of-order sequence accepted")
	}
}

func TestSixExactScenariosAndAdjacentUnsupported(t *testing.T) {
	if len(scenarios) != 6 {
		t.Fatalf("scenario count = %d", len(scenarios))
	}
	if scenarios["http2.websocket.rfc8441.multi-message-text-echo"].MessageCount != 100 {
		t.Fatal("multi-message count mismatch")
	}
	if !contains(knownUnsupported, "http1.websocket.rfc6455.tls.text-echo") || !contains(knownUnsupported, "http3.websocket.rfc9220.extended-connect") {
		t.Fatal("adjacent binding IDs not recognized unsupported")
	}
}
