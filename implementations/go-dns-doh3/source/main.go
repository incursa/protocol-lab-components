package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

const implementationID = "go-dns-doh3"
const implementationVersion = "0.1.0"
const mediaType = "application/dns-message"

func main() {
	listen := flag.String("listen", envOr("PLAB_DOH3_LISTEN", "127.0.0.1:18533"), "UDP listen address")
	cert := flag.String("certificate", envOr("PLAB_DOH3_CERTIFICATE_PATH", filepath.Join("certs", "leaf.pem")), "leaf certificate PEM")
	key := flag.String("private-key", envOr("PLAB_DOH3_PRIVATE_KEY_PATH", filepath.Join("certs", "leaf-key.pem")), "leaf private key PEM")
	flag.Parse()
	for _, f := range fixtures {
		if err := validateFixture(f); err != nil {
			fatal(err)
		}
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, NextProtos: []string{http3.NextProtoH3}, SessionTicketsDisabled: true, CurvePreferences: []tls.CurveID{tls.X25519}}
	server := &http3.Server{Addr: *listen, Handler: http.HandlerFunc(handle), TLSConfig: tlsConfig, QUICConfig: &quic.Config{Versions: []quic.Version{quic.Version1}, Allow0RTT: false, EnableDatagrams: false, MaxIdleTimeout: 30 * time.Second}}
	ready, _ := json.Marshal(map[string]any{"status": "ready", "implementationId": implementationID, "version": implementationVersion, "protocol": "doh3", "protocolVariant": "doh-h3-quic-v1", "quicVersion": "v1", "tlsVersion": "TLS1.3", "alpn": "h3", "authorityMode": "local-fixture-authoritative", "externalUpstream": "prohibited", "cacheState": "disabled", "scenarioCount": len(fixtures)})
	fmt.Println(string(ready))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	if err := server.ListenAndServeTLS(*cert, *key); err != nil && ctx.Err() == nil {
		fatal(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor != 3 || r.Proto != "HTTP/3.0" || r.TLS == nil || r.TLS.Version != tls.VersionTLS13 || r.TLS.NegotiatedProtocol != http3.NextProtoH3 || r.TLS.DidResume {
		http.Error(w, "exact HTTP/3 over fresh TLS 1.3 required", http.StatusHTTPVersionNotSupported)
		return
	}
	if r.Host != "dns.plab.test" || r.URL.Path != "/dns-query" || r.Header.Get("Accept") != mediaType || r.Header.Get("Cache-Control") != "no-cache" {
		http.Error(w, "exact DoH3 authority/path/headers required", http.StatusBadRequest)
		return
	}
	var query []byte
	if r.Method == http.MethodGet {
		if r.Header.Get("Content-Type") != "" || r.ContentLength > 0 {
			http.Error(w, "GET body/content-type prohibited", http.StatusBadRequest)
			return
		}
		if len(r.URL.Query()) != 1 {
			http.Error(w, "exact dns parameter required", http.StatusBadRequest)
			return
		}
		f := fixtures["dns.doh3.get.a"]
		if r.URL.RawQuery != "dns="+getValue(f) {
			http.Error(w, "canonical unpadded base64url dns parameter required", http.StatusBadRequest)
			return
		}
		query = bytesOf(f.QueryHex)
	} else if r.Method == http.MethodPost {
		if r.URL.RawQuery != "" || r.Header.Get("Content-Type") != mediaType {
			http.Error(w, "exact POST media binding required", http.StatusUnsupportedMediaType)
			return
		}
		var err error
		query, err = io.ReadAll(io.LimitReader(r.Body, 1024))
		if err != nil {
			http.Error(w, "query read failed", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "GET or POST required", http.StatusMethodNotAllowed)
		return
	}
	var selected *fixture
	for _, f := range fixtures {
		if f.Method == r.Method && hashOf(query) == f.QueryHash && string(query) == string(bytesOf(f.QueryHex)) {
			copy := f
			selected = &copy
			break
		}
	}
	if selected == nil {
		http.Error(w, "unknown deterministic DNS fixture", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bytesOf(selected.ResponseHex))
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
