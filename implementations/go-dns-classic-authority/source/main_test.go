package main

import (
	"encoding/binary"
	"testing"
)

func TestCanonicalFixtureMaterial(t *testing.T) {
	tests := []struct {
		name   string
		value  []byte
		length int
		digest string
	}{
		{"a", aResponse, 43, "9d488461675ad5ab9f74c7b203861e1ad17521e413a407a25e6611012a595620"},
		{"truncated", truncatedResponse, 45, "753eff72120531d638e839e088ca22875e550b89f554a11af50d4423a512710d"},
		{"large", largeResponse, 630, "1cc5bafd114a4f34d824c01a685b04e82cae6110c648046c461d3688782e8665"},
	}
	for _, test := range tests {
		if len(test.value) != test.length || hash(test.value) != test.digest {
			t.Fatalf("%s length/hash mismatch", test.name)
		}
	}
	if got := binary.BigEndian.Uint16(frame(largeResponse)[:2]); got != 630 {
		t.Fatalf("large prefix=%d", got)
	}
}

func TestResponsesPreserveRuntimeIDAndBinding(t *testing.T) {
	a := mustHex("12340000000100000000000004706c616204746573740000010001")
	udp, err := responseFor(a, true)
	if err != nil {
		t.Fatal(err)
	}
	if binary.BigEndian.Uint16(udp[:2]) != 0x1234 || len(udp) != 43 {
		t.Fatal("A UDP response mismatch")
	}
	large := mustHex("22220000000100000000000106646e736b657904706c6162047465737400003000010000290200000080000000")
	truncated, err := responseFor(large, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(truncated) != 45 || truncated[2]&0x02 == 0 {
		t.Fatal("UDP truncation mismatch")
	}
	full, err := responseFor(large, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(full) != 630 || full[2]&0x02 != 0 {
		t.Fatal("TCP full response mismatch")
	}
}

func TestRejectsZeroAndUnknownQueries(t *testing.T) {
	if _, err := responseFor(make([]byte, 27), true); err == nil {
		t.Fatal("accepted zero ID")
	}
	unknown := make([]byte, 27)
	binary.BigEndian.PutUint16(unknown[:2], 1)
	if _, err := responseFor(unknown, true); err == nil {
		t.Fatal("accepted unknown query")
	}
}
