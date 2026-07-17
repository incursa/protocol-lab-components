package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	webtransport "github.com/quic-go/webtransport-go"
)

const (
	implementationID      = "webtransport-go"
	implementationVersion = "0.1.2"
	upstreamVersion       = "v0.11.1"
	pathValue             = "/webtransport/echo"
	authority             = "webtransport.plab.test"
	payloadBytes          = 65536
	payloadSHA256         = "4b640d85ab3ba30fd02c9fc9db4a8928f416322ad27022ea58a65aaee68a4df2"
	datagramCount         = 32
	datagramBytes         = 256
	payloadSetSHA256      = "2e975a37b4bff0a8022c0f89ab19e9a8e2599300e557e9b8ce3eff364cd33e8b"
)

func main() {
	listen := flag.String("listen", ":4433", "UDP listen address")
	cert := flag.String("cert", "/certs/leaf.pem", "TLS certificate")
	key := flag.String("key", "/certs/leaf-key.pem", "TLS private key")
	version := flag.Bool("version", false, "print version")
	flag.Parse()
	if *version {
		fmt.Printf("%s %s webtransport-go %s\n", implementationID, implementationVersion, upstreamVersion)
		return
	}

	tlsCertificate, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		log.Fatal(err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{tlsCertificate}, MinVersion: tls.VersionTLS13}
	h3Server := &http3.Server{
		Addr:      *listen,
		TLSConfig: http3.ConfigureTLSConfig(tlsConfig),
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	webtransport.ConfigureHTTP3Server(h3Server)
	server := &webtransport.Server{H3: h3Server, CheckOrigin: func(*http.Request) bool { return true }}
	mux := http.NewServeMux()
	h3Server.Handler = mux
	mux.HandleFunc(pathValue, func(w http.ResponseWriter, r *http.Request) {
		if !authorityMatches(r.Host) {
			http.Error(w, "authority mismatch", http.StatusMisdirectedRequest)
			return
		}
		session, upgradeErr := server.Upgrade(w, r)
		if upgradeErr != nil {
			logJSON("webtransport-session-rejected", map[string]any{"error": upgradeErr.Error()})
			return
		}
		state := session.SessionState()
		logJSON("webtransport-session-accepted", map[string]any{
			"implementationId": implementationID, "protocol": "webtransport-over-h3",
			"alpn": state.ConnectionState.TLS.NegotiatedProtocol, "authority": r.Host, "path": r.URL.Path,
		})
		go handleSession(session)
	})

	logJSON("ready", map[string]any{
		"implementationId": implementationID, "implementationVersion": implementationVersion,
		"upstreamVersion": upstreamVersion, "listenAddress": *listen, "protocol": "webtransport-over-h3",
		"alpn": "h3", "tlsVersion": "TLS 1.3", "path": pathValue, "payloadBytes": payloadBytes,
		"payloadSha256": payloadSHA256, "datagramCount": datagramCount,
		"payloadBytesPerDatagram": datagramBytes, "payloadSetSha256": payloadSetSHA256,
	})
	if err := server.ListenAndServeTLS(*cert, *key); err != nil && !strings.Contains(err.Error(), "closed") {
		log.Fatal(err)
	}
}

func handleSession(session *webtransport.Session) {
	go handleStream(session)
	go handleDatagrams(session)
}

func handleStream(session *webtransport.Session) {
	ctx, cancel := context.WithTimeout(session.Context(), 15*time.Second)
	defer cancel()
	stream, err := session.AcceptStream(ctx)
	if err != nil {
		if session.Context().Err() == nil && ctx.Err() == nil {
			logJSON("webtransport-stream-rejected", map[string]any{"error": err.Error()})
		}
		return
	}
	data, err := io.ReadAll(io.LimitReader(stream, payloadBytes+1))
	if err != nil || len(data) != payloadBytes || hash(data) != payloadSHA256 {
		logJSON("webtransport-stream-invalid", map[string]any{"bytes": len(data), "sha256": hash(data), "error": errorText(err)})
		stream.CancelRead(1)
		stream.CancelWrite(1)
		_ = session.CloseWithError(2, "payload mismatch")
		return
	}
	if _, err = stream.Write(data); err != nil {
		logJSON("webtransport-stream-write-failed", map[string]any{"error": err.Error()})
		_ = session.CloseWithError(3, "echo failed")
		return
	}
	if err = stream.Close(); err != nil {
		logJSON("webtransport-stream-close-failed", map[string]any{"error": err.Error()})
		return
	}
	logJSON("webtransport-stream-echoed", map[string]any{
		"implementationId": implementationID, "bytes": len(data), "sha256": payloadSHA256,
		"streamDirection": "client-initiated-bidirectional", "streamCount": 1,
	})
}

func handleDatagrams(session *webtransport.Session) {
	ctx, cancel := context.WithTimeout(session.Context(), 15*time.Second)
	defer cancel()
	setHasher := sha256.New()
	for index := 0; index < datagramCount; index++ {
		data, err := session.ReceiveDatagram(ctx)
		if err != nil {
			if index > 0 && session.Context().Err() == nil && ctx.Err() == nil {
				logJSON("webtransport-datagram-receive-failed", map[string]any{"index": index, "error": err.Error()})
				_ = session.CloseWithError(4, "datagram receive failed")
			}
			return
		}
		expected := makeDatagramPayload(index)
		if len(data) != datagramBytes || !bytes.Equal(data, expected) {
			logJSON("webtransport-datagram-invalid", map[string]any{"index": index, "bytes": len(data), "sha256": hash(data)})
			_ = session.CloseWithError(5, "datagram payload mismatch")
			return
		}
		_, _ = setHasher.Write(data)
		if err := session.SendDatagram(data); err != nil {
			logJSON("webtransport-datagram-send-failed", map[string]any{"index": index, "error": err.Error()})
			_ = session.CloseWithError(6, "datagram echo failed")
			return
		}
	}
	observed := hex.EncodeToString(setHasher.Sum(nil))
	if observed != payloadSetSHA256 {
		logJSON("webtransport-datagram-set-invalid", map[string]any{"sha256": observed})
		_ = session.CloseWithError(7, "datagram payload set mismatch")
		return
	}
	logJSON("webtransport-datagrams-echoed", map[string]any{
		"implementationId": implementationID, "datagramCount": datagramCount,
		"payloadBytesPerDatagram": datagramBytes, "payloadSetSha256": observed,
	})
}

func makeDatagramPayload(datagramIndex int) []byte {
	payload := make([]byte, datagramBytes)
	for octetIndex := range payload {
		payload[octetIndex] = byte((datagramIndex + octetIndex) % 251)
	}
	return payload
}

func authorityMatches(value string) bool {
	host := value
	if parsed, _, err := net.SplitHostPort(value); err == nil {
		host = parsed
	}
	return strings.EqualFold(host, authority)
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func logJSON(event string, values map[string]any) {
	values["eventName"] = event
	data, _ := json.Marshal(values)
	fmt.Fprintln(os.Stdout, string(data))
}
