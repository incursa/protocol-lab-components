package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractPayloadHashes(t *testing.T) {
	for id, expectation := range expectations {
		if actual := hash(expectation.payload); actual != expectation.payloadHash {
			t.Fatalf("%s hash=%s expected=%s", id, actual, expectation.payloadHash)
		}
	}
}

func TestPackageSourceMatchesAuthorityLockedContracts(t *testing.T) {
	root := filepath.Join("..", "..", "..", "scenarios", "http1-websocket-cleartext-performance")
	for id := range expectations {
		name := strings.TrimPrefix(id, "http1.websocket.rfc6455.cleartext.")
		path := filepath.Join(root, "scenarios", "http1", "websocket", "rfc6455-cleartext-"+name+".yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, required := range []string{"id: " + id, "binding: http1-upgrade", "scheme: ws", "path: /websocket", "masking: client-required", "fragmentation: none", "transportSecurity: cleartext", "protocolVariant: websocket-h1-cleartext-upgrade"} {
			if !strings.Contains(text, required) {
				t.Fatalf("%s missing %q", path, required)
			}
		}
	}
	suite, err := os.ReadFile(filepath.Join(root, "suites", "http1-websocket-cleartext-performance-smoke.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for id := range expectations {
		if !strings.Contains(string(suite), id) {
			t.Fatalf("suite missing %s", id)
		}
	}
	profile, err := os.ReadFile(filepath.Join(root, "load-profiles", "websocket-smoke.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"durationSeconds: 5", "warmupSeconds: 1", "repetitions: 1", "connections: 1", "concurrency: 1", "operationTimeoutMilliseconds: 5000"} {
		if !strings.Contains(string(profile), required) {
			t.Fatalf("profile missing %q", required)
		}
	}
}

func TestSupportedAndUnsupportedInventoriesAreDisjoint(t *testing.T) {
	if len(expectations) != 5 {
		t.Fatalf("supported=%d", len(expectations))
	}
	if len(knownUnsupported) != 20 {
		t.Fatalf("unsupported=%d", len(knownUnsupported))
	}
	for id := range expectations {
		if _, duplicate := knownUnsupported[id]; duplicate {
			t.Fatalf("duplicate %s", id)
		}
	}
}

func TestClientFrameIsMaskedAndRoundTrips(t *testing.T) {
	var wire bytes.Buffer
	if _, err := writeClientFrame(&wire, 0x1, []byte(textPayload)); err != nil {
		t.Fatal(err)
	}
	frame, err := readFrame(bufio.NewReader(&wire), true)
	if err != nil {
		t.Fatal(err)
	}
	if !frame.masked || frame.opcode != 0x1 || string(frame.payload) != textPayload {
		t.Fatalf("frame=%+v", frame)
	}
}

func TestNormalizeTargetRejectsTLSAndWrongPath(t *testing.T) {
	for _, value := range []string{"https://127.0.0.1:1/websocket", "http://127.0.0.1:1/not-websocket"} {
		if _, err := normalizeTarget(value); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	if got, err := normalizeTarget("http://127.0.0.1:18081"); err != nil || !strings.HasSuffix(got, "/websocket") {
		t.Fatalf("got=%s err=%v", got, err)
	}
}
