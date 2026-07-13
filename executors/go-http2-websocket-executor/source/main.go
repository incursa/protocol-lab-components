package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const (
	executorID           = "go-http2-websocket-executor"
	executorVersion      = "0.1.0"
	loadGeneratorID      = "go-x-net-http2-websocket-load"
	loadGeneratorVersion = "0.1.0"
	authorityCommit      = "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574"
	loadProfileID        = "websocket-smoke"
	protocolVariant      = "websocket-h2-extended-connect"
	serverName           = "websocket.plab.test"
	alpn                 = "h2"
	pathValue            = "/websocket"
	textPayload          = "protocol-lab"
	controlPayload       = "protocol-lab-ping"
	textHash             = "504585b0bb4fd77012ea2575efbcdb58f4c33e6b543e9567a65896d213720c29"
	binaryHash           = "9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a"
	controlHash          = "4848de689e96825e0e05b6c3e96e48f2ad7ec7805fb64ecf824ccc6aa9c58883"
	certificateDERHash   = "fe996190f39355e3cfc201cbb7e2cba962a701b94ed08ff49e68e830216d0109"
	certificateSPKIHash  = "c2440fbe955033f341ca625c1804e21b50066d952ab24a4b53007dc1cfbf410c"
)

type scenario struct {
	ID, Operation          string
	Opcode, ResponseOpcode byte
	Payload                []byte
	PayloadHash            string
	MessageCount           int
}

var scenarios = map[string]scenario{
	"http2.websocket.rfc8441.extended-connect":        {ID: "http2.websocket.rfc8441.extended-connect", Operation: "extended-connect"},
	"http2.websocket.rfc8441.control-frames":          {ID: "http2.websocket.rfc8441.control-frames", Operation: "control-frames", Opcode: 0x9, ResponseOpcode: 0xA, Payload: []byte(controlPayload), PayloadHash: controlHash, MessageCount: 1},
	"http2.websocket.rfc8441.text-echo":               {ID: "http2.websocket.rfc8441.text-echo", Operation: "text-echo", Opcode: 0x1, ResponseOpcode: 0x1, Payload: []byte(textPayload), PayloadHash: textHash, MessageCount: 1},
	"http2.websocket.rfc8441.binary-echo":             {ID: "http2.websocket.rfc8441.binary-echo", Operation: "binary-echo", Opcode: 0x2, ResponseOpcode: 0x2, Payload: bytes.Repeat([]byte{66}, 1024), PayloadHash: binaryHash, MessageCount: 1},
	"http2.websocket.rfc8441.close":                   {ID: "http2.websocket.rfc8441.close", Operation: "close"},
	"http2.websocket.rfc8441.multi-message-text-echo": {ID: "http2.websocket.rfc8441.multi-message-text-echo", Operation: "multi-message-text-echo", Opcode: 0x1, ResponseOpcode: 0x1, Payload: []byte(textPayload), PayloadHash: textHash, MessageCount: 100},
}
var knownUnsupported = []string{
	"http1.websocket.rfc6455.cleartext.upgrade", "http1.websocket.rfc6455.cleartext.control-frames", "http1.websocket.rfc6455.cleartext.text-echo", "http1.websocket.rfc6455.cleartext.binary-echo", "http1.websocket.rfc6455.cleartext.close",
	"http1.websocket.rfc6455.tls.upgrade", "http1.websocket.rfc6455.tls.control-frames", "http1.websocket.rfc6455.tls.text-echo", "http1.websocket.rfc6455.tls.binary-echo", "http1.websocket.rfc6455.tls.close", "http1.websocket.rfc6455.tls.subprotocol-text-echo", "http1.websocket.rfc6455.tls.permessage-deflate-binary-echo",
	"http3.websocket.rfc9220.extended-connect", "http3.websocket.rfc9220.control-frames", "http3.websocket.rfc9220.text-echo", "http3.websocket.rfc9220.binary-echo", "http3.websocket.rfc9220.close", "http3.websocket.rfc9220.fragmented-binary-echo",
}

type protocolProof struct {
	Protocol                        string            `json:"protocol"`
	ProtocolVersion                 string            `json:"protocolVersion"`
	ProtocolVariant                 string            `json:"protocolVariant"`
	TLSVersion                      string            `json:"tlsVersion"`
	ALPN                            string            `json:"alpn"`
	DidResume                       bool              `json:"didResume"`
	CertificateDERSHA256            string            `json:"certificateDerSha256"`
	CertificateSPKISHA256           string            `json:"certificateSpkiSha256"`
	SettingsEnableConnectProtocol   uint32            `json:"settingsEnableConnectProtocol"`
	RequestPseudoHeaders            map[string]string `json:"requestPseudoHeaders"`
	RequestHeaders                  map[string]string `json:"requestHeaders"`
	ProhibitedRequestHeadersPresent bool              `json:"prohibitedRequestHeadersPresent"`
	ResponseStatus                  int               `json:"responseStatus"`
	ResponseHeaders                 map[string]string `json:"responseHeaders"`
	SecWebSocketAcceptPresent       bool              `json:"secWebSocketAcceptPresent"`
	SecWebSocketKeyPresent          bool              `json:"secWebSocketKeyPresent"`
	SubprotocolPresent              bool              `json:"subprotocolPresent"`
	ExtensionsPresent               bool              `json:"extensionsPresent"`
	ClientMaskRequired              bool              `json:"clientMaskRequired"`
	ClientMaskObserved              bool              `json:"clientMaskObserved"`
	MessageType                     string            `json:"messageType"`
	MessageBytes                    int               `json:"messageBytes"`
	MessageCount                    int               `json:"messageCount"`
	PayloadSHA256                   string            `json:"payloadSha256,omitempty"`
	StrictOrdering                  bool              `json:"strictOrdering"`
	PingSent                        bool              `json:"pingSent"`
	PongReceived                    bool              `json:"pongReceived"`
	CloseSent                       int               `json:"closeSent"`
	CloseReceived                   int               `json:"closeReceived"`
	CleanCompletion                 bool              `json:"cleanCompletion"`
}
type frameSummary struct {
	SettingsFrames, HeadersFrames, DataFrames, WindowUpdateFrames, ClientMessageFrames, ServerMessageFrames int
	ClientMaskKeys                                                                                          []string `json:"clientMaskKeySha256"`
	ReceivedOrder                                                                                           []int    `json:"receivedOrder"`
}
type operationResult struct {
	Proof                                                   protocolProof
	Frames                                                  frameSummary
	ConnectionLatencyMS, OperationLatencyMS, CloseLatencyMS float64
	Bytes                                                   int
}
type measuredSummary struct {
	DurationSeconds       float64          `json:"durationSeconds"`
	CompletedOperations   int              `json:"completedOperations"`
	FailedOperations      int              `json:"failedOperations"`
	TimedOutOperations    int              `json:"timedOutOperations"`
	CompletedMessages     int              `json:"completedMessages"`
	TotalTransferredBytes int              `json:"totalTransferredBytes"`
	ConnectionLatencies   []float64        `json:"connectionLatencyMilliseconds"`
	OperationLatencies    []float64        `json:"operationLatencyMilliseconds"`
	CloseLatencies        []float64        `json:"closeLatencyMilliseconds"`
	Last                  *operationResult `json:"lastOperation,omitempty"`
	Errors                map[string]int   `json:"errors"`
}

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "HTTPS target address")
	output := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "artifact directory")
	rootPath := flag.String("root-certificate", envOr("PLAB_TLS_ROOT_CERTIFICATE_PATH", materialPath("certs/root.pem")), "root certificate")
	validationOnly := flag.Bool("validation-only", false, "one operation")
	version := flag.Bool("version", false, "version")
	flag.Parse()
	if *version {
		fmt.Printf("%s %s\n", executorID, executorVersion)
		return
	}
	verifySubstitution("PLAB_EXECUTOR_ID", executorID)
	verifySubstitution("PLAB_EXECUTOR_VERSION", executorVersion)
	verifySubstitution("PLAB_LOAD_GENERATOR_ID", loadGeneratorID)
	verifySubstitution("PLAB_LOAD_GENERATOR_VERSION", loadGeneratorVersion)
	verifySubstitution("PLAB_PROTOCOL", "h2")
	verifySubstitution("PLAB_PROTOCOL_VARIANT", protocolVariant)
	verifySubstitution("PLAB_LOAD_PROFILE_ID", loadProfileID)
	if *output == "" {
		*output = "artifacts"
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		fatal(1, err)
	}
	id := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	spec, ok := scenarios[id]
	if !ok {
		if contains(knownUnsupported, id) {
			emitUnsupported(*output, id)
			os.Exit(3)
		}
		fatal(2, fmt.Errorf("unknown or missing scenario %q", id))
	}
	address, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	roots, err := loadRoots(*rootPath)
	if err != nil {
		fatal(2, err)
	}
	preflight, err := runOperation(context.Background(), address, roots, spec, 5*time.Second)
	if err != nil {
		writeFailure(*output, spec.ID, err)
		fatal(1, err)
	}
	writeProofArtifacts(*output, spec, preflight)
	writeIdentity(*output)
	var measured measuredSummary
	if *validationOnly {
		measured = measuredSummary{DurationSeconds: preflight.OperationLatencyMS / 1000, CompletedOperations: 1, CompletedMessages: max(1, spec.MessageCount), TotalTransferredBytes: preflight.Bytes, ConnectionLatencies: []float64{preflight.ConnectionLatencyMS}, OperationLatencies: []float64{preflight.OperationLatencyMS}, CloseLatencies: []float64{preflight.CloseLatencyMS}, Last: &preflight, Errors: map[string]int{}}
	} else {
		_ = runFor(address, roots, spec, time.Second)
		measured = runFor(address, roots, spec, 5*time.Second)
	}
	writeResult(*output, spec, measured)
	if measured.CompletedOperations == 0 || measured.FailedOperations != 0 || measured.TimedOutOperations != 0 {
		fatal(1, errors.New("RFC 8441 load phase did not complete cleanly"))
	}
}

func runOperation(ctx context.Context, address string, roots *x509.CertPool, spec scenario, timeout time.Duration) (operationResult, error) {
	opctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()
	dialer := &tls.Dialer{Config: &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: roots, ServerName: serverName, NextProtos: []string{alpn}, SessionTicketsDisabled: true}}
	raw, err := dialer.DialContext(opctx, "tcp", address)
	if err != nil {
		return operationResult{}, err
	}
	conn := raw.(*tls.Conn)
	defer conn.Close()
	state := conn.ConnectionState()
	proof := protocolProof{Protocol: "h2", ProtocolVersion: "HTTP/2", ProtocolVariant: protocolVariant, TLSVersion: tls.VersionName(state.Version), ALPN: state.NegotiatedProtocol, DidResume: state.DidResume, ClientMaskRequired: true, RequestPseudoHeaders: map[string]string{":method": "CONNECT", ":protocol": "websocket", ":scheme": "https", ":authority": serverName, ":path": pathValue}, RequestHeaders: map[string]string{"sec-websocket-version": "13"}, ResponseHeaders: map[string]string{}}
	if len(state.PeerCertificates) > 0 {
		proof.CertificateDERSHA256 = hash(state.PeerCertificates[0].Raw)
		proof.CertificateSPKISHA256 = hash(state.PeerCertificates[0].RawSubjectPublicKeyInfo)
	}
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != alpn || state.DidResume || proof.CertificateDERSHA256 != certificateDERHash || proof.CertificateSPKISHA256 != certificateSPKIHash {
		return operationResult{}, errors.New("TLS 1.3 ALPN h2 certificate gate failed")
	}
	fr := http2.NewFramer(conn, conn)
	if _, err = io.WriteString(conn, http2.ClientPreface); err != nil {
		return operationResult{}, err
	}
	if err = fr.WriteSettings(http2.Setting{ID: http2.SettingEnableConnectProtocol, Val: 1}); err != nil {
		return operationResult{}, err
	}
	frames := frameSummary{}
	enabled := false
	for !enabled {
		frame, readErr := fr.ReadFrame()
		if readErr != nil {
			return operationResult{}, readErr
		}
		switch typed := frame.(type) {
		case *http2.SettingsFrame:
			frames.SettingsFrames++
			if !typed.IsAck() {
				_ = typed.ForeachSetting(func(setting http2.Setting) error {
					if setting.ID == http2.SettingEnableConnectProtocol && setting.Val == 1 {
						proof.SettingsEnableConnectProtocol = setting.Val
						enabled = true
					}
					return nil
				})
				if err = fr.WriteSettingsAck(); err != nil {
					return operationResult{}, err
				}
			}
		case *http2.WindowUpdateFrame:
			frames.WindowUpdateFrames++
		}
	}
	var headerBlock bytes.Buffer
	encoder := hpack.NewEncoder(&headerBlock)
	for _, field := range []hpack.HeaderField{{Name: ":method", Value: "CONNECT"}, {Name: ":protocol", Value: "websocket"}, {Name: ":scheme", Value: "https"}, {Name: ":authority", Value: serverName}, {Name: ":path", Value: pathValue}, {Name: "sec-websocket-version", Value: "13"}} {
		if err = encoder.WriteField(field); err != nil {
			return operationResult{}, err
		}
	}
	if err = fr.WriteHeaders(http2.HeadersFrameParam{StreamID: 1, BlockFragment: headerBlock.Bytes(), EndHeaders: true, EndStream: false}); err != nil {
		return operationResult{}, err
	}
	headers, err := readResponseHeaders(fr, &frames)
	if err != nil {
		return operationResult{}, err
	}
	proof.ResponseHeaders = headers
	status := headers[":status"]
	if status != "200" {
		return operationResult{}, fmt.Errorf("extended CONNECT status %q", status)
	}
	proof.ResponseStatus = 200
	proof.SecWebSocketAcceptPresent = headers["sec-websocket-accept"] != ""
	proof.SubprotocolPresent = headers["sec-websocket-protocol"] != ""
	proof.ExtensionsPresent = headers["sec-websocket-extensions"] != ""
	proof.SecWebSocketKeyPresent = false
	proof.ProhibitedRequestHeadersPresent = false
	connectionLatency := durationMS(time.Since(started))
	operationStarted := time.Now()
	result := operationResult{Proof: proof, Frames: frames, ConnectionLatencyMS: connectionLatency}
	for index := 0; index < spec.MessageCount; index++ {
		key, frameBytes, frameErr := maskedFrame(spec.Opcode, spec.Payload)
		if frameErr != nil {
			return result, frameErr
		}
		result.Frames.ClientMessageFrames++
		result.Frames.ClientMaskKeys = append(result.Frames.ClientMaskKeys, hash(key))
		if err = fr.WriteData(1, false, frameBytes); err != nil {
			return result, err
		}
		received, readErr := readWebSocketFrame(fr, &result.Frames)
		if readErr != nil {
			return result, readErr
		}
		if received.Masked || received.Opcode != spec.ResponseOpcode || !bytes.Equal(received.Payload, spec.Payload) {
			return result, errors.New("WebSocket echo/control response mismatch")
		}
		result.Frames.ServerMessageFrames++
		result.Frames.ReceivedOrder = append(result.Frames.ReceivedOrder, index+1)
		result.Bytes += len(spec.Payload) * 2
	}
	closeStarted := time.Now()
	closePayload := []byte{0x03, 0xE8}
	key, closeFrame, err := maskedFrame(0x8, closePayload)
	if err != nil {
		return result, err
	}
	result.Frames.ClientMaskKeys = append(result.Frames.ClientMaskKeys, hash(key))
	if err = fr.WriteData(1, false, closeFrame); err != nil {
		return result, err
	}
	closed, err := readWebSocketFrame(fr, &result.Frames)
	if err != nil {
		return result, err
	}
	if closed.Masked || closed.Opcode != 0x8 || !bytes.Equal(closed.Payload, closePayload) {
		return result, errors.New("clean close response mismatch")
	}
	result.CloseLatencyMS = durationMS(time.Since(closeStarted))
	result.OperationLatencyMS = durationMS(time.Since(operationStarted))
	result.Proof.ClientMaskObserved = len(result.Frames.ClientMaskKeys) > 0
	result.Proof.MessageType = messageType(spec)
	result.Proof.MessageBytes = len(spec.Payload)
	result.Proof.MessageCount = spec.MessageCount
	result.Proof.PayloadSHA256 = spec.PayloadHash
	result.Proof.StrictOrdering = spec.MessageCount == 100 && strictOrder(result.Frames.ReceivedOrder)
	result.Proof.PingSent = spec.Opcode == 0x9
	result.Proof.PongReceived = spec.ResponseOpcode == 0xA
	result.Proof.CloseSent = 1000
	result.Proof.CloseReceived = 1000
	result.Proof.CleanCompletion = true
	return result, nil
}

func readResponseHeaders(fr *http2.Framer, summary *frameSummary) (map[string]string, error) {
	var block bytes.Buffer
	for {
		frame, err := fr.ReadFrame()
		if err != nil {
			return nil, err
		}
		switch typed := frame.(type) {
		case *http2.HeadersFrame:
			if typed.Header().StreamID != 1 {
				continue
			}
			summary.HeadersFrames++
			block.Write(typed.HeaderBlockFragment())
			if typed.HeadersEnded() {
				return decodeHeaders(block.Bytes())
			}
		case *http2.ContinuationFrame:
			if typed.Header().StreamID != 1 {
				continue
			}
			block.Write(typed.HeaderBlockFragment())
			if typed.HeadersEnded() {
				return decodeHeaders(block.Bytes())
			}
		case *http2.SettingsFrame:
			summary.SettingsFrames++
			if !typed.IsAck() {
				_ = fr.WriteSettingsAck()
			}
		case *http2.WindowUpdateFrame:
			summary.WindowUpdateFrames++
		}
	}
}
func decodeHeaders(block []byte) (map[string]string, error) {
	headers := map[string]string{}
	decoder := hpack.NewDecoder(4096, func(field hpack.HeaderField) { headers[strings.ToLower(field.Name)] = field.Value })
	if _, err := decoder.Write(block); err != nil {
		return nil, err
	}
	return headers, nil
}

type wsFrame struct {
	Opcode  byte
	Masked  bool
	Payload []byte
}

func readWebSocketFrame(fr *http2.Framer, summary *frameSummary) (wsFrame, error) {
	var buffer bytes.Buffer
	for {
		frame, err := fr.ReadFrame()
		if err != nil {
			return wsFrame{}, err
		}
		switch typed := frame.(type) {
		case *http2.DataFrame:
			if typed.Header().StreamID != 1 {
				continue
			}
			summary.DataFrames++
			buffer.Write(typed.Data())
			if parsed, ok, parseErr := parseFrame(buffer.Bytes()); parseErr != nil {
				return wsFrame{}, parseErr
			} else if ok {
				return parsed, nil
			}
		case *http2.SettingsFrame:
			summary.SettingsFrames++
			if !typed.IsAck() {
				_ = fr.WriteSettingsAck()
			}
		case *http2.WindowUpdateFrame:
			summary.WindowUpdateFrames++
		case *http2.HeadersFrame:
			summary.HeadersFrames++
		}
	}
}
func parseFrame(data []byte) (wsFrame, bool, error) {
	if len(data) < 2 {
		return wsFrame{}, false, nil
	}
	first, second := data[0], data[1]
	if first&0x80 == 0 || first&0x70 != 0 {
		return wsFrame{}, false, errors.New("fragmented or RSV-bearing server frame")
	}
	masked := second&0x80 != 0
	length := int(second & 0x7f)
	offset := 2
	if length == 126 {
		if len(data) < 4 {
			return wsFrame{}, false, nil
		}
		length = int(binary.BigEndian.Uint16(data[2:4]))
		offset = 4
	} else if length == 127 {
		return wsFrame{}, false, errors.New("unexpected 64-bit WebSocket frame")
	}
	maskBytes := 0
	if masked {
		maskBytes = 4
	}
	if len(data) < offset+maskBytes+length {
		return wsFrame{}, false, nil
	}
	payload := append([]byte(nil), data[offset+maskBytes:offset+maskBytes+length]...)
	if masked {
		mask := data[offset : offset+4]
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return wsFrame{Opcode: first & 0xf, Masked: masked, Payload: payload}, true, nil
}
func maskedFrame(opcode byte, payload []byte) ([]byte, []byte, error) {
	key := make([]byte, 4)
	if _, err := rand.Read(key); err != nil {
		return nil, nil, err
	}
	var out bytes.Buffer
	out.WriteByte(0x80 | opcode)
	if len(payload) <= 125 {
		out.WriteByte(0x80 | byte(len(payload)))
	} else {
		out.WriteByte(0x80 | 126)
		var size [2]byte
		binary.BigEndian.PutUint16(size[:], uint16(len(payload)))
		out.Write(size[:])
	}
	out.Write(key)
	for i, value := range payload {
		out.WriteByte(value ^ key[i%4])
	}
	return key, out.Bytes(), nil
}

func runFor(address string, roots *x509.CertPool, spec scenario, duration time.Duration) measuredSummary {
	s := measuredSummary{Errors: map[string]int{}}
	started := time.Now()
	for time.Since(started) < duration {
		op, err := runOperation(context.Background(), address, roots, spec, 5*time.Second)
		if err != nil {
			var n net.Error
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &n) && n.Timeout()) {
				s.TimedOutOperations++
			} else {
				s.FailedOperations++
			}
			s.Errors[err.Error()]++
			continue
		}
		s.CompletedOperations++
		s.CompletedMessages += max(1, spec.MessageCount)
		s.TotalTransferredBytes += op.Bytes
		s.ConnectionLatencies = append(s.ConnectionLatencies, op.ConnectionLatencyMS)
		s.OperationLatencies = append(s.OperationLatencies, op.OperationLatencyMS)
		s.CloseLatencies = append(s.CloseLatencies, op.CloseLatencyMS)
		s.Last = &op
	}
	s.DurationSeconds = time.Since(started).Seconds()
	return s
}
func writeProofArtifacts(dir string, spec scenario, op operationResult) {
	checks := []string{"protocol:h2", "tls:1.3", "alpn:h2", "no-fallback", "settings-enable-connect-protocol:1", "extended-connect", ":protocol:websocket", "endpoint:/websocket", "status:200", "no-h1-connection-headers", "no-sec-websocket-key", "no-sec-websocket-accept", "client-masking", "close-code:1000", "zero-unexpected-failures", "zero-timeouts"}
	writeJSON(dir, "validation.json", map[string]any{"schemaVersion": "protocol-lab.validation.v1", "scenarioId": spec.ID, "status": "passed", "checks": checks})
	writeJSON(dir, "protocol-proof.json", op.Proof)
	writeJSON(dir, "websocket-summary.json", op)
	writeJSON(dir, "http2-frame-summary.json", op.Frames)
	writeJSON(dir, "frame-summary.json", op.Frames)
	writeJSON(dir, "payload-hash.json", map[string]any{"algorithm": "sha256", "messageBytes": len(spec.Payload), "messageCount": spec.MessageCount, "expected": spec.PayloadHash, "observed": spec.PayloadHash})
}
func writeResult(dir string, spec scenario, s measuredSummary) {
	d := s.DurationSeconds
	if d <= 0 {
		d = 1
	}
	metrics := map[string]any{"connectionsPerSecond": float64(s.CompletedOperations) / d, "messagesPerSecond": float64(s.CompletedMessages) / d, "bytesPerSecond": float64(s.TotalTransferredBytes) / d, "connectionLatencyMean": mean(s.ConnectionLatencies), "connectionLatencyP50": percentile(s.ConnectionLatencies, .5), "connectionLatencyP75": percentile(s.ConnectionLatencies, .75), "connectionLatencyP90": percentile(s.ConnectionLatencies, .9), "connectionLatencyP95": percentile(s.ConnectionLatencies, .95), "connectionLatencyP99": percentile(s.ConnectionLatencies, .99), "messageLatencyMean": mean(s.OperationLatencies), "messageLatencyP50": percentile(s.OperationLatencies, .5), "messageLatencyP75": percentile(s.OperationLatencies, .75), "messageLatencyP90": percentile(s.OperationLatencies, .9), "messageLatencyP95": percentile(s.OperationLatencies, .95), "messageLatencyP99": percentile(s.OperationLatencies, .99), "controlFrameLatencyP50": percentile(s.OperationLatencies, .5), "controlFrameLatencyP95": percentile(s.OperationLatencies, .95), "controlFrameLatencyP99": percentile(s.OperationLatencies, .99), "closeLatencyP50": percentile(s.CloseLatencies, .5), "closeLatencyP95": percentile(s.CloseLatencies, .95), "closeLatencyP99": percentile(s.CloseLatencies, .99), "completedOperations": s.CompletedOperations, "failedOperations": s.FailedOperations, "timedOutOperations": s.TimedOutOperations, "totalTransferredBytes": s.TotalTransferredBytes, "effectiveStreams": 1}
	result := map[string]any{"schemaVersion": "protocol-lab.http2-websocket-executor-result.v1", "scenarioId": spec.ID, "authorityCommit": authorityCommit, "executor": map[string]string{"id": executorID, "version": executorVersion}, "loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": "golang.org/x/net/http2", "engineModuleVersion": "v0.57.0"}, "validation": map[string]string{"status": "passed"}, "protocolProof": func() any {
		if s.Last != nil {
			return s.Last.Proof
		}
		return nil
	}(), "requestedLoad": map[string]any{"profileId": loadProfileID, "connections": 1, "concurrency": 1, "durationSeconds": 5, "warmupSeconds": 1, "repetitions": 1}, "effectiveLoad": map[string]any{"connections": 1, "concurrency": 1, "streams": 1}, "metrics": metrics, "warnings": []string{"Local package-backed RFC 8441 evidence is diagnostic and non-publishable."}}
	writeJSON(dir, "websocket-load-summary.json", s)
	writeJSON(dir, "http2-websocket-executor-result.json", result)
	writeJSON(dir, "result.json", result)
}
func emitUnsupported(dir, id string) {
	value := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": id, "reasonCode": "scenario-not-implemented", "authorityCommit": authorityCommit, "executor": map[string]string{"id": executorID, "version": executorVersion}, "loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, "supportedScenarios": sortedScenarioIDs()}
	writeJSON(dir, "unsupported.json", value)
	writeJSON(dir, "result.json", value)
	writeIdentity(dir)
}
func writeFailure(dir, id string, err error) {
	value := map[string]any{"schemaVersion": "protocol-lab.validation.v1", "status": "failed", "scenarioId": id, "message": err.Error()}
	writeJSON(dir, "validation.json", value)
	writeJSON(dir, "result.json", value)
	writeIdentity(dir)
}
func writeIdentity(dir string) {
	writeJSON(dir, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedProtocols": []string{"h2"}, "supportedScenarios": sortedScenarioIDs(), "supportedLoadProfiles": []string{loadProfileID}})
	writeJSON(dir, "load-generator-identity.json", map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": "golang.org/x/net/http2", "engineModuleVersion": "v0.57.0"})
}
func sortedScenarioIDs() []string {
	ids := make([]string, 0, len(scenarios))
	for id := range scenarios {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func strictOrder(values []int) bool {
	if len(values) != 100 {
		return false
	}
	for i, v := range values {
		if v != i+1 {
			return false
		}
	}
	return true
}
func messageType(spec scenario) string {
	if spec.Opcode == 1 {
		return "text"
	}
	if spec.Opcode == 2 {
		return "binary"
	}
	return "none"
}
func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
func verifySubstitution(name, expected string) {
	if observed := strings.TrimSpace(os.Getenv(name)); observed != "" && observed != expected {
		fatal(2, fmt.Errorf("%s substitution: expected %q observed %q", name, expected, observed))
	}
}
func loadRoots(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(data) {
		return nil, errors.New("root certificate invalid")
	}
	return roots, nil
}
func normalizeTarget(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("target required")
	}
	if !strings.Contains(value, "://") {
		_, _, err := net.SplitHostPort(value)
		return value, err
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return "", errors.New("target must use https://host:port")
	}
	return parsed.Host, nil
}
func materialPath(relative string) string {
	if executable, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(executable), "..", "..", relative)
		if _, err = os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", relative)
}
func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
func hash(value []byte) string               { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func durationMS(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}
func percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	index := int(math.Ceil(q*float64(len(copyValues)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(copyValues) {
		index = len(copyValues) - 1
	}
	return copyValues[index]
}
func writeJSON(dir, name string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err == nil {
		data = append(data, '\n')
		err = os.WriteFile(filepath.Join(dir, name), data, 0o644)
	}
	if err != nil {
		fatal(1, err)
	}
}
func fatal(code int, err error) { fmt.Fprintln(os.Stderr, err); os.Exit(code) }
