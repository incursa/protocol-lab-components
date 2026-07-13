package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	implementationID      = "go-tls12"
	implementationVersion = "0.1.0"
	scenarioID            = "tls.handshake.full.tls12"
	profileID             = "plab-tls12-aes128gcm-p256-server-auth-v2"
	certificateProfileID  = "plab-single-leaf-p256-server-v2"
	alpn                  = "protocol-lab-tls"
	expectedLeafDERHash   = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
)

func main() {
	listen := flag.String("listen", configuredListenAddress(), "TLS listen address")
	certPath := flag.String("cert", envOrDefault("PLAB_TLS_CERT_FILE", materialPath("certs/leaf.pem")), "server certificate")
	keyPath := flag.String("key", envOrDefault("PLAB_TLS_KEY_FILE", materialPath("certs/leaf-key.pem")), "server private key")
	flag.Parse()
	if requested := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")); requested != "" && requested != scenarioID {
		fatal(fmt.Errorf("scenario %q is explicitly unsupported by %s", requested, implementationID))
	}
	certificate, err := loadServerIdentity(*certPath, *keyPath)
	if err != nil {
		fatal(err)
	}
	listener, err := tls.Listen("tcp", *listen, serverConfig(certificate))
	if err != nil {
		fatal(err)
	}
	defer listener.Close()
	writeReady(*listen)
	for {
		connection, err := listener.Accept()
		if err != nil {
			fatal(err)
		}
		go handle(connection)
	}
}

func serverConfig(certificate tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates:           []tls.Certificate{certificate},
		MinVersion:             tls.VersionTLS12,
		MaxVersion:             tls.VersionTLS12,
		CipherSuites:           []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
		CurvePreferences:       []tls.CurveID{tls.X25519},
		NextProtos:             []string{alpn},
		SessionTicketsDisabled: true,
	}
}

func loadServerIdentity(certPath, keyPath string) (tls.Certificate, error) {
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	if len(certificate.Certificate) != 1 || hash(certificate.Certificate[0]) != expectedLeafDERHash {
		return tls.Certificate{}, errors.New("server certificate substitution or chain expansion detected")
	}
	leaf, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		return tls.Certificate{}, err
	}
	key, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if leaf.SignatureAlgorithm != x509.ECDSAWithSHA256 || leaf.PublicKeyAlgorithm != x509.ECDSA || !ok || key.Curve.Params().Name != "P-256" {
		return tls.Certificate{}, errors.New("server certificate algorithm or curve mismatch")
	}
	return certificate, nil
}

func handle(connection net.Conn) {
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(10 * time.Second))
	tlsConnection, ok := connection.(*tls.Conn)
	if !ok {
		fmt.Fprintln(os.Stderr, "accepted connection was not TLS")
		return
	}
	if err := tlsConnection.Handshake(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	state := tlsConnection.ConnectionState()
	if state.Version != tls.VersionTLS12 || state.CipherSuite != tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 || state.CurveID != tls.X25519 || state.NegotiatedProtocol != alpn || state.DidResume {
		fmt.Fprintln(os.Stderr, "negotiated TLS state did not match the exact TLS 1.2 profile")
		return
	}
	one := make([]byte, 1)
	n, err := tlsConnection.Read(one)
	if n != 0 {
		fmt.Fprintln(os.Stderr, "application data is forbidden for the handshake-only scenario")
		return
	}
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(os.Stderr, err)
	}
}

func writeReady(listen string) {
	ready := map[string]any{
		"eventName": "ready", "implementationId": implementationID, "implementationVersion": implementationVersion,
		"listenAddress": listen, "protocol": "tls", "tlsVersion": "TLS1.2", "alpn": alpn,
		"cipherSuite": tls.CipherSuiteName(tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256), "keyExchangeGroup": "X25519",
		"tlsProfileId": profileID, "certificateProfileId": certificateProfileID, "certificateDerSha256": expectedLeafDERHash,
		"sessionTicketsEnabled": false, "supportedScenarios": []string{scenarioID},
	}
	encoded, _ := json.Marshal(ready)
	fmt.Println(string(encoded))
}

func configuredListenAddress() string {
	if value := strings.TrimSpace(os.Getenv("PLAB_LISTEN_ADDRESS")); value != "" {
		return value
	}
	if port := strings.TrimSpace(os.Getenv("PLAB_TARGET_PORT")); port != "" {
		return "127.0.0.1:" + port
	}
	return "127.0.0.1:18443"
}
func materialPath(relative string) string {
	if executable, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(executable), "..", "..", relative)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", relative)
}
func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func fatal(err error)          { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
