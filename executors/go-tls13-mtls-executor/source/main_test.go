package main

import (
	"crypto/tls"
	"crypto/x509"
	"path/filepath"
	"testing"
)

func TestPackagedClientIdentityIsExactSingleLeaf(t *testing.T) {
	certificate, err := loadClientIdentity(filepath.Clean("../certs/client.pem"), filepath.Clean("../certs/client-key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	if len(certificate.Certificate) != 1 {
		t.Fatalf("expected one sent leaf, got %d", len(certificate.Certificate))
	}
	leaf, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if hash(leaf.Raw) != clientLeafDERHash || hash(leaf.RawSubjectPublicKeyInfo) != clientLeafSPKIHash {
		t.Fatal("packaged client certificate does not match canonical hashes")
	}
}

func TestValidateStateRejectsFallbackAndResumption(t *testing.T) {
	certificate, err := loadClientIdentity(filepath.Clean("../certs/client.pem"), filepath.Clean("../certs/client-key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	state := tls.ConnectionState{Version: tls.VersionTLS12, HandshakeComplete: true, DidResume: true, CipherSuite: tls.TLS_AES_128_GCM_SHA256, CurveID: tls.X25519, NegotiatedProtocol: alpn}
	if _, err := validateState(state, certificate); err == nil {
		t.Fatal("fallback and resumed state was accepted")
	}
}

func TestAllOtherCommittedTLSIdentitiesAreKnownUnsupported(t *testing.T) {
	expected := []string{"tls.handshake.full", "tls.handshake.resumed", "tls.handshake.full.tls12", "tls.handshake.full.chacha20", "tls.early-data.accepted", "tls.early-data.rejected", "tls.key-update.diagnostic", "tls.record.throughput", "tls.record.coverage"}
	for _, id := range expected {
		if !isKnownUnsupported(id) {
			t.Errorf("%s was not recognized as unsupported", id)
		}
	}
	if isKnownUnsupported(scenarioID) {
		t.Fatal("supported mutual-auth scenario was marked unsupported")
	}
}
