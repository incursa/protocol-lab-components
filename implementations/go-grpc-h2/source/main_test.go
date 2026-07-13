package main

import (
	"bytes"
	"io"
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

func TestStreamingFrameSequenceIsCanonical(t *testing.T) {
	payload := bytes.Repeat([]byte{'B'}, 1024)
	protobuf := encodeTestProtobuf(payload)
	frame, err := encodeFrame(protobuf, "identity")
	if err != nil {
		t.Fatal(err)
	}
	sequence := bytes.Repeat(frame, streamMessageCount)
	reader := bytes.NewReader(sequence)
	for index := 0; index < streamMessageCount; index++ {
		observedFrame, observedProtobuf, observedPayload, err := readIdentityFrame(reader)
		if err != nil {
			t.Fatalf("message %d: %v", index, err)
		}
		if !bytes.Equal(observedFrame, frame) || !bytes.Equal(observedProtobuf, protobuf) || !validStreamingPayload(observedPayload, observedProtobuf) {
			t.Fatalf("message %d drifted from the canonical byte scopes", index)
		}
	}
	var extra [1]byte
	if _, err := reader.Read(extra[:]); err != io.EOF {
		t.Fatalf("expected exact stream exhaustion, observed %v", err)
	}
}

func TestStreamingRejectsNonCanonicalPayload(t *testing.T) {
	payload := bytes.Repeat([]byte{'X'}, 1024)
	if validStreamingPayload(payload, encodeTestProtobuf(payload)) {
		t.Fatal("streaming payload substitution accepted")
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
