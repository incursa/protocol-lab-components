package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	masque "github.com/quic-go/masque-go"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/yosida95/uritemplate/v3"
)

const (
	packageVersion  = "0.1.0"
	upstreamVersion = "v0.4.0"
	proxyAuthority  = "masque-proxy.plab.test"
	targetAuthority = "masque-echo.plab.test:4433"
	proxyBind       = ":4443"
	targetBind      = ":4433"
	pathTemplate    = "/.well-known/masque/udp/{target_host}/{target_port}/"
)

func main() {
	version := flag.Bool("version", false, "print package and upstream versions")
	flag.Parse()
	if *version {
		fmt.Printf("masque-go-connect-udp %s masque-go %s\n", packageVersion, upstreamVersion)
		return
	}

	echoConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 4433})
	if err != nil {
		log.Fatalf("udp-target listen failed: %v", err)
	}
	defer echoConn.Close()
	go serveEcho(echoConn)
	log.Printf("role=udp-target authority=%s bind=%s behavior=exact-echo ready=true", targetAuthority, targetBind)

	publicPort := envOr("PLAB_PUBLIC_PORT", "4443")
	uriTemplate := "https://" + net.JoinHostPort(proxyAuthority, publicPort) + pathTemplate
	template, err := uritemplate.New(uriTemplate)
	if err != nil {
		log.Fatalf("invalid URI template: %v", err)
	}
	cert, err := tls.LoadX509KeyPair("/certs/server.pem", "/certs/server-key.pem")
	if err != nil {
		log.Fatalf("certificate load failed: %v", err)
	}
	tlsConfig := http3.ConfigureTLSConfig(&tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13})
	mux := http.NewServeMux()
	proxy := &masque.Proxy{}
	defer proxy.Close()
	u, _ := url.Parse(uriTemplate)
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		req, parseErr := masque.ParseProxyRequest(r, template)
		if parseErr != nil {
			log.Printf("role=proxy parse_error=%q authority=%q path=%q protocol=%q", parseErr, r.Host, r.URL.String(), r.Proto)
			var proxyErr *masque.ProxyRequestParseError
			if errors.As(parseErr, &proxyErr) {
				w.WriteHeader(proxyErr.HTTPStatus)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !strings.EqualFold(req.Target, targetAuthority) {
			log.Printf("role=proxy rejected_target=%q expected=%q", req.Target, targetAuthority)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("X-ProtocolLab-Proxy-Role", "masque-go-connect-udp")
		w.Header().Set("X-ProtocolLab-Target-Role", targetAuthority)
		log.Printf("role=proxy protocol=connect-udp requested_target=%s effective_target=127.0.0.1:4433", req.Target)
		req.Target = "127.0.0.1:4433"
		if proxyErr := proxy.Proxy(w, req); proxyErr != nil {
			log.Printf("role=proxy forwarding_error=%q", proxyErr)
		}
	})
	server := &http3.Server{
		Addr:            proxyBind,
		TLSConfig:       tlsConfig,
		QUICConfig:      &quic.Config{EnableDatagrams: true},
		EnableDatagrams: true,
		Handler:         mux,
	}
	log.Printf("role=proxy implementation=masque-go version=%s authority=%s bind=%s protocol=connect-udp ready=true", upstreamVersion, proxyAuthority, proxyBind)
	if err = server.ListenAndServe(); err != nil {
		log.Fatalf("proxy server failed: %v", err)
	}
}

func serveEcho(conn *net.UDPConn) {
	buffer := make([]byte, 1500)
	for {
		n, peer, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return
		}
		if n != 256 {
			log.Printf("role=udp-target peer=%s rejected_bytes=%d", peer, n)
			continue
		}
		if _, err = conn.WriteToUDP(buffer[:n], peer); err != nil {
			log.Printf("role=udp-target peer=%s echo_error=%q", peer, err)
			continue
		}
		log.Printf("role=udp-target peer=%s echoed_bytes=%d", peer, n)
	}
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lmicroseconds)
	_ = os.Setenv("QUIC_GO_DISABLE_RECEIVE_BUFFER_WARNING", "true")
}
