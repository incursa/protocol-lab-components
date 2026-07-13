package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateStateAcceptsExactTLS12Profile(t *testing.T) {
	state := exactState(t)
	if _, err := validateState(state); err != nil {
		t.Fatal(err)
	}
}
func TestValidateStateRejectsDowngrade(t *testing.T) {
	state := exactState(t)
	state.Version = tls.VersionTLS11
	if _, err := validateState(state); err == nil {
		t.Fatal("downgraded TLS version was accepted")
	}
}
func TestValidateStateRejectsCipherSubstitution(t *testing.T) {
	state := exactState(t)
	state.CipherSuite = tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
	if _, err := validateState(state); err == nil {
		t.Fatal("wrong cipher was accepted")
	}
}
func TestValidateStateRejectsALPNSubstitution(t *testing.T) {
	state := exactState(t)
	state.NegotiatedProtocol = "wrong"
	if _, err := validateState(state); err == nil {
		t.Fatal("wrong ALPN was accepted")
	}
}
func TestValidateStateRejectsCertificateSubstitution(t *testing.T) {
	state := exactState(t)
	wrong := *state.PeerCertificates[0]
	wrong.Raw = []byte("substituted")
	state.PeerCertificates = []*x509.Certificate{&wrong}
	if _, err := validateState(state); err == nil {
		t.Fatal("wrong certificate was accepted")
	}
}
func TestValidateStateRejectsResumedSession(t *testing.T) {
	state := exactState(t)
	state.DidResume = true
	if _, err := validateState(state); err == nil {
		t.Fatal("resumed TLS 1.2 session was accepted")
	}
}

func TestAllOtherTLSIdentitiesAreKnownUnsupported(t *testing.T) {
	expected := []string{"tls.handshake.full", "tls.handshake.resumed", "tls.handshake.full.chacha20", "tls.handshake.mutual-auth", "tls.early-data.accepted", "tls.early-data.rejected", "tls.key-update.diagnostic", "tls.record.throughput", "tls.record.coverage"}
	for _, id := range expected {
		if !isKnownUnsupported(id) {
			t.Errorf("%s was not recognized as unsupported", id)
		}
	}
	if isKnownUnsupported(scenarioID) {
		t.Fatal("supported TLS 1.2 identity was marked unsupported")
	}
}

func exactState(t *testing.T) tls.ConnectionState {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean("../../../implementations/go-tls12/certs/leaf.pem"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("server certificate PEM did not decode")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return tls.ConnectionState{Version: tls.VersionTLS12, HandshakeComplete: true, CipherSuite: tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, CurveID: tls.X25519, NegotiatedProtocol: alpn, PeerCertificates: []*x509.Certificate{leaf}, VerifiedChains: [][]*x509.Certificate{{leaf}}}
}
