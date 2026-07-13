package main

import (
	"crypto/tls"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExactTLS12Handshake(t *testing.T) {
	certificate := mustServerIdentity(t)
	serverErr, clientErr := handshakePair(serverConfig(certificate), exactClientConfig())
	if serverErr != nil || clientErr != nil {
		t.Fatalf("exact TLS 1.2 handshake failed: server=%v client=%v", serverErr, clientErr)
	}
}

func TestRejectsWrongCipher(t *testing.T) {
	config := exactClientConfig()
	config.CipherSuites = []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384}
	serverErr, _ := handshakePair(serverConfig(mustServerIdentity(t)), config)
	if serverErr == nil {
		t.Fatal("wrong cipher suite was accepted")
	}
}

func TestRejectsWrongALPN(t *testing.T) {
	config := exactClientConfig()
	config.NextProtos = []string{"wrong-alpn"}
	serverErr, _ := handshakePair(serverConfig(mustServerIdentity(t)), config)
	if serverErr == nil {
		t.Fatal("wrong ALPN was accepted")
	}
}

func TestRejectsServerCertificateSubstitution(t *testing.T) {
	_, err := loadServerIdentity(filepath.Clean("../../../executors/go-tls13-mtls-executor/certs/client.pem"), filepath.Clean("../../../executors/go-tls13-mtls-executor/certs/client-key.pem"))
	if err == nil {
		t.Fatal("valid but non-canonical certificate identity was accepted")
	}
}

func exactClientConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12, InsecureSkipVerify: true, CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256}, CurvePreferences: []tls.CurveID{tls.X25519}, NextProtos: []string{alpn}, SessionTicketsDisabled: true}
}
func mustServerIdentity(t *testing.T) tls.Certificate {
	t.Helper()
	certificate, err := loadServerIdentity(filepath.Clean("../certs/leaf.pem"), filepath.Clean("../certs/leaf-key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}
func handshakePair(serverConfig, clientConfig *tls.Config) (error, error) {
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()
	server := tls.Server(serverSide, serverConfig)
	client := tls.Client(clientSide, clientConfig)
	_ = server.SetDeadline(time.Now().Add(time.Second))
	_ = client.SetDeadline(time.Now().Add(time.Second))
	result := make(chan error, 1)
	go func() { result <- server.Handshake() }()
	clientErr := client.Handshake()
	serverErr := <-result
	return serverErr, clientErr
}

func TestReadyProfileIdentity(t *testing.T) {
	if !strings.Contains(profileID, "tls12-aes128gcm") || certificateProfileID != "plab-single-leaf-p256-server-v2" {
		t.Fatal("profile identities drifted")
	}
}
