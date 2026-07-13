package main

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func canonicalFrame() []byte {
	protobuf := append([]byte{0x0a, 0x80, 0x01}, expectedPayload...)
	return append([]byte{0, 0, 0, 0, byte(len(protobuf))}, protobuf...)
}

func TestHandlerReturnsCanonicalUnaryEcho(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://grpc.plab.test"+grpcPath, bytes.NewReader(canonicalFrame()))
	req.Proto, req.ProtoMajor, req.ProtoMinor = "HTTP/2.0", 2, 0
	req.TLS = &tls.ConnectionState{Version: tls.VersionTLS13, NegotiatedProtocol: "h2"}
	req.Header.Set("Content-Type", "application/grpc+proto")
	req.Header.Set("Te", "trailers")
	response := httptest.NewRecorder()
	handleUnaryEcho(response, req)
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), canonicalFrame()) {
		t.Fatalf("unexpected unary echo response: status=%d bytes=%d", response.Code, response.Body.Len())
	}
}

func TestValidateFrame(t *testing.T) {
	if err := validateFrame(canonicalFrame()); err != nil {
		t.Fatalf("canonical frame rejected: %v", err)
	}
	mutated := canonicalFrame()
	mutated[len(mutated)-1] = 'X'
	if err := validateFrame(mutated); err == nil {
		t.Fatal("mutated payload was accepted")
	}
}

func TestHandlerRejectsProtocolFallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://grpc.plab.test"+grpcPath, bytes.NewReader(canonicalFrame()))
	req.Header.Set("Content-Type", "application/grpc+proto")
	req.Header.Set("Te", "trailers")
	response := httptest.NewRecorder()
	handleUnaryEcho(response, req)
	if response.Code != http.StatusHTTPVersionNotSupported {
		t.Fatalf("expected protocol rejection, got %d", response.Code)
	}
}
