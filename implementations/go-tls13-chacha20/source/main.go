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
	implementationID      = "go-tls13-chacha20"
	implementationVersion = "0.1.0"
	scenarioID            = "tls.handshake.full.chacha20"
	profileID             = "plab-tls13-chacha20-p256-server-auth-v2"
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
	return &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, CurvePreferences: []tls.CurveID{tls.X25519}, NextProtos: []string{alpn}, SessionTicketsDisabled: true}
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

func validateState(state tls.ConnectionState) error {
	var failures []string
	if state.Version != tls.VersionTLS13 || !state.HandshakeComplete {
		failures = append(failures, "exact TLS 1.3 handshake not complete")
	}
	if state.CipherSuite != tls.TLS_CHACHA20_POLY1305_SHA256 {
		failures = append(failures, "ChaCha20 cipher suite mismatch")
	}
	if state.CurveID != tls.X25519 {
		failures = append(failures, "key exchange group mismatch")
	}
	if state.NegotiatedProtocol != alpn {
		failures = append(failures, "ALPN mismatch")
	}
	if state.DidResume {
		failures = append(failures, "session resumption detected")
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
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
	if err := validateState(tlsConnection.ConnectionState()); err != nil {
		fmt.Fprintln(os.Stderr, err)
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
	ready := map[string]any{"eventName": "ready", "implementationId": implementationID, "implementationVersion": implementationVersion, "listenAddress": listen, "protocol": "tls", "tlsVersion": "TLS1.3", "alpn": alpn, "cipherSuite": tls.CipherSuiteName(tls.TLS_CHACHA20_POLY1305_SHA256), "keyExchangeGroup": "X25519", "tlsProfileId": profileID, "certificateProfileId": certificateProfileID, "certificateDerSha256": expectedLeafDERHash, "sessionTicketsEnabled": false, "supportedScenarios": []string{scenarioID}}
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
