package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMutualAuthAcceptsExactTrustedClientLeaf(t *testing.T) {
	serverCertificate := mustKeyPair(t, "../certs/leaf.pem", "../certs/leaf-key.pem")
	clientCertificate, clientRoots := generatedClientIdentity(t, "client.plab.test")
	leaf, err := x509.ParseCertificate(clientCertificate.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	serverErr, clientErr := handshakePair(serverConfig(serverCertificate, clientRoots, hash(leaf.Raw), hash(leaf.RawSubjectPublicKeyInfo)), clientCertificate)
	if serverErr != nil || clientErr != nil {
		t.Fatalf("canonical mutual authentication failed: server=%v client=%v", serverErr, clientErr)
	}
}

func TestMutualAuthRejectsAbsentClientCertificate(t *testing.T) {
	serverCertificate := mustKeyPair(t, "../certs/leaf.pem", "../certs/leaf-key.pem")
	serverErr, _ := handshakePair(serverConfig(serverCertificate, mustRoots(t, "../certs/client-root.pem"), clientLeafDERHash, clientLeafSPKIHash), tls.Certificate{})
	if serverErr == nil {
		t.Fatal("server accepted a connection with no client certificate")
	}
}

func TestMutualAuthRejectsTrustedButWrongClientCertificate(t *testing.T) {
	serverCertificate := mustKeyPair(t, "../certs/leaf.pem", "../certs/leaf-key.pem")
	wrong, trustedRoot := generatedClientIdentity(t, "wrong.plab.test")
	serverErr, _ := handshakePair(serverConfig(serverCertificate, trustedRoot, clientLeafDERHash, clientLeafSPKIHash), wrong)
	if serverErr == nil || !strings.Contains(serverErr.Error(), "client certificate identity mismatch") {
		t.Fatalf("expected exact identity rejection, got %v", serverErr)
	}
}

func TestMutualAuthRejectsUntrustedClientCertificate(t *testing.T) {
	serverCertificate := mustKeyPair(t, "../certs/leaf.pem", "../certs/leaf-key.pem")
	untrusted, _ := generatedClientIdentity(t, "untrusted.plab.test")
	serverErr, _ := handshakePair(serverConfig(serverCertificate, mustRoots(t, "../certs/client-root.pem"), clientLeafDERHash, clientLeafSPKIHash), untrusted)
	if serverErr == nil {
		t.Fatal("server accepted an untrusted client certificate")
	}
}

func handshakePair(serverConfig *tls.Config, clientCertificate tls.Certificate) (error, error) {
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()
	server := tls.Server(serverSide, serverConfig)
	clientConfig := &tls.Config{
		MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, InsecureSkipVerify: true,
		NextProtos: []string{alpn}, CurvePreferences: []tls.CurveID{tls.X25519}, SessionTicketsDisabled: true,
	}
	if len(clientCertificate.Certificate) > 0 {
		clientConfig.Certificates = []tls.Certificate{clientCertificate}
	}
	client := tls.Client(clientSide, clientConfig)
	_ = server.SetDeadline(time.Now().Add(time.Second))
	_ = client.SetDeadline(time.Now().Add(time.Second))
	serverResult := make(chan error, 1)
	go func() { serverResult <- server.Handshake() }()
	clientErr := client.Handshake()
	serverErr := <-serverResult
	return serverErr, clientErr
}

func mustKeyPair(t *testing.T, certificatePath, keyPath string) tls.Certificate {
	t.Helper()
	certificate, err := tls.LoadX509KeyPair(filepath.Clean(certificatePath), filepath.Clean(keyPath))
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}

func mustRoots(t *testing.T, path string) *x509.CertPool {
	t.Helper()
	roots, err := loadRoots(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	return roots
}

func generatedClientIdentity(t *testing.T, commonName string) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rootTemplate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test root"}, NotBefore: now, NotAfter: now.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: commonName}, DNSNames: []string{commonName}, NotBefore: now, NotAfter: now.Add(24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	rootCertificate, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, rootCertificate, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(rootCertificate)
	return pair, roots
}
