package main

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/base64"
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
	implementationVersion = "0.2.0"
	maxFrameBytes         = 1048585
)

func main() {
	listen := flag.String("listen", configuredListenAddress(), "TLS listen address")
	certPath := flag.String("cert", envOrDefault("PLAB_TLS_CERT_FILE", defaultMaterialPath("certs/leaf.pem")), "server certificate")
	keyPath := flag.String("key", envOrDefault("PLAB_TLS_KEY_FILE", defaultMaterialPath("certs/leaf-key.pem")), "server private key")
	flag.Parse()
	certificate, err := tls.LoadX509KeyPair(*certPath, *keyPath)
	if err != nil {
		fatal(err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, NextProtos: []string{"h2"}}
	server := &http.Server{Addr: *listen, Handler: http.HandlerFunc(handleUnary), TLSConfig: tlsConfig}
	if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
		fatal(err)
	}
	ready := map[string]any{"implementationId": implementationID, "implementationVersion": implementationVersion, "listenAddress": *listen, "protocol": "grpc-over-h2", "tlsVersion": "TLS1.3", "alpn": "h2", "scenarioIds": []string{"grpc.h2.unary.echo", "grpc.h2.unary.empty", "grpc.h2.unary.fixed-metadata", "grpc.h2.unary.gzip", "grpc.h2.unary.large"}}
	encoded, _ := json.Marshal(ready)
	fmt.Fprintln(os.Stdout, string(encoded))
	if err := server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fatal(err)
	}
}

func handleUnary(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor != 2 || r.TLS == nil || r.TLS.Version != tls.VersionTLS13 || r.TLS.NegotiatedProtocol != "h2" {
		http.Error(w, "exact TLS 1.3 with HTTP/2 ALPN h2 required", http.StatusHTTPVersionNotSupported)
		return
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	method := strings.TrimPrefix(r.URL.Path, "/protocollab.performance.v1.EchoService/")
	if method != "UnaryEcho" && method != "UnaryGzip" && method != "UnaryFixedMetadata" {
		http.NotFound(w, r)
		return
	}
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/grpc+proto") || !headerContainsToken(r.Header, "Te", "trailers") {
		http.Error(w, "invalid gRPC request headers", http.StatusUnsupportedMediaType)
		return
	}
	compression := "identity"
	if method == "UnaryGzip" {
		compression = "gzip"
	}
	if r.Header.Get("grpc-encoding") != compression || r.Header.Get("grpc-accept-encoding") != compression {
		writeGRPCError(w, "12", "compression profile mismatch")
		return
	}
	if method == "UnaryFixedMetadata" && (r.Header.Get("x-plab-text") != "protocol-lab" || !matchesBinaryMetadata(r.Header.Get("x-plab-bin-bin"))) {
		writeGRPCError(w, "3", "fixed request metadata mismatch")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxFrameBytes+1))
	if err != nil {
		writeGRPCError(w, "13", "request read failed")
		return
	}
	protobuf, payload, err := decodeFrame(body, compression)
	if err != nil || !validScenarioPayload(method, payload, protobuf) {
		writeGRPCError(w, "3", "invalid deterministic request")
		return
	}
	responseFrame, err := encodeFrame(protobuf, compression)
	if err != nil {
		writeGRPCError(w, "13", "response compression failed")
		return
	}
	w.Header().Set("Content-Type", "application/grpc+proto")
	w.Header().Set("grpc-encoding", compression)
	trailers := "grpc-status, grpc-message"
	if method == "UnaryFixedMetadata" {
		w.Header().Set("x-plab-text", "protocol-lab")
		trailers += ", x-plab-bin-bin"
	}
	w.Header().Set("Trailer", trailers)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseFrame)
	w.Header().Set("grpc-status", "0")
	w.Header().Set("grpc-message", "")
	if method == "UnaryFixedMetadata" {
		w.Header().Set("x-plab-bin-bin", "AAECAw==")
	}
}

func matchesBinaryMetadata(value string) bool {
	decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(value, "="))
	return err == nil && bytes.Equal(decoded, []byte{0, 1, 2, 3})
}

func validScenarioPayload(method string, payload, protobuf []byte) bool {
	switch method {
	case "UnaryGzip":
		return len(payload) == 1024 && bytes.Equal(payload, bytes.Repeat([]byte{'B'}, 1024)) && len(protobuf) == 1027
	case "UnaryFixedMetadata":
		return len(payload) == 128 && bytes.Equal(payload, bytes.Repeat([]byte{'G'}, 128)) && len(protobuf) == 131
	case "UnaryEcho":
		return (len(payload) == 0 && len(protobuf) == 0) || (len(payload) == 128 && bytes.Equal(payload, bytes.Repeat([]byte{'G'}, 128)) && len(protobuf) == 131) || (len(payload) == 1<<20 && bytes.Equal(payload, bytes.Repeat([]byte{'L'}, 1<<20)) && len(protobuf) == 1048580)
	default:
		return false
	}
}

func encodeFrame(protobuf []byte, compression string) ([]byte, error) {
	message := protobuf
	flag := byte(0)
	if compression == "gzip" {
		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		if _, err := w.Write(protobuf); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		message = b.Bytes()
		flag = 1
	}
	frame := make([]byte, 5, 5+len(message))
	frame[0] = flag
	binary.BigEndian.PutUint32(frame[1:], uint32(len(message)))
	return append(frame, message...), nil
}
func decodeFrame(frame []byte, compression string) ([]byte, []byte, error) {
	if len(frame) < 5 || int(binary.BigEndian.Uint32(frame[1:5])) != len(frame)-5 {
		return nil, nil, errors.New("invalid gRPC envelope")
	}
	message := frame[5:]
	if compression == "gzip" {
		if frame[0] != 1 {
			return nil, nil, errors.New("gzip flag mismatch")
		}
		reader, err := gzip.NewReader(bytes.NewReader(message))
		if err != nil {
			return nil, nil, err
		}
		decompressed, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			return nil, nil, err
		}
		message = decompressed
	} else if frame[0] != 0 {
		return nil, nil, errors.New("identity flag mismatch")
	}
	payload, err := decodeProtobuf(message)
	return message, payload, err
}
func decodeProtobuf(p []byte) ([]byte, error) {
	if len(p) == 0 {
		return []byte{}, nil
	}
	if p[0] != 0x0a {
		return nil, errors.New("unexpected protobuf field")
	}
	var n uint64
	var shift uint
	i := 1
	for {
		if i >= len(p) || shift > 63 {
			return nil, errors.New("invalid protobuf length")
		}
		b := p[i]
		i++
		n |= uint64(b&0x7f) << shift
		if b < 0x80 {
			break
		}
		shift += 7
	}
	if int(n) != len(p)-i {
		return nil, errors.New("protobuf payload length mismatch")
	}
	return p[i:], nil
}
func writeGRPCError(w http.ResponseWriter, status, message string) {
	w.Header().Set("Content-Type", "application/grpc+proto")
	w.Header().Set("Trailer", "grpc-status, grpc-message")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("grpc-status", status)
	w.Header().Set("grpc-message", message)
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
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
