package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"flag"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/quic-go/quic-go/http3"
)

const (
	implementationID = "quic-go-http3"
	packageID        = "org.protocol-lab.components.implementation.quic-go-http3"
)

var quicGoVersion = "v0.60.0"

func main() {
	listen := flag.String("listen", envOrDefault("PLAB_LISTEN", ":4433"), "UDP listen address")
	flag.Parse()

	server := http3.Server{
		Addr:      *listen,
		Handler:   routes(),
		TLSConfig: mustTLSConfig(),
	}

	log.Printf("quic-go HTTP/3 server listening on %s", *listen)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func routes() http.Handler {
	mux := http.NewServeMux()
	bytes1KB := strings.Repeat("x", 1024)
	bytes64KB := strings.Repeat("x", 65_536)
	bytes1MB := strings.Repeat("x", 1_048_576)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":           "ok",
			"implementationId": implementationID,
			"protocol":         "h3",
		})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"protocol":       "h3",
			"server":         "quic-go",
			"implementation": implementationID,
			"utc":            time.Now().UTC().Format(time.RFC3339Nano),
			"processId":      os.Getpid(),
		})
	})
	mux.HandleFunc("/protocol-lab/metadata", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"implementationId": implementationID,
			"packageId":        packageID,
			"protocol":         "h3",
			"protocolVersion":  "http/3",
			"quicGoVersion":    quicGoVersion,
			"supportedScenarios": []string{
				"http3.core.status",
				"http3.payload.bytes.1kb",
				"http3.payload.bytes.64kb",
				"http3.payload.bytes.1mb",
			},
			"unsupportedKnownCases": []string{"h1", "h2", "h2c", "raw-quic", "websocket", "server-sent-events"},
		})
	})
	mux.HandleFunc("/plaintext", textHandler("Hello, World!", "text/plain"))
	mux.HandleFunc("/json", textHandler(`{"message":"Hello, World!"}`, "application/json"))
	mux.HandleFunc("/bytes/1024", textHandler(bytes1KB, "application/octet-stream"))
	mux.HandleFunc("/bytes/1kb", textHandler(bytes1KB, "application/octet-stream"))
	mux.HandleFunc("/bytes/65536", textHandler(bytes64KB, "application/octet-stream"))
	mux.HandleFunc("/bytes/64kb", textHandler(bytes64KB, "application/octet-stream"))
	mux.HandleFunc("/bytes/1048576", textHandler(bytes1MB, "application/octet-stream"))
	mux.HandleFunc("/bytes/1mb", textHandler(bytes1MB, "application/octet-stream"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	return mux
}

func textHandler(body string, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", strconvLen(body))
		_, _ = w.Write([]byte(body))
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func mustTLSConfig() *tls.Config {
	cert, err := generateCertificate()
	if err != nil {
		log.Fatalf("failed to generate TLS certificate: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{http3.NextProtoH3},
		MinVersion:   tls.VersionTLS13,
	}
}

func generateCertificate() (tls.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "ProtocolLab quic-go HTTP/3 local certificate",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "host.docker.internal"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return tls.X509KeyPair(certPEM, keyPEM)
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}

func strconvLen(value string) string {
	return strconv.Itoa(len(value))
}
