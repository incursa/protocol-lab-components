package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

func TestRFC6455AcceptExample(t *testing.T) {
	if got := websocketAccept("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("accept=%q", got)
	}
}

func TestValidateUpgradeRequest(t *testing.T) {
	request := exactRequest(t, "", "")
	profile, err := validateUpgradeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if profile.subprotocol != "" || profile.perMessageDeflate {
		t.Fatalf("profile=%+v", profile)
	}
	request = exactRequest(t, "", "permessage-deflate")
	if _, err := validateUpgradeRequest(request); err == nil {
		t.Fatal("extension substitution accepted")
	}
}

func TestValidateExactDiagnosticUpgradeProfiles(t *testing.T) {
	request := exactRequest(t, subprotocol, "")
	profile, err := validateUpgradeRequest(request)
	if err != nil || profile.subprotocol != subprotocol || profile.perMessageDeflate {
		t.Fatalf("subprotocol profile=%+v err=%v", profile, err)
	}
	request = exactRequest(t, "", perMessageDeflateExtension)
	profile, err = validateUpgradeRequest(request)
	if err != nil || profile.subprotocol != "" || !profile.perMessageDeflate {
		t.Fatalf("deflate profile=%+v err=%v", profile, err)
	}
	for _, invalid := range [][2]string{
		{"plab.echo.v2", ""},
		{subprotocol, perMessageDeflateExtension},
		{"", "permessage-deflate; server_no_context_takeover"},
		{"", perMessageDeflateExtension + "; client_max_window_bits=15"},
	} {
		if _, err := validateUpgradeRequest(exactRequest(t, invalid[0], invalid[1])); err == nil {
			t.Fatalf("accepted adjacent profile subprotocol=%q extension=%q", invalid[0], invalid[1])
		}
	}
}

func TestPerMessageDeflateSemanticRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte{66}, 1024)
	compressed, err := compressMessage(payload)
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := decompressMessage(compressed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decompressed, payload) {
		t.Fatal("decompressed semantic payload mismatch")
	}
	var wire bytes.Buffer
	if err := writeFrameWithRSV1(&wire, 0x2, compressed, false, true); err != nil {
		t.Fatal(err)
	}
	frame, err := readFrame(bufio.NewReader(&wire), false)
	if err != nil {
		t.Fatal(err)
	}
	if frame.rsv != 0x40 || frame.opcode != 0x2 || frame.masked {
		t.Fatalf("frame=%+v", frame)
	}
}

func exactRequest(t *testing.T, offeredSubprotocol, offeredExtension string) *http.Request {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 16))
	raw := "GET /websocket HTTP/1.1\r\nHost: websocket.plab.test\r\nUpgrade: websocket\r\nConnection: keep-alive, Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: " + key + "\r\n"
	if offeredSubprotocol != "" {
		raw += "Sec-WebSocket-Protocol: " + offeredSubprotocol + "\r\n"
	}
	if offeredExtension != "" {
		raw += "Sec-WebSocket-Extensions: " + offeredExtension + "\r\n"
	}
	raw += "\r\n"
	request, err := http.ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func TestReadFrameRequiresClientMask(t *testing.T) {
	_, err := readFrame(bufio.NewReader(bytes.NewReader([]byte{0x81, 0x01, 'x'})), true)
	if err == nil || !strings.Contains(err.Error(), "masking") {
		t.Fatalf("err=%v", err)
	}
}
