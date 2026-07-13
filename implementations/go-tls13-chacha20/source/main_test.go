package main

import (
	"crypto/tls"
	"path/filepath"
	"testing"
)

func TestValidateStateAcceptsExactChaCha20(t *testing.T) {
	if err := validateState(exactState()); err != nil {
		t.Fatal(err)
	}
}
func TestValidateStateRejectsAESSubstitution(t *testing.T) {
	state := exactState()
	state.CipherSuite = tls.TLS_AES_128_GCM_SHA256
	if err := validateState(state); err == nil {
		t.Fatal("AES substitution was accepted")
	}
}
func TestValidateStateRejectsVersionSubstitution(t *testing.T) {
	state := exactState()
	state.Version = tls.VersionTLS12
	if err := validateState(state); err == nil {
		t.Fatal("TLS version substitution was accepted")
	}
}
func TestValidateStateRejectsALPNSubstitution(t *testing.T) {
	state := exactState()
	state.NegotiatedProtocol = "wrong"
	if err := validateState(state); err == nil {
		t.Fatal("ALPN substitution was accepted")
	}
}
func TestValidateStateRejectsResumedSession(t *testing.T) {
	state := exactState()
	state.DidResume = true
	if err := validateState(state); err == nil {
		t.Fatal("resumed session was accepted")
	}
}
func TestRejectsServerCertificateSubstitution(t *testing.T) {
	_, err := loadServerIdentity(filepath.Clean("../../../executors/go-tls13-mtls-executor/certs/client.pem"), filepath.Clean("../../../executors/go-tls13-mtls-executor/certs/client-key.pem"))
	if err == nil {
		t.Fatal("valid but non-canonical certificate identity was accepted")
	}
}
func exactState() tls.ConnectionState {
	return tls.ConnectionState{Version: tls.VersionTLS13, HandshakeComplete: true, CipherSuite: tls.TLS_CHACHA20_POLY1305_SHA256, CurveID: tls.X25519, NegotiatedProtocol: alpn}
}
