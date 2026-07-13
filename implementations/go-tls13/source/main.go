package main

import (
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
	"path/filepath"
	"strings"
	"time"
)

const (
	implementationID      = "go-tls13"
	implementationVersion = "0.2.0"
	alpn                  = "protocol-lab-tls"
	requiredCipherSuite   = tls.TLS_AES_128_GCM_SHA256
	expectedLeafDERHash   = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
	commandMagic          = "PLABTLS1"
)

var supportedScenarios = []string{
	"tls.handshake.full", "tls.handshake.resumed", "tls.record.throughput", "tls.record.coverage",
}

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
		Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13,
		NextProtos: []string{alpn}, CurvePreferences: []tls.CurveID{tls.X25519},
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
	_ = connection.SetDeadline(time.Now().Add(20 * time.Second))
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
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != alpn || state.CipherSuite != requiredCipherSuite || state.CurveID != tls.X25519 {
		fmt.Fprintf(os.Stderr, "unexpected negotiation: version=%x alpn=%q cipher=%s curve=%s\n", state.Version, state.NegotiatedProtocol, tls.CipherSuiteName(state.CipherSuite), state.CurveID)
		return
	}
	requested := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if requested == "tls.handshake.full" || requested == "tls.handshake.resumed" || requested == "" {
		return
	}
	if requested != "tls.record.throughput" && requested != "tls.record.coverage" {
		fmt.Fprintf(os.Stderr, "scenario %q is not supported by this target\n", requested)
		return
	}
	if err := serveRecordOperation(tlsConnection); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

// The 13-byte command is setup material outside the measured payload window:
// 8-byte magic, one direction byte, and a network-order uint32 payload length.
func serveRecordOperation(connection *tls.Conn) error {
	header := make([]byte, 13)
	if _, err := io.ReadFull(connection, header); err != nil {
		return fmt.Errorf("read record command: %w", err)
	}
	if string(header[:8]) != commandMagic {
		return errors.New("record command magic mismatch")
	}
	direction, size := header[8], int(binary.BigEndian.Uint32(header[9:13]))
	if size != 1024 && size != 65536 && size != 1048576 {
		return fmt.Errorf("unsupported record payload size %d", size)
	}
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = 0x5a
	}
	expected := sha256.Sum256(payload)
	switch direction {
	case 'S':
		if err := writeAll(connection, payload); err != nil {
			return fmt.Errorf("send deterministic payload: %w", err)
		}
	case 'C':
		received := make([]byte, size)
		if _, err := io.ReadFull(connection, received); err != nil {
			return fmt.Errorf("receive deterministic payload: %w", err)
		}
		actual := sha256.Sum256(received)
		if actual != expected {
			return errors.New("received payload hash mismatch")
		}
		ack := append([]byte("OK"), actual[:]...)
		if err := writeAll(connection, ack); err != nil {
			return fmt.Errorf("send payload acknowledgement: %w", err)
		}
	default:
		return fmt.Errorf("unsupported record direction %q", direction)
	}
	return nil
}

func writeAll(writer io.Writer, value []byte) error {
	for len(value) > 0 {
		n, err := writer.Write(value)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		value = value[n:]
	}
	return nil
}

func writeReady(listen string) {
	profileID, certificateProfileID := "plab-tls13-p256-v1", "plab-single-leaf-p256-v1"
	if strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")) == "tls.record.coverage" {
		profileID, certificateProfileID = "plab-tls13-aes128gcm-p256-server-auth-v2", "plab-single-leaf-p256-server-v2"
	}
	ready := map[string]any{
		"eventName": "ready", "implementationId": implementationID, "implementationVersion": implementationVersion,
		"listenAddress": listen, "protocol": "tls", "tlsVersion": "TLS1.3", "alpn": alpn,
		"cipherSuite": tls.CipherSuiteName(requiredCipherSuite), "keyExchangeGroup": "X25519",
		"tlsProfileId": profileID, "certificateProfileId": certificateProfileID, "certificateDerSha256": expectedLeafDERHash,
		"sessionTicketsEnabled": true, "supportedScenarios": supportedScenarios,
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
func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func fatal(err error)          { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
