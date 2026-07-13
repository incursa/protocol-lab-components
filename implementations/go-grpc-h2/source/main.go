package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/http2"
)

const (
	implementationID      = "go-grpc-h2"
	implementationVersion = "0.1.0"
	grpcPath              = "/protocollab.performance.v1.EchoService/UnaryEcho"
	expectedFrameBytes    = 136
)

var expectedPayload = bytes.Repeat([]byte{'G'}, 128)

func main() {
	listen := flag.String("listen", configuredListenAddress(), "TLS listen address")
	certPath := flag.String("cert", envOrDefault("PLAB_TLS_CERT_FILE", defaultMaterialPath("certs/leaf.pem")), "server certificate")
	keyPath := flag.String("key", envOrDefault("PLAB_TLS_KEY_FILE", defaultMaterialPath("certs/leaf-key.pem")), "server private key")
	flag.Parse()

	certificate, err := tls.LoadX509KeyPair(*certPath, *keyPath)
	if err != nil {
		fatal(err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		NextProtos:   []string{"h2"},
	}
	server := &http.Server{Addr: *listen, Handler: http.HandlerFunc(handleUnaryEcho), TLSConfig: tlsConfig}
	if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
		fatal(err)
	}
	ready := map[string]any{
		"implementationId": implementationID, "implementationVersion": implementationVersion,
		"listenAddress": *listen, "protocol": "grpc-over-h2", "tlsVersion": "TLS1.3", "alpn": "h2",
		"scenarioId": "grpc.h2.unary.echo",
	}
	encoded, _ := json.Marshal(ready)
	fmt.Fprintln(os.Stdout, string(encoded))
	if err := server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fatal(err)
	}
}

func handleUnaryEcho(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor != 2 || r.TLS == nil || r.TLS.Version != tls.VersionTLS13 || r.TLS.NegotiatedProtocol != "h2" {
		http.Error(w, "exact TLS 1.3 with HTTP/2 ALPN h2 required", http.StatusHTTPVersionNotSupported)
		return
	}
	if r.Method != http.MethodPost || r.URL.Path != grpcPath {
		http.NotFound(w, r)
		return
	}
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/grpc+proto") || !headerContainsToken(r.Header, "Te", "trailers") {
		http.Error(w, "invalid gRPC request headers", http.StatusUnsupportedMediaType)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, expectedFrameBytes+1))
	if err != nil || validateFrame(body) != nil {
		w.Header().Set("Content-Type", "application/grpc+proto")
		w.Header().Set("Trailer", "grpc-status, grpc-message")
		w.WriteHeader(http.StatusOK)
		w.Header().Set("grpc-status", "3")
		w.Header().Set("grpc-message", "invalid deterministic request")
		return
	}
	w.Header().Set("Content-Type", "application/grpc+proto")
	w.Header().Set("grpc-encoding", "identity")
	w.Header().Set("Trailer", "grpc-status, grpc-message")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	w.Header().Set("grpc-status", "0")
	w.Header().Set("grpc-message", "")
}

func validateFrame(frame []byte) error {
	if len(frame) != expectedFrameBytes {
		return fmt.Errorf("expected %d frame bytes, observed %d", expectedFrameBytes, len(frame))
	}
	if frame[0] != 0 || binary.BigEndian.Uint32(frame[1:5]) != 131 {
		return errors.New("invalid identity-compressed gRPC envelope")
	}
	protobuf := frame[5:]
	if len(protobuf) != 131 || protobuf[0] != 0x0a || protobuf[1] != 0x80 || protobuf[2] != 0x01 || !bytes.Equal(protobuf[3:], expectedPayload) {
		return errors.New("invalid canonical EchoRequest protobuf")
	}
	return nil
}

func headerContainsToken(header http.Header, name, token string) bool {
	for _, value := range header.Values(name) {
		for _, item := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(item), token) {
				return true
			}
		}
	}
	return false
}

func defaultMaterialPath(relative string) string {
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

func configuredListenAddress() string {
	if explicit := strings.TrimSpace(os.Getenv("PLAB_LISTEN_ADDRESS")); explicit != "" {
		return explicit
	}
	if port := strings.TrimSpace(os.Getenv("PLAB_TARGET_PORT")); port != "" {
		return "127.0.0.1:" + port
	}
	return "127.0.0.1:18444"
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
