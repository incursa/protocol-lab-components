package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	implementationID      = "go-dns-doh2"
	implementationVersion = "0.2.0"
	fixtureID             = "dns.plab-test-a.canonical"
	queryHash             = "c46b9fb76019b5a644d0884b17e816cb7c3076275d9468c27d180f70488eb8ec"
	mediaType             = "application/dns-message"
)

var canonicalResponse = mustHex("00008400000100010000000004706c616204746573740000010001c00c00010001000000000004c0000201")

func main() {
	listen := flag.String("listen", defaultListen(), "TCP listen address")
	certPath := flag.String("certificate", envOr("PLAB_DOH2_CERTIFICATE_PATH", filepath.Join("certs", "leaf.pem")), "leaf certificate PEM")
	keyPath := flag.String("private-key", envOr("PLAB_DOH2_PRIVATE_KEY_PATH", filepath.Join("certs", "leaf-key.pem")), "leaf private key PEM")
	flag.Parse()

	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		fatal(err)
	}
	defer listener.Close()

	tlsConfig := &tls.Config{
		MinVersion:             tls.VersionTLS13,
		MaxVersion:             tls.VersionTLS13,
		NextProtos:             []string{"h2"},
		SessionTicketsDisabled: true,
		CurvePreferences:       []tls.CurveID{tls.X25519},
	}
	server := &http.Server{
		Handler:           http.HandlerFunc(handle),
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	ready := map[string]any{
		"status": "ready", "implementationId": implementationID, "version": implementationVersion,
		"host": host, "port": port, "protocol": "doh2", "protocolVariant": "doh-h2-tls-alpn",
		"tlsVersion": "TLS1.3", "alpn": "h2", "fixtureId": fixtureID,
		"authorityMode": "local-fixture-authoritative", "externalUpstream": "prohibited", "cacheState": "disabled",
	}
	data, _ := json.Marshal(ready)
	fmt.Println(string(data))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	err = server.ServeTLS(listener, *certPath, *keyPath)
	if err != nil && err != http.ErrServerClosed {
		fatal(err)
	}
}

func handle(response http.ResponseWriter, request *http.Request) {
	if request.ProtoMajor != 2 || request.TLS == nil || request.TLS.Version != tls.VersionTLS13 || request.TLS.NegotiatedProtocol != "h2" || request.TLS.DidResume {
		http.Error(response, "exact HTTP/2 over fresh TLS 1.3 with h2 ALPN required", http.StatusHTTPVersionNotSupported)
		return
	}
	if request.Method != http.MethodPost || request.URL.Path != "/dns-query" || request.Host != "dns.plab.test" {
		http.Error(response, "exact POST /dns-query authority required", http.StatusNotFound)
		return
	}
	if request.Header.Get("Content-Type") != mediaType || request.Header.Get("Accept") != mediaType || request.Header.Get("Cache-Control") != "no-cache" {
		http.Error(response, "exact DoH request headers required", http.StatusUnsupportedMediaType)
		return
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, 28))
	if err != nil || len(body) != 27 || sha256Hex(body) != queryHash || body[0] != 0 || body[1] != 0 {
		http.Error(response, "request is not dns.plab-test-a.canonical", http.StatusBadRequest)
		return
	}
	response.Header().Set("Content-Type", mediaType)
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(canonicalResponse)
}

func sha256Hex(value []byte) string { return fmt.Sprintf("%x", digest(value)) }
func mustHex(value string) []byte {
	result, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return result
}
func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
func defaultListen() string {
	if value := strings.TrimSpace(os.Getenv("PLAB_DOH2_LISTEN")); value != "" {
		return value
	}
	return net.JoinHostPort("127.0.0.1", envOr("PLAB_DOH2_PORT", "18531"))
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
