package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	implementationID      = "go-tls13"
	implementationVersion = "0.1.0"
	alpn                  = "protocol-lab-tls"
	requiredCipherSuite   = tls.TLS_AES_128_GCM_SHA256
	expectedLeafDERHash   = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
)

func main() {
	listen := flag.String("listen", configuredListenAddress(), "TLS listen address")
	certPath := flag.String("cert", envOrDefault("PLAB_TLS_CERT_FILE", materialPath("certs/leaf.pem")), "server certificate")
	keyPath := flag.String("key", envOrDefault("PLAB_TLS_KEY_FILE", materialPath("certs/leaf-key.pem")), "server private key")
	flag.Parse()

	certificate, err := tls.LoadX509KeyPair(*certPath, *keyPath)
	if err != nil {
		fatal(err)
	}
	if len(certificate.Certificate) == 0 || hash(certificate.Certificate[0]) != expectedLeafDERHash {
		fatal(errors.New("TLS certificate substitution detected"))
	}
	config := &tls.Config{
		Certificates:     []tls.Certificate{certificate},
		MinVersion:       tls.VersionTLS13,
		MaxVersion:       tls.VersionTLS13,
		NextProtos:       []string{alpn},
		CurvePreferences: []tls.CurveID{tls.X25519},
	}
	listener, err := tls.Listen("tcp", *listen, config)
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

func handle(connection net.Conn) {
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
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
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != alpn || state.CipherSuite != requiredCipherSuite {
		fmt.Fprintf(os.Stderr, "unexpected negotiation: version=%x alpn=%q cipher=%s\n", state.Version, state.NegotiatedProtocol, tls.CipherSuiteName(state.CipherSuite))
		return
	}
	// Closing after the handshake lets crypto/tls send its session ticket and
	// close alert without accepting application bytes. The executor proves the
	// exact zero-byte and resumed/full session state for each operation.
}

func writeReady(listen string) {
	ready := map[string]any{
		"eventName":             "ready",
		"implementationId":      implementationID,
		"implementationVersion": implementationVersion,
		"listenAddress":         listen,
		"protocol":              "tls",
		"tlsVersion":            "TLS1.3",
		"alpn":                  alpn,
		"cipherSuite":           tls.CipherSuiteName(requiredCipherSuite),
		"keyExchangeGroup":      "X25519",
		"certificateProfileId":  "plab-single-leaf-p256-v1",
		"certificateDerSha256":  expectedLeafDERHash,
		"sessionTicketsEnabled": true,
		"supportedScenarios":    []string{"tls.handshake.full", "tls.handshake.resumed"},
	}
	encoded, _ := json.Marshal(ready)
	fmt.Println(string(encoded))
}

func configuredListenAddress() string {
	if explicit := strings.TrimSpace(os.Getenv("PLAB_LISTEN_ADDRESS")); explicit != "" {
		return explicit
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

func hash(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
