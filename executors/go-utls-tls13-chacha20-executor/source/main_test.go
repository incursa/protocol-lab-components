package main

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	utls "github.com/refraction-networking/utls"
)

func TestClientHelloAcceptsExactChaCha20Profile(t *testing.T) {
	hello := exactHello()
	proof, err := validateClientHello(hello)
	if err != nil {
		t.Fatal(err)
	}
	if len(proof.CipherSuites) != 1 || proof.CipherSuites[0] != "TLS_CHACHA20_POLY1305_SHA256" {
		t.Fatalf("unexpected proof: %#v", proof)
	}
}

func TestClientHelloRejectsAESSubstitution(t *testing.T) {
	hello := exactHello()
	hello.CipherSuites = []uint16{utls.TLS_AES_128_GCM_SHA256}
	if _, err := validateClientHello(hello); err == nil {
		t.Fatal("AES ClientHello substitution accepted")
	}
}

func TestClientHelloRejectsSessionStateAndEarlyData(t *testing.T) {
	hello := exactHello()
	hello.TicketSupported = true
	hello.PskIdentities = []utls.PskIdentity{{}}
	hello.EarlyData = true
	if _, err := validateClientHello(hello); err == nil {
		t.Fatal("resumption or early-data offer accepted")
	}
}

func TestValidateStateAcceptsExactProfile(t *testing.T) {
	if _, err := validateState(exactState(t), utls.X25519, false); err != nil {
		t.Fatal(err)
	}
}
func TestValidateStateRejectsAESSubstitution(t *testing.T) {
	state := exactState(t)
	state.CipherSuite = utls.TLS_AES_128_GCM_SHA256
	if _, err := validateState(state, utls.X25519, false); err == nil {
		t.Fatal("AES substitution accepted")
	}
}
func TestValidateStateRejectsVersionSubstitution(t *testing.T) {
	state := exactState(t)
	state.Version = utls.VersionTLS12
	if _, err := validateState(state, utls.X25519, false); err == nil {
		t.Fatal("version substitution accepted")
	}
}
func TestValidateStateRejectsGroupSubstitution(t *testing.T) {
	if _, err := validateState(exactState(t), utls.CurveP256, false); err == nil {
		t.Fatal("group substitution accepted")
	}
}
func TestValidateStateRejectsALPNSubstitution(t *testing.T) {
	state := exactState(t)
	state.NegotiatedProtocol = "wrong"
	if _, err := validateState(state, utls.X25519, false); err == nil {
		t.Fatal("ALPN substitution accepted")
	}
}
func TestValidateStateRejectsCertificateSubstitution(t *testing.T) {
	state := exactState(t)
	wrong := *state.PeerCertificates[0]
	wrong.Raw = []byte("wrong")
	state.PeerCertificates = []*x509.Certificate{&wrong}
	if _, err := validateState(state, utls.X25519, false); err == nil {
		t.Fatal("certificate substitution accepted")
	}
}
func TestValidateStateRejectsResumedSession(t *testing.T) {
	state := exactState(t)
	state.DidResume = true
	if _, err := validateState(state, utls.X25519, true); err == nil {
		t.Fatal("resumed session accepted")
	}
}
func TestOtherTLSIdentitiesUnsupported(t *testing.T) {
	expected := []string{"tls.handshake.full", "tls.handshake.resumed", "tls.handshake.full.tls12", "tls.handshake.mutual-auth", "tls.early-data.accepted", "tls.early-data.rejected", "tls.key-update.diagnostic", "tls.record.throughput", "tls.record.coverage"}
	for _, id := range expected {
		if !isKnownUnsupported(id) {
			t.Errorf("%s not recognized unsupported", id)
		}
	}
}

func exactHello() *utls.PubClientHelloMsg {
	return &utls.PubClientHelloMsg{
		CipherSuites: []uint16{utls.TLS_CHACHA20_POLY1305_SHA256}, CompressionMethods: []uint8{0}, ServerName: serverName,
		SupportedCurves: []utls.CurveID{utls.X25519}, SupportedPoints: []uint8{0}, SupportedSignatureAlgorithms: []utls.SignatureScheme{utls.ECDSAWithP256AndSHA256},
		AlpnProtocols: []string{alpn}, SupportedVersions: []uint16{utls.VersionTLS13}, KeyShares: []utls.KeyShare{{Group: utls.X25519}},
	}
}

func exactState(t *testing.T) utls.ConnectionState {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean("../../../implementations/go-tls13-chacha20/certs/leaf.pem"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("certificate PEM did not decode")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return utls.ConnectionState{Version: utls.VersionTLS13, HandshakeComplete: true, CipherSuite: utls.TLS_CHACHA20_POLY1305_SHA256, NegotiatedProtocol: alpn, PeerCertificates: []*x509.Certificate{leaf}, VerifiedChains: [][]*x509.Certificate{{leaf}}}
}
