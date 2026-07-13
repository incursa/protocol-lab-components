package main

import (
	"bufio"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	implementationID      = "go-http1-websocket-tls"
	implementationVersion = "0.1.0"
	websocketGUID         = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	textPayload           = "protocol-lab"
	controlPayload        = "protocol-lab-ping"
)

func main() {
	listenAddress := envOr("PLAB_HTTP1_WEBSOCKET_TLS_LISTEN", net.JoinHostPort("127.0.0.1", envOr("PLAB_TARGET_PORT", "18443")))
	certificatePath := envOr("PLAB_TLS_CERTIFICATE_PATH", filepath.Join(packagedRoot(), "certs", "leaf.pem"))
	privateKeyPath := envOr("PLAB_TLS_PRIVATE_KEY_PATH", filepath.Join(packagedRoot(), "certs", "leaf-key.pem"))
	certificate, err := tls.LoadX509KeyPair(certificatePath, privateKeyPath)
	if err != nil {
		fatal(fmt.Errorf("load TLS certificate: %w", err))
	}
	if len(certificate.Certificate) != 1 {
		fatal(fmt.Errorf("expected one leaf certificate, observed %d", len(certificate.Certificate)))
	}
	leaf, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		fatal(fmt.Errorf("parse TLS leaf certificate: %w", err))
	}
	tcpListener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		fatal(err)
	}
	listener := tls.NewListener(tcpListener, &tls.Config{
		Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13,
		NextProtos: []string{"http/1.1"}, SessionTicketsDisabled: true,
	})
	defer listener.Close()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	ready := map[string]any{
		"status": "ready", "implementationId": implementationID, "version": implementationVersion,
		"host": host, "port": port, "protocol": "h1", "protocolVersion": "HTTP/1.1",
		"protocolVariant": "websocket-h1-tls1.3-upgrade", "binding": "rfc6455-http1-upgrade",
		"transportSecurity": "tls", "tlsVersion": "TLS1.3", "alpn": "http/1.1", "serverName": "websocket.plab.test",
		"certificateDerSha256":  fmt.Sprintf("%x", sha256.Sum256(leaf.Raw)),
		"certificateSpkiSha256": fmt.Sprintf("%x", sha256.Sum256(leaf.RawSubjectPublicKeyInfo)), "path": "/websocket",
	}
	encoded, _ := json.Marshal(ready)
	fmt.Println(string(encoded))

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
			fmt.Fprintln(os.Stderr, "accept failed:", err)
			continue
		}
		wait.Add(1)
		go func() {
			defer wait.Done()
			if err := handleConnection(conn); err != nil && !errors.Is(err, io.EOF) {
				fmt.Fprintln(os.Stderr, "connection rejected:", err)
			}
		}()
	}
	wait.Wait()
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	tlsConnection, ok := conn.(*tls.Conn)
	if !ok {
		return errors.New("cleartext connection reached TLS-only WebSocket target")
	}
	if err := tlsConnection.Handshake(); err != nil {
		return fmt.Errorf("TLS 1.3 handshake: %w", err)
	}
	state := tlsConnection.ConnectionState()
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != "http/1.1" || state.DidResume {
		return fmt.Errorf("TLS policy mismatch: version=%s alpn=%q didResume=%t", tls.VersionName(state.Version), state.NegotiatedProtocol, state.DidResume)
	}
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		return fmt.Errorf("HTTP/1.1 opening handshake: %w", err)
	}
	defer request.Body.Close()
	if err := validateUpgradeRequest(request); err != nil {
		_, _ = io.WriteString(conn, "HTTP/1.1 400 Bad Request\r\nConnection: close\r\nContent-Length: 0\r\n\r\n")
		return err
	}
	accept := websocketAccept(request.Header.Get("Sec-WebSocket-Key"))
	response := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := io.WriteString(conn, response); err != nil {
		return err
	}
	for {
		frame, err := readFrame(reader, true)
		if err != nil {
			return err
		}
		if !frame.fin || frame.rsv != 0 {
			return errors.New("fragmented or RSV-bearing frames are not supported by this exact target")
		}
		switch frame.opcode {
		case 0x1:
			if string(frame.payload) != textPayload {
				return errors.New("unexpected text payload")
			}
			if err := writeFrame(conn, 0x1, frame.payload, false); err != nil {
				return err
			}
		case 0x2:
			if len(frame.payload) != 1024 {
				return errors.New("unexpected binary payload length")
			}
			for _, value := range frame.payload {
				if value != 66 {
					return errors.New("unexpected binary payload content")
				}
			}
			if err := writeFrame(conn, 0x2, frame.payload, false); err != nil {
				return err
			}
		case 0x9:
			if string(frame.payload) != controlPayload {
				return errors.New("unexpected ping payload")
			}
			if err := writeFrame(conn, 0xA, frame.payload, false); err != nil {
				return err
			}
		case 0x8:
			if len(frame.payload) != 2 || binary.BigEndian.Uint16(frame.payload) != 1000 {
				return errors.New("close payload must contain code 1000 with an empty UTF-8 reason")
			}
			return writeFrame(conn, 0x8, frame.payload, false)
		default:
			return fmt.Errorf("unsupported RFC 6455 opcode 0x%x", frame.opcode)
		}
	}
}

func packagedRoot() string {
	executable, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(executable), "..", ".."))
}

func validateUpgradeRequest(request *http.Request) error {
	var failures []string
	if request.Method != http.MethodGet {
		failures = append(failures, "method is not GET")
	}
	if request.ProtoMajor != 1 || request.ProtoMinor != 1 {
		failures = append(failures, "protocol is not exact HTTP/1.1")
	}
	if request.URL.Path != "/websocket" {
		failures = append(failures, "path is not /websocket")
	}
	if request.Host != "websocket.plab.test" {
		failures = append(failures, "authority is not websocket.plab.test")
	}
	if !strings.EqualFold(request.Header.Get("Upgrade"), "websocket") || !hasToken(request.Header.Get("Connection"), "upgrade") {
		failures = append(failures, "Upgrade/Connection headers are invalid")
	}
	if request.Header.Get("Sec-WebSocket-Version") != "13" {
		failures = append(failures, "Sec-WebSocket-Version is not 13")
	}
	decoded, err := base64.StdEncoding.DecodeString(request.Header.Get("Sec-WebSocket-Key"))
	if err != nil || len(decoded) != 16 {
		failures = append(failures, "Sec-WebSocket-Key is not base64 of exactly 16 octets")
	}
	for _, absent := range []string{"Origin", "Sec-WebSocket-Protocol", "Sec-WebSocket-Extensions"} {
		if request.Header.Get(absent) != "" {
			failures = append(failures, absent+" must be absent")
		}
	}
	if len(failures) != 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

type wireFrame struct {
	fin     bool
	rsv     byte
	opcode  byte
	masked  bool
	payload []byte
}

func readFrame(reader *bufio.Reader, requireMask bool) (wireFrame, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return wireFrame{}, err
	}
	result := wireFrame{fin: header[0]&0x80 != 0, rsv: header[0] & 0x70, opcode: header[0] & 0x0f, masked: header[1]&0x80 != 0}
	if result.masked != requireMask {
		return result, errors.New("RFC 6455 masking direction mismatch")
	}
	length, err := readPayloadLength(reader, header[1]&0x7f)
	if err != nil {
		return result, err
	}
	if length > 1<<20 {
		return result, errors.New("frame exceeds package limit")
	}
	mask := make([]byte, 4)
	if result.masked {
		if _, err := io.ReadFull(reader, mask); err != nil {
			return result, err
		}
	}
	result.payload = make([]byte, length)
	if _, err := io.ReadFull(reader, result.payload); err != nil {
		return result, err
	}
	if result.masked {
		for i := range result.payload {
			result.payload[i] ^= mask[i%4]
		}
	}
	if result.opcode >= 0x8 && (!result.fin || len(result.payload) > 125) {
		return result, errors.New("invalid fragmented or oversized control frame")
	}
	return result, nil
}

func writeFrame(writer io.Writer, opcode byte, payload []byte, masked bool) error {
	header := []byte{0x80 | opcode}
	maskBit := byte(0)
	if masked {
		maskBit = 0x80
	}
	switch {
	case len(payload) <= 125:
		header = append(header, maskBit|byte(len(payload)))
	case len(payload) <= 65535:
		header = append(header, maskBit|126, byte(len(payload)>>8), byte(len(payload)))
	default:
		return errors.New("payload exceeds package limit")
	}
	if masked {
		return errors.New("target must never mask server-to-client frames")
	}
	_, err := writer.Write(append(header, payload...))
	return err
}

func readPayloadLength(reader io.Reader, encoded byte) (int, error) {
	switch encoded {
	case 126:
		value := make([]byte, 2)
		if _, err := io.ReadFull(reader, value); err != nil {
			return 0, err
		}
		return int(binary.BigEndian.Uint16(value)), nil
	case 127:
		value := make([]byte, 8)
		if _, err := io.ReadFull(reader, value); err != nil {
			return 0, err
		}
		length := binary.BigEndian.Uint64(value)
		if length > uint64(^uint(0)>>1) {
			return 0, errors.New("frame length overflows int")
		}
		return int(length), nil
	default:
		return int(encoded), nil
	}
}

func websocketAccept(key string) string {
	digest := sha1.Sum([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(digest[:])
}
func hasToken(value, token string) bool {
	for _, candidate := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(candidate), token) {
			return true
		}
	}
	return false
}
func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
