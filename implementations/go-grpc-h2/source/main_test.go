package main

import (
	"bytes"
	"testing"
)

func TestSupportedUnaryFrames(t *testing.T) {
	cases := []struct {
		method, compression string
		payload             []byte
	}{{"UnaryEcho", "identity", nil}, {"UnaryEcho", "identity", bytes.Repeat([]byte{'G'}, 128)}, {"UnaryEcho", "identity", bytes.Repeat([]byte{'L'}, 1<<20)}, {"UnaryFixedMetadata", "identity", bytes.Repeat([]byte{'G'}, 128)}, {"UnaryGzip", "gzip", bytes.Repeat([]byte{'B'}, 1024)}}
	for _, tc := range cases {
		protobuf := encodeTestProtobuf(tc.payload)
		frame, err := encodeFrame(protobuf, tc.compression)
		if err != nil {
			t.Fatal(err)
		}
		observedProtobuf, observedPayload, err := decodeFrame(frame, tc.compression)
		if err != nil {
			t.Fatal(err)
		}
		if !validScenarioPayload(tc.method, observedPayload, observedProtobuf) {
			t.Fatalf("rejected %s/%d", tc.method, len(tc.payload))
		}
	}
}
func TestRejectsPayloadSubstitution(t *testing.T) {
	payload := bytes.Repeat([]byte{'X'}, 128)
	if validScenarioPayload("UnaryEcho", payload, encodeTestProtobuf(payload)) {
		t.Fatal("substitution accepted")
	}
}
func encodeTestProtobuf(payload []byte) []byte {
	if len(payload) == 0 {
		return nil
	}
	out := []byte{0x0a}
	n := uint64(len(payload))
	for n >= 0x80 {
		out = append(out, byte(n)|0x80)
		n >>= 7
	}
	out = append(out, byte(n))
	return append(out, payload...)
}
