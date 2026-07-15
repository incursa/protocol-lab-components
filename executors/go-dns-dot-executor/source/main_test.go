package main

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
)

func TestCanonicalFixtureHashesAndLengths(t *testing.T) {
	if len(canonicalQuery) != 27 || hash(canonicalQuery) != queryHash {
		t.Fatalf("canonical query mismatch: length=%d hash=%s", len(canonicalQuery), hash(canonicalQuery))
	}
	if len(canonicalResponse) != 43 || hash(canonicalResponse) != responseHash {
		t.Fatalf("canonical response mismatch: length=%d hash=%s", len(canonicalResponse), hash(canonicalResponse))
	}
	if got := hexString(frame(canonicalQuery)); got != "001b00000000000100000000000004706c616204746573740000010001" {
		t.Fatalf("framed query=%s", got)
	}
}

func TestExchangeClassifiesMismatchedMessageIDAsMalformed(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	go func() {
		defer server.Close()
		prefix := make([]byte, 2)
		_, _ = io.ReadFull(server, prefix)
		query := make([]byte, binary.BigEndian.Uint16(prefix))
		_, _ = io.ReadFull(server, query)
		response := append([]byte(nil), canonicalResponse...)
		binary.BigEndian.PutUint16(response[:2], 0x9999)
		_, _ = server.Write(frame(response))
	}()
	_, _, err := exchange(client, 0x1234)
	var malformed malformedResponseError
	if !errors.As(err, &malformed) {
		t.Fatalf("expected malformed response classification, got %v", err)
	}
}

func TestExchangeCorrelatesRuntimeIDAndNormalizesHashes(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	done := make(chan error, 1)
	go func() {
		defer server.Close()
		prefix := make([]byte, 2)
		if _, err := io.ReadFull(server, prefix); err != nil {
			done <- err
			return
		}
		query := make([]byte, binary.BigEndian.Uint16(prefix))
		if _, err := io.ReadFull(server, query); err != nil {
			done <- err
			return
		}
		response := append([]byte(nil), canonicalResponse...)
		copy(response[:2], query[:2])
		_, err := server.Write(frame(response))
		done <- err
	}()
	proof, bytes, err := exchange(client, 0x1234)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if bytes != 74 {
		t.Fatalf("transferred=%d", bytes)
	}
	if proof.RuntimeMessageID != 0x1234 || proof.ResponseMessageID != 0x1234 || !proof.MessageIDCorrelated {
		t.Fatalf("correlation=%+v", proof)
	}
	if proof.QueryNormalizedSHA256 != queryHash || proof.ResponseNormalizedSHA256 != responseHash {
		t.Fatalf("hashes=%+v", proof)
	}
}

func TestKnownUnsupportedInventoryIsExact(t *testing.T) {
	expected := []string{"dns.classic.tcp.query.a", "dns.classic.udp-truncated-tcp-retry", "dns.classic.udp.query.a", "dns.doh2.query.a", "dns.doh2.interoperability.query.a", "dns.doh3.get.a", "dns.doh3.query.a", "dns.doh3.query.aaaa", "dns.doh3.query.cname-chain", "dns.doh3.query.large-dnssec-shaped", "dns.doh3.query.nodata", "dns.doh3.query.nxdomain", "dns.doh3.interoperability.query.a", "dns.doq.query.a", "dns.doq.interoperability.query.a"}
	if len(knownUnsupported) != len(expected) {
		t.Fatalf("unsupported count=%d", len(knownUnsupported))
	}
	for _, id := range expected {
		if _, ok := knownUnsupported[id]; !ok {
			t.Fatalf("missing %s", id)
		}
	}
}

func TestStrictAndInteroperabilityScenariosAreAdmitted(t *testing.T) {
	for _, id := range []string{strictScenario, interopScenario} {
		if _, ok := supportedScenarios[id]; !ok {
			t.Fatalf("missing supported scenario %s", id)
		}
	}
}

func TestProtocolVariantFollowsSelectedScenario(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", strictScenario)
	if got := protocolVariant(); got != "dot-tls1.3-tcp" {
		t.Fatalf("strict variant=%q", got)
	}
	t.Setenv("PLAB_SCENARIO_ID", interopScenario)
	if got := protocolVariant(); got != "dot-rfc7858-interoperability" {
		t.Fatalf("interop variant=%q", got)
	}
}

func TestNormalizeTargetRejectsFallbackSchemes(t *testing.T) {
	if got, err := normalizeTarget("tls://127.0.0.1:853"); err != nil || got != "127.0.0.1:853" {
		t.Fatalf("got=%s err=%v", got, err)
	}
	for _, value := range []string{"udp://127.0.0.1:53", "tcp://127.0.0.1:53", "https://127.0.0.1/dns-query"} {
		if _, err := normalizeTarget(value); err == nil {
			t.Fatalf("accepted fallback %s", value)
		}
	}
}

func hexString(value []byte) string { return fmtHex(value) }
