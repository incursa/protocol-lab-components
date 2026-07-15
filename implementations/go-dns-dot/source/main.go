package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	implementationID      = "go-dns-dot"
	implementationVersion = "0.2.0"
	alpn                  = "dot"
	requiredQueryHash     = "c46b9fb76019b5a644d0884b17e816cb7c3076275d9468c27d180f70488eb8ec"
)

var canonicalResponse = mustHex("00008400000100010000000004706c616204746573740000010001c00c00010001000000000004c0000201")

func main() {
	listen := flag.String("listen", defaultListen(), "TCP listen address")
	certPath := flag.String("certificate", envOr("PLAB_DOT_CERTIFICATE_PATH", filepath.Join("certs", "leaf.pem")), "leaf certificate PEM")
	keyPath := flag.String("private-key", envOr("PLAB_DOT_PRIVATE_KEY_PATH", filepath.Join("certs", "leaf-key.pem")), "leaf private key PEM")
	flag.Parse()
	certificate, err := tls.LoadX509KeyPair(*certPath, *keyPath)
	if err != nil {
		fatal(err)
	}
	config := &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, NextProtos: []string{alpn}, SessionTicketsDisabled: true, CurvePreferences: []tls.CurveID{tls.X25519}}
	listener, err := tls.Listen("tcp", *listen, config)
	if err != nil {
		fatal(err)
	}
	defer listener.Close()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	ready := map[string]any{"status": "ready", "implementationId": implementationID, "version": implementationVersion, "host": host, "port": port, "protocol": "dot", "protocolVariant": "dot-tls1.3-tcp", "tlsVersion": "TLS1.3", "alpn": alpn, "fixtureId": "dns.plab-test-a.canonical", "authorityMode": "local-fixture-authoritative", "externalUpstream": "prohibited", "cacheState": "disabled"}
	data, _ := json.Marshal(ready)
	fmt.Println(string(data))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var wait sync.WaitGroup
	go func() { <-ctx.Done(); _ = listener.Close() }()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		wait.Add(1)
		go func() { defer wait.Done(); handle(conn) }()
	}
	wait.Wait()
}

func handle(raw net.Conn) {
	defer raw.Close()
	conn, ok := raw.(*tls.Conn)
	if !ok {
		fmt.Fprintln(os.Stderr, "rejected non-TLS connection")
		return
	}
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	if err := conn.Handshake(); err != nil {
		fmt.Fprintln(os.Stderr, "TLS handshake rejected:", err)
		return
	}
	state := conn.ConnectionState()
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != alpn || state.DidResume {
		fmt.Fprintln(os.Stderr, "TLS negotiation mismatch")
		return
	}
	outstanding := map[uint16]struct{}{}
	for {
		prefix := make([]byte, 2)
		if _, err := io.ReadFull(conn, prefix); err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				fmt.Fprintln(os.Stderr, "DoT prefix read failed:", err)
			}
			return
		}
		length := int(binary.BigEndian.Uint16(prefix))
		if length != 27 {
			fmt.Fprintln(os.Stderr, "rejected non-canonical query length", length)
			return
		}
		query := make([]byte, length)
		if _, err := io.ReadFull(conn, query); err != nil {
			fmt.Fprintln(os.Stderr, "DoT query read failed:", err)
			return
		}
		id := binary.BigEndian.Uint16(query[:2])
		if id == 0 {
			fmt.Fprintln(os.Stderr, "rejected zero runtime message ID")
			return
		}
		if _, duplicate := outstanding[id]; duplicate {
			fmt.Fprintln(os.Stderr, "rejected reused runtime message ID")
			return
		}
		outstanding[id] = struct{}{}
		normalized := append([]byte(nil), query...)
		binary.BigEndian.PutUint16(normalized[:2], 0)
		if sha256Hex(normalized) != requiredQueryHash {
			fmt.Fprintln(os.Stderr, "rejected query outside dns.plab-test-a.canonical")
			return
		}
		response := append([]byte(nil), canonicalResponse...)
		binary.BigEndian.PutUint16(response[:2], id)
		framed := make([]byte, 2+len(response))
		binary.BigEndian.PutUint16(framed[:2], uint16(len(response)))
		copy(framed[2:], response)
		if _, err := conn.Write(framed); err != nil {
			fmt.Fprintln(os.Stderr, "DoT response write failed:", err)
			return
		}
		delete(outstanding, id)
	}
}

func sha256Hex(value []byte) string {
	// Keep the fixture material local and dependency-free.
	return fmt.Sprintf("%x", sha256Sum(value))
}

func sha256Sum(value []byte) [32]byte {
	// Defined in hash.go to keep imports explicit for tests.
	return digest(value)
}

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
	if value := strings.TrimSpace(os.Getenv("PLAB_DOT_LISTEN")); value != "" {
		return value
	}
	return net.JoinHostPort("127.0.0.1", envOr("PLAB_DOT_PORT", "18530"))
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
