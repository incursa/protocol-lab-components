package main

import (
	"context"
	"crypto/sha256"
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
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
)

const (
	implementationID      = "go-dns-doq"
	implementationVersion = "0.1.0"
	fixtureID             = "dns.plab-test-a.canonical"
	queryHash             = "c46b9fb76019b5a644d0884b17e816cb7c3076275d9468c27d180f70488eb8ec"
	responseHash          = "9d488461675ad5ab9f74c7b203861e1ad17521e413a407a25e6611012a595620"
	doqALPN               = "doq"
	doqProtocolError      = quic.ApplicationErrorCode(0x2)
)

var (
	canonicalQuery    = mustHex("00000000000100000000000004706c616204746573740000010001")
	canonicalResponse = mustHex("00008400000100010000000004706c616204746573740000010001c00c00010001000000000004c0000201")
)

func main() {
	listen := flag.String("listen", defaultListen(), "UDP listen address")
	certPath := flag.String("certificate", envOr("PLAB_DOQ_CERTIFICATE_PATH", filepath.Join("certs", "leaf.pem")), "leaf certificate PEM")
	keyPath := flag.String("private-key", envOr("PLAB_DOQ_PRIVATE_KEY_PATH", filepath.Join("certs", "leaf-key.pem")), "leaf private key PEM")
	flag.Parse()

	certificate, err := tls.LoadX509KeyPair(*certPath, *keyPath)
	if err != nil {
		fatal(err)
	}
	tlsConfig := &tls.Config{
		Certificates:           []tls.Certificate{certificate},
		MinVersion:             tls.VersionTLS13,
		MaxVersion:             tls.VersionTLS13,
		NextProtos:             []string{doqALPN},
		SessionTicketsDisabled: true,
		CurvePreferences:       []tls.CurveID{tls.X25519},
	}
	quicConfig := &quic.Config{
		Versions:                   []quic.Version{quic.Version1},
		Allow0RTT:                  false,
		HandshakeIdleTimeout:       5 * time.Second,
		MaxIdleTimeout:             30 * time.Second,
		KeepAlivePeriod:            5 * time.Second,
		MaxIncomingStreams:         1024,
		MaxIncomingUniStreams:      -1,
		InitialStreamReceiveWindow: 64 * 1024,
		MaxStreamReceiveWindow:     64 * 1024,
	}
	listener, err := quic.ListenAddr(*listen, tlsConfig, quicConfig)
	if err != nil {
		fatal(err)
	}
	defer listener.Close()

	host, port, _ := net.SplitHostPort(listener.Addr().String())
	ready := map[string]any{
		"status": "ready", "implementationId": implementationID, "version": implementationVersion,
		"host": host, "port": port, "protocol": "doq", "protocolVariant": "dns-over-quic-v1",
		"quicVersion": "v1", "tlsVersion": "TLS1.3", "alpn": doqALPN, "fixtureId": fixtureID,
		"authorityMode": "local-fixture-authoritative", "externalUpstream": "prohibited", "cacheState": "disabled",
	}
	data, _ := json.Marshal(ready)
	fmt.Println(string(data))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	for {
		connection, acceptErr := listener.Accept(ctx)
		if acceptErr != nil {
			if ctx.Err() != nil {
				return
			}
			fatal(acceptErr)
		}
		go serveConnection(ctx, connection)
	}
}

func serveConnection(ctx context.Context, connection *quic.Conn) {
	state := connection.ConnectionState()
	if state.Version != quic.Version1 || state.TLS.Version != tls.VersionTLS13 ||
		state.TLS.NegotiatedProtocol != doqALPN || state.TLS.DidResume || state.Used0RTT {
		_ = connection.CloseWithError(doqProtocolError, "exact QUIC v1 / DoQ TLS binding required")
		return
	}
	for {
		requestStream, err := connection.AcceptStream(ctx)
		if err != nil {
			return
		}
		go handleStream(connection, requestStream)
	}
}

func handleStream(connection *quic.Conn, requestStream *quic.Stream) {
	if err := requestStream.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		_ = connection.CloseWithError(quic.ApplicationErrorCode(0x1), "DoQ stream deadline failed")
		return
	}
	request, err := io.ReadAll(io.LimitReader(requestStream, 30))
	if err != nil || validateRequestFrame(request) != nil {
		_ = connection.CloseWithError(doqProtocolError, "invalid DoQ request message")
		return
	}
	response := frame(canonicalResponse)
	if _, err = requestStream.Write(response); err != nil {
		_ = connection.CloseWithError(quic.ApplicationErrorCode(0x1), "DoQ response write failed")
		return
	}
	if err = requestStream.Close(); err != nil {
		_ = connection.CloseWithError(quic.ApplicationErrorCode(0x1), "DoQ response FIN failed")
	}
}

func validateRequestFrame(value []byte) error {
	if len(value) != 29 || binary.BigEndian.Uint16(value[:2]) != 27 {
		return errors.New("DoQ request framing mismatch")
	}
	message := value[2:]
	if len(message) != 27 || binary.BigEndian.Uint16(message[:2]) != 0 || hash(message) != queryHash || !equal(message, canonicalQuery) {
		return errors.New("DoQ request semantic identity mismatch")
	}
	return nil
}

func frame(message []byte) []byte {
	value := make([]byte, 2+len(message))
	binary.BigEndian.PutUint16(value[:2], uint16(len(message)))
	copy(value[2:], message)
	return value
}

func equal(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
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
	if value := strings.TrimSpace(os.Getenv("PLAB_DOQ_LISTEN")); value != "" {
		return value
	}
	return net.JoinHostPort("127.0.0.1", envOr("PLAB_DOQ_PORT", "18532"))
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
