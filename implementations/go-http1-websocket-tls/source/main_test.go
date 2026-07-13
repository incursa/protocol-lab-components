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
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 16))
	raw := "GET /websocket HTTP/1.1\r\nHost: websocket.plab.test\r\nUpgrade: websocket\r\nConnection: keep-alive, Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: " + key + "\r\n\r\n"
	request, err := http.ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateUpgradeRequest(request); err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Sec-WebSocket-Extensions", "permessage-deflate")
	if err := validateUpgradeRequest(request); err == nil {
		t.Fatal("extension substitution accepted")
	}
}

func TestReadFrameRequiresClientMask(t *testing.T) {
	_, err := readFrame(bufio.NewReader(bytes.NewReader([]byte{0x81, 0x01, 'x'})), true)
	if err == nil || !strings.Contains(err.Error(), "masking") {
		t.Fatalf("err=%v", err)
	}
}
