package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"os"
	"testing"
)

func TestCanonicalFixtureResponse(t *testing.T) {
	if len(canonicalResponse) != 43 {
		t.Fatalf("response length = %d", len(canonicalResponse))
	}
	if got := sha256Hex(canonicalResponse); got != "9d488461675ad5ab9f74c7b203861e1ad17521e413a407a25e6611012a595620" {
		t.Fatalf("response hash = %s", got)
	}
	if binary.BigEndian.Uint16(canonicalResponse[:2]) != 0 {
		t.Fatal("canonical response message ID must be zero")
	}
}

func TestPackagedLeafCertificateIdentity(t *testing.T) {
	data, err := os.ReadFile("../certs/leaf.pem")
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("leaf PEM did not contain a certificate")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256Hex(certificate.Raw); got != "b57bdd3eb90b36455900c17de9ff9a02c623e1f6b27626ad7821a40e35e8251c" {
		t.Fatalf("leaf DER hash = %s", got)
	}
	if err := certificate.VerifyHostname("dns.plab.test"); err != nil {
		t.Fatal(err)
	}
	key, ok := certificate.PublicKey.(*ecdsa.PublicKey)
	if !ok || key.Curve.Params().Name != "P-256" {
		t.Fatalf("leaf key = %T", certificate.PublicKey)
	}
	if certificate.SignatureAlgorithm != x509.ECDSAWithSHA256 {
		t.Fatalf("leaf signature = %s", certificate.SignatureAlgorithm)
	}
}

func TestCanonicalFixtureSemantics(t *testing.T) {
	if canonicalResponse[2] != 0x84 || canonicalResponse[3] != 0 {
		t.Fatal("response must set QR+AA and clear RA with NOERROR")
	}
	if got := binary.BigEndian.Uint32(canonicalResponse[len(canonicalResponse)-10 : len(canonicalResponse)-6]); got != 0 {
		t.Fatalf("TTL = %d", got)
	}
	expected := []byte{192, 0, 2, 1}
	actual := canonicalResponse[len(canonicalResponse)-4:]
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("A answer = %v", actual)
		}
	}
}
