package main

import (
	"encoding/binary"
	"testing"
)

func TestCanonicalDoQFixtureAndFraming(t *testing.T) {
	if len(canonicalQuery) != 27 || hash(canonicalQuery) != queryHash {
		t.Fatal("canonical query mismatch")
	}
	if len(canonicalResponse) != 43 || hash(canonicalResponse) != responseHash {
		t.Fatal("canonical response mismatch")
	}
	queryFrame := frame(canonicalQuery)
	responseFrame := frame(canonicalResponse)
	if len(queryFrame) != 29 || binary.BigEndian.Uint16(queryFrame[:2]) != 27 {
		t.Fatal("query framing mismatch")
	}
	if len(responseFrame) != 45 || binary.BigEndian.Uint16(responseFrame[:2]) != 43 {
		t.Fatal("response framing mismatch")
	}
	if err := validateRequestFrame(queryFrame); err != nil {
		t.Fatal(err)
	}
}

func TestRequestValidationRejectsSubstitution(t *testing.T) {
	mutations := [][]byte{
		canonicalQuery,
		append([]byte{0, 26}, canonicalQuery...),
		append([]byte{0, 27}, append([]byte{1}, canonicalQuery[1:]...)...),
		append(frame(canonicalQuery), 0),
	}
	for index, mutation := range mutations {
		if index == 0 {
			mutation = append([]byte{0, 27}, mutation...)
			mutation[len(mutation)-1] ^= 1
		}
		if err := validateRequestFrame(mutation); err == nil {
			t.Fatalf("mutation %d was accepted", index)
		}
	}
}
