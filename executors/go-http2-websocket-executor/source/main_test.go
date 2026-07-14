package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestImmutableExecutorAndProfileIdentities(t *testing.T) {
	if executorVersion != "0.2.0" || loadGeneratorVersion != "0.2.0" {
		t.Fatalf("immutable identities = %s/%s", executorVersion, loadGeneratorVersion)
	}
	if authorityCommit != "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574" {
		t.Fatalf("authority commit = %s", authorityCommit)
	}
	if smokeProfile != (loadProfile{ID: "websocket-smoke", Connections: 1, Concurrency: 1, StreamsPerConnection: 1, Duration: 5 * time.Second, Warmup: time.Second, Repetitions: 1, OperationTimeout: 5 * time.Second}) {
		t.Fatalf("smoke profile = %#v", smokeProfile)
	}
	if diagnosticProfile != (loadProfile{ID: "diagnostic", Connections: 1, Concurrency: 8, StreamsPerConnection: 8, Duration: 10 * time.Second, Warmup: time.Second, Cooldown: time.Second, Repetitions: 1, OperationTimeout: 10 * time.Second}) {
		t.Fatalf("diagnostic profile = %#v", diagnosticProfile)
	}
	if got := profileFor(scenarios["http2.websocket.rfc8441.multi-message-text-echo"]); got.ID != diagnosticProfileID {
		t.Fatalf("multi-message profile = %#v", got)
	}
	for id, spec := range scenarios {
		if strings.Contains(id, "multi-message") {
			continue
		}
		if got := profileFor(spec); got.ID != smokeProfileID {
			t.Fatalf("%s profile = %#v", id, got)
		}
	}
}

func TestExactScenarioConstants(t *testing.T) {
	multi := scenarios["http2.websocket.rfc8441.multi-message-text-echo"]
	if multi.MessageCount != 100 || string(multi.Payload) != "protocol-lab" || len(multi.Payload) != 12 || multi.PayloadHash != textHash {
		t.Fatalf("multi-message constants = %#v", multi)
	}
	if hash(multi.Payload) != "504585b0bb4fd77012ea2575efbcdb58f4c33e6b543e9567a65896d213720c29" {
		t.Fatal("multi-message payload hash mismatch")
	}
	if string(scenarios["http2.websocket.rfc8441.control-frames"].Payload) != "protocol-lab-ping" || controlHash != "4848de689e96825e0e05b6c3e96e48f2ad7ec7805fb64ecf824ccc6aa9c58883" {
		t.Fatal("control payload constants mismatch")
	}
	if len(scenarios["http2.websocket.rfc8441.binary-echo"].Payload) != 1024 || binaryHash != "9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a" {
		t.Fatal("binary payload constants mismatch")
	}
}

func TestAuthorityLockParity(t *testing.T) {
	packageRoot := filepath.Clean(filepath.Join("..", "..", "..", "scenarios", "http2-websocket-performance"))
	data, err := os.ReadFile(filepath.Join(packageRoot, "authority-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var authority struct {
		Commit string            `json:"commit"`
		Files  map[string]string `json:"files"`
	}
	if err = json.Unmarshal(data, &authority); err != nil {
		t.Fatal(err)
	}
	if authority.Commit != authorityCommit {
		t.Fatalf("authority commit = %s", authority.Commit)
	}
	required := []string{
		"scenarios/http2/websocket/rfc8441-extended-connect.yaml",
		"scenarios/http2/websocket/rfc8441-control-frames.yaml",
		"scenarios/http2/websocket/rfc8441-text-echo.yaml",
		"scenarios/http2/websocket/rfc8441-binary-echo.yaml",
		"scenarios/http2/websocket/rfc8441-close.yaml",
		"scenarios/http2/websocket/rfc8441-multi-message-text-echo.yaml",
		"load-profiles/websocket-smoke.yaml",
		"load-profiles/diagnostic.yaml",
	}
	for _, relative := range required {
		expected := authority.Files[relative]
		if expected == "" {
			t.Fatalf("authority lock missing %s", relative)
		}
		contents, readErr := os.ReadFile(filepath.Join(packageRoot, filepath.FromSlash(relative)))
		if readErr != nil {
			t.Fatal(readErr)
		}
		sum := sha256.Sum256(contents)
		if observed := hex.EncodeToString(sum[:]); observed != expected {
			t.Fatalf("%s hash = %s, want %s", relative, observed, expected)
		}
	}
}

func TestParseFrameConsumedPreservesAdjacentFrames(t *testing.T) {
	_, first, err := maskedFrame(0x1, []byte(textPayload))
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := maskedFrame(0x1, []byte(textPayload))
	if err != nil {
		t.Fatal(err)
	}
	joined := append(first, second...)
	frame, consumed, ok, err := parseFrameConsumed(joined)
	if err != nil || !ok || consumed != len(first) || string(frame.Payload) != textPayload {
		t.Fatalf("first adjacent frame: consumed=%d ok=%t err=%v", consumed, ok, err)
	}
	_, consumed, ok, err = parseFrameConsumed(joined[consumed:])
	if err != nil || !ok || consumed != len(second) {
		t.Fatalf("second adjacent frame: consumed=%d ok=%t err=%v", consumed, ok, err)
	}
}

func TestMaskKeyCountsRejectsReuse(t *testing.T) {
	unique, duplicates := maskKeyCounts([]string{"a", "b", "a"})
	if unique != 2 || duplicates != 1 {
		t.Fatalf("mask key counts = %d/%d", unique, duplicates)
	}
}
