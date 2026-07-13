package main

import (
	"bufio"
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
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
	root := filepath.Join("..", "..", "..", "scenarios", "http1-websocket-tls-performance")
	for id := range expectations {
		name := strings.TrimPrefix(id, "http1.websocket.rfc6455.tls.")
		path := filepath.Join(root, "scenarios", "http1", "websocket", "rfc6455-tls-"+name+".yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, required := range []string{"id: " + id, "binding: http1-upgrade", "scheme: wss", "path: /websocket", "masking: client-required", "fragmentation: none", "transportSecurity: tls", "protocolVariant: websocket-h1-tls1.3-upgrade"} {
			if !strings.Contains(text, required) {
				t.Fatalf("%s missing %q", path, required)
			}
		}
	}
	suite, err := os.ReadFile(filepath.Join(root, "suites", "http1-websocket-tls-performance-smoke.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	canonicalSuiteIDs := []string{
		"http1.websocket.rfc6455.tls.upgrade",
		"http1.websocket.rfc6455.tls.control-frames",
		"http1.websocket.rfc6455.tls.text-echo",
		"http1.websocket.rfc6455.tls.binary-echo",
		"http1.websocket.rfc6455.tls.close",
	}
	for _, id := range canonicalSuiteIDs {
		if !strings.Contains(string(suite), id) {
			t.Fatalf("suite missing %s", id)
		}
	}
	for _, diagnostic := range []string{"http1.websocket.rfc6455.tls.subprotocol-text-echo", "http1.websocket.rfc6455.tls.permessage-deflate-binary-echo"} {
		if strings.Contains(string(suite), diagnostic) {
			t.Fatalf("canonical five-ID suite was broadened with %s", diagnostic)
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
	if len(expectations) != 7 {
		t.Fatalf("supported=%d", len(expectations))
	}
	if len(knownUnsupported) != 18 {
		t.Fatalf("unsupported=%d", len(knownUnsupported))
	}
	for id := range expectations {
		if _, duplicate := knownUnsupported[id]; duplicate {
			t.Fatalf("duplicate %s", id)
		}
	}
}

func TestOpeningHandshakeAggregateDetectsKeyReuseAndAcceptMismatch(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2a}, 16))
	proof := handshakeProof{
		Binding: "http1-upgrade", Scheme: "wss", TransportSecurity: "tls",
		Endpoint: "/websocket", Authority: "websocket.plab.test", RequestMethod: "GET",
		RequestedHTTPVersion: "HTTP/1.1", ObservedHTTPVersion: "HTTP/1.1", ResponseStatus: 101,
		UpgradeHeader: "websocket", ConnectionHeader: "Upgrade", SecWebSocketVersion: "13",
		SampleSecWebSocketKey: key, ExpectedSecWebSocketAccept: websocketAccept(key),
		ObservedSecWebSocketAccept: websocketAccept(key),
	}
	summary := newPhaseSummary("test")
	recordOpeningHandshake(&summary, proof, scenarioExpectation{})
	proof.ObservedSecWebSocketAccept = "mismatch"
	recordOpeningHandshake(&summary, proof, scenarioExpectation{})
	if summary.OpeningHandshakes != 2 || summary.KeyReuseCount != 1 || summary.InvalidDecodedKeyCount != 0 || summary.AcceptMismatchCount != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	if !summary.UpgradeRequestHeadersMatched || summary.UpgradeResponseHeadersMatched {
		t.Fatalf("requestMatched=%t responseMatched=%t", summary.UpgradeRequestHeadersMatched, summary.UpgradeResponseHeadersMatched)
	}
}

func TestDiagnosticAuthoritySemantics(t *testing.T) {
	root := filepath.Join("..", "..", "..", "scenarios", "http1-websocket-tls-performance", "scenarios", "http1", "websocket")
	subprotocolContract, err := os.ReadFile(filepath.Join(root, "rfc6455-tls-subprotocol-text-echo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"subprotocol: plab.echo.v1", "secWebSocketProtocol: plab.echo.v1", "secWebSocketExtensionsPresent: false"} {
		if !strings.Contains(string(subprotocolContract), required) {
			t.Fatalf("subprotocol contract missing %q", required)
		}
	}
	deflateContract, err := os.ReadFile(filepath.Join(root, "rfc6455-tls-permessage-deflate-binary-echo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{perMessageDeflateExtension, "clientNoContextTakeover: true", "serverNoContextTakeover: true", "compressedMessages: data-messages-required", "validation: decompress-and-compare-payload-sha256"} {
		if !strings.Contains(string(deflateContract), required) {
			t.Fatalf("permessage-deflate contract missing %q", required)
		}
	}
}

func TestPerMessageDeflateUsesRSV1AndSemanticRoundTrip(t *testing.T) {
	payload := repeatedByte(66, 1024)
	compressed, err := compressMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(compressed, payload) {
		t.Fatal("compressed wire payload unexpectedly equals semantic payload")
	}
	decompressed, err := decompressMessage(compressed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decompressed, payload) || hash(decompressed) != "9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a" {
		t.Fatal("semantic payload or hash mismatch")
	}
	var wire bytes.Buffer
	if _, err := writeClientFrameWithRSV1(&wire, 0x2, compressed, true); err != nil {
		t.Fatal(err)
	}
	frame, err := readFrame(bufio.NewReader(&wire), true)
	if err != nil {
		t.Fatal(err)
	}
	if frame.rsv != 0x40 || frame.opcode != 0x2 || !frame.masked {
		t.Fatalf("frame=%+v", frame)
	}
}

func TestHandshakeProofFailsClosedOnDiagnosticAndTLSSubstitution(t *testing.T) {
	for _, expectation := range []scenarioExpectation{
		expectations["http1.websocket.rfc6455.tls.subprotocol-text-echo"],
		expectations["http1.websocket.rfc6455.tls.permessage-deflate-binary-echo"],
	} {
		valid := validHandshakeProof(expectation)
		if err := validateHandshakeProof(expectation, valid); err != nil {
			t.Fatalf("valid %s: %v", expectation.id, err)
		}
		mutations := map[string]func(*handshakeProof){
			"wrong subprotocol": func(proof *handshakeProof) { proof.SubprotocolAccepted = "plab.echo.v2" },
			"context takeover":  func(proof *handshakeProof) { proof.ExtensionAccepted = "permessage-deflate" },
			"TLS 1.2":           func(proof *handshakeProof) { proof.TLSVersion = "TLS 1.2" },
			"wrong ALPN":        func(proof *handshakeProof) { proof.ALPN = "h2" },
			"wrong SNI":         func(proof *handshakeProof) { proof.ServerName = "wrong.plab.test" },
			"resumed":           func(proof *handshakeProof) { proof.DidResume = true },
			"early data":        func(proof *handshakeProof) { proof.EarlyData = true },
			"wrong DER":         func(proof *handshakeProof) { proof.CertificateDERSHA256 = strings.Repeat("0", 64) },
			"wrong SPKI":        func(proof *handshakeProof) { proof.CertificateSPKISHA256 = strings.Repeat("0", 64) },
		}
		for name, mutate := range mutations {
			proof := valid
			mutate(&proof)
			if err := validateHandshakeProof(expectation, proof); err == nil {
				t.Fatalf("%s accepted %s", expectation.id, name)
			}
		}
	}
}

func validHandshakeProof(expectation scenarioExpectation) handshakeProof {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2a}, 16))
	extension := expectedExtension(expectation)
	return handshakeProof{
		Binding: "http1-upgrade", Scheme: "wss", TransportSecurity: "tls", Endpoint: "/websocket", Authority: "websocket.plab.test",
		RequestMethod: "GET", RequestedHTTPVersion: "HTTP/1.1", ObservedHTTPVersion: "HTTP/1.1", ResponseStatus: 101,
		UpgradeHeader: "websocket", ConnectionHeader: "Upgrade", SecWebSocketVersion: "13", SampleSecWebSocketKey: key,
		ExpectedSecWebSocketAccept: websocketAccept(key), ObservedSecWebSocketAccept: websocketAccept(key),
		SubprotocolRequested: expectation.subprotocol != "", SubprotocolNegotiated: expectation.subprotocol != "",
		SubprotocolOffered: expectation.subprotocol, SubprotocolAccepted: expectation.subprotocol,
		ExtensionsRequested: expectation.perMessageDeflate, ExtensionsNegotiated: expectation.perMessageDeflate,
		ExtensionOffered: extension, ExtensionAccepted: extension,
		ClientNoContextTakeover: expectation.perMessageDeflate, ServerNoContextTakeover: expectation.perMessageDeflate,
		TLSVersion: "TLS 1.3", ALPN: "http/1.1", ServerName: serverName, CipherSuite: "TLS_AES_128_GCM_SHA256",
		VerifiedChainCount: 1, CertificateDERSHA256: expectedCertificateDERSHA256, CertificateSPKISHA256: expectedCertificateSPKISHA256,
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

func TestNormalizeTargetRequiresTLSAndExactPath(t *testing.T) {
	for _, value := range []string{"http://127.0.0.1:1/websocket", "https://127.0.0.1:1/not-websocket"} {
		if _, err := normalizeTarget(value); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	if got, err := normalizeTarget("https://127.0.0.1:18443"); err != nil || !strings.HasSuffix(got, "/websocket") {
		t.Fatalf("got=%s err=%v", got, err)
	}
}

func TestAuthenticatedRootCertificateHashesMatchExecutorConstants(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "certs", "root.pem"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("root PEM did not contain a certificate")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if got := hash(certificate.Raw); got != expectedCertificateDERSHA256 {
		t.Fatalf("DER hash=%s", got)
	}
	if got := hash(certificate.RawSubjectPublicKeyInfo); got != expectedCertificateSPKISHA256 {
		t.Fatalf("SPKI hash=%s", got)
	}
}
