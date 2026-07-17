package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/quic-go/quic-go/http3"
)

const (
	implementationID = "quic-go-http3"
	packageID        = "org.protocol-lab.components.implementation.quic-go-http3"

	streamBytesPath          = "/stream/bytes"
	streamBytesCanonicalURL  = "chunks=100&size=16384&delayMs=0"
	streamBytesCanonicalRows = 100
	streamBytesChunkSize     = 16 * 1024
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
				"http3.core.plaintext",
				"http3.core.json",
				"http3.core.status",
				"http3.payload.bytes.1kb",
				"http3.payload.bytes.64kb",
				"http3.payload.bytes.1mb",
				"http3.payload.stream.100x16kb",
				"http3.headers.response-headers-50x32",
				"http3.protocol.qpack-repeated-headers",
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
	mux.HandleFunc("/headers/response", responseHeadersHandler)
	mux.HandleFunc(streamBytesPath, streamBytesHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	return mux
}

func responseHeadersHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("count") != "50" || r.URL.Query().Get("size") != "32" {
		http.Error(w, "expected count=50 and size=32", http.StatusBadRequest)
		return
	}

	value := strings.Repeat("h", 32)
	for index := 0; index < 50; index++ {
		w.Header().Set(fmt.Sprintf("x-protocol-bench-header-%02d", index), value)
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", "7")
	_, _ = w.Write([]byte("headers"))
}

func textHandler(body string, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", strconvLen(body))
		_, _ = w.Write([]byte(body))
	}
}

func streamBytesHandler(w http.ResponseWriter, r *http.Request) {
	params, err := parseStreamBytesParams(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	if err := streamBytes(r.Context(), w, params); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}

		log.Printf("failed to stream response body: %v", err)
	}
}

type streamBytesParams struct {
	chunks  int
	size    int
	delayMs int
}

func parseStreamBytesParams(query url.Values) (streamBytesParams, error) {
	params := streamBytesParams{
		chunks:  streamBytesCanonicalRows,
		size:    streamBytesChunkSize,
		delayMs: 0,
	}

	if value := query.Get("chunks"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return streamBytesParams{}, fmt.Errorf("invalid chunks query parameter")
		}

		params.chunks = parsed
	}

	if value := query.Get("size"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return streamBytesParams{}, fmt.Errorf("invalid size query parameter")
		}

		params.size = parsed
	}

	if value := query.Get("delayMs"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return streamBytesParams{}, fmt.Errorf("invalid delayMs query parameter")
		}

		params.delayMs = parsed
	}

	if params.chunks != streamBytesCanonicalRows || params.size != streamBytesChunkSize || params.delayMs != 0 {
		return streamBytesParams{}, fmt.Errorf("unsupported stream query; expected %s", streamBytesCanonicalURL)
	}

	return params, nil
}

func streamBytes(ctx context.Context, w http.ResponseWriter, params streamBytesParams) error {
	chunk := deterministicChunk(params.size)

	flusher, _ := w.(http.Flusher)
	for chunkIndex := 0; chunkIndex < params.chunks; chunkIndex++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if _, err := w.Write(chunk); err != nil {
			return err
		}

		if flusher != nil {
			flusher.Flush()
		}
	}

	return nil
}

func deterministicChunk(size int) []byte {
	chunk := make([]byte, size)
	for i := range chunk {
		chunk[i] = byte(i % 251)
	}

	return chunk
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
