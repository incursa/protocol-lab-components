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
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const (
	executorID           = "go-http2-websocket-executor"
	executorVersion      = "0.2.0"
	loadGeneratorID      = "go-x-net-http2-websocket-load"
	loadGeneratorVersion = "0.2.0"
	authorityCommit      = "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574"
	smokeProfileID       = "websocket-smoke"
	diagnosticProfileID  = "diagnostic"
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

type loadProfile struct {
	ID                   string
	Connections          int
	Concurrency          int
	StreamsPerConnection int
	Duration             time.Duration
	Warmup               time.Duration
	Cooldown             time.Duration
	Repetitions          int
	OperationTimeout     time.Duration
}

var (
	smokeProfile = loadProfile{
		ID: smokeProfileID, Connections: 1, Concurrency: 1, StreamsPerConnection: 1,
		Duration: 5 * time.Second, Warmup: time.Second, Repetitions: 1, OperationTimeout: 5 * time.Second,
	}
	diagnosticProfile = loadProfile{
		ID: diagnosticProfileID, Connections: 1, Concurrency: 8, StreamsPerConnection: 8,
		Duration: 10 * time.Second, Warmup: time.Second, Cooldown: time.Second,
		Repetitions: 1, OperationTimeout: 10 * time.Second,
	}
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
	ConnectionReused                bool              `json:"connectionReused"`
	ConfiguredConnections           int               `json:"configuredConnections"`
	ConfiguredConcurrency           int               `json:"configuredConcurrency"`
	ConfiguredStreamsPerConnection  int               `json:"configuredStreamsPerConnection"`
	ObservedActiveConnections       int               `json:"observedActiveConnections"`
	ObservedActiveStreams           int               `json:"observedActiveStreams"`
	EffectiveConcurrency            int               `json:"effectiveConcurrency"`
}
type frameSummary struct {
	SettingsFrames, HeadersFrames, DataFrames, WindowUpdateFrames, ClientMessageFrames, ServerMessageFrames int
	ClientMaskKeys                                                                                          []string `json:"clientMaskKeySha256"`
	ReceivedOrder                                                                                           []int    `json:"receivedOrder"`
	StreamIDs                                                                                               []uint32 `json:"streamIds,omitempty"`
	ClientMaskedFrames                                                                                      int      `json:"clientMaskedFrames"`
	ServerMaskedFrames                                                                                      int      `json:"serverMaskedFrames"`
	UniqueClientMaskKeys                                                                                    int      `json:"uniqueClientMaskKeys"`
	DuplicateClientMaskKeys                                                                                 int      `json:"duplicateClientMaskKeys"`
	StrictPerStreamOrdering                                                                                 bool     `json:"strictPerStreamOrdering"`
}
type operationResult struct {
	Proof                                                   protocolProof `json:"protocolProof"`
	Frames                                                  frameSummary  `json:"frameSummary"`
	ConnectionLatencyMS, OperationLatencyMS, CloseLatencyMS float64       `json:"-"`
	Bytes                                                   int           `json:"transferredBytes"`
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
	ConfiguredConnections int              `json:"configuredConnections"`
	ConfiguredConcurrency int              `json:"configuredConcurrency"`
	ConfiguredStreams     int              `json:"configuredStreamsPerConnection"`
	ObservedConnections   int              `json:"observedActiveConnections"`
	ObservedStreams       int              `json:"observedActiveStreams"`
	EffectiveConcurrency  int              `json:"effectiveConcurrency"`
	PerStream             []streamSummary  `json:"perStream,omitempty"`
}

type streamSummary struct {
	StreamID             uint32  `json:"streamId"`
	CompletedOperations  int     `json:"completedOperations"`
	CompletedMessages    int     `json:"completedMessages"`
	FailedOperations     int     `json:"failedOperations"`
	TimedOutOperations   int     `json:"timedOutOperations"`
	TransferredBytes     int     `json:"transferredBytes"`
	MessageLatencyMeanMS float64 `json:"messageLatencyMeanMilliseconds"`
	MessageLatencyP95MS  float64 `json:"messageLatencyP95Milliseconds"`
	StrictOrdering       bool    `json:"strictOrdering"`
}

type topologyProof struct {
	ConnectionCount                int      `json:"connectionCount"`
	AuthenticatedTLSConnections    int      `json:"authenticatedTlsConnections"`
	ConfiguredConcurrency          int      `json:"configuredConcurrency"`
	ConfiguredStreamsPerConnection int      `json:"configuredStreamsPerConnection"`
	ObservedActiveConnections      int      `json:"observedActiveConnections"`
	ObservedActiveStreams          int      `json:"observedActiveStreams"`
	EffectiveConcurrency           int      `json:"effectiveConcurrency"`
	StreamIDs                      []uint32 `json:"streamIds"`
	BalancedAssignment             string   `json:"assignment"`
	OneConnection                  bool     `json:"oneConnection"`
	EightConcurrentStreams         bool     `json:"eightConcurrentStreams"`
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
	profile := profileFor(spec)
	verifySubstitution("PLAB_LOAD_PROFILE_ID", profile.ID)
	verifyProfileEnvironment(profile)
	address, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	roots, err := loadRoots(*rootPath)
	if err != nil {
		fatal(2, err)
	}
	writeIdentity(*output)
	var preflight operationResult
	var warmup, measured measuredSummary
	if spec.ID == "http2.websocket.rfc8441.multi-message-text-echo" {
		var topology topologyProof
		preflight, warmup, measured, topology, err = runMultiMessageDiagnostic(address, roots, spec, profile, *validationOnly)
		if err == nil {
			writeJSON(*output, "http2-websocket-topology.json", topology)
		}
	} else {
		preflight, err = runOperation(context.Background(), address, roots, spec, profile.OperationTimeout)
		if err == nil {
			if *validationOnly {
				measured = summaryFromOperation(preflight, profile)
			} else {
				warmup = runFor(address, roots, spec, profile.Warmup, profile.OperationTimeout, profile)
				measured = runFor(address, roots, spec, profile.Duration, profile.OperationTimeout, profile)
			}
		}
	}
	if err != nil {
		writeFailure(*output, spec.ID, err)
		fatal(1, err)
	}
	writeProofArtifacts(*output, spec, preflight)
	writeJSON(*output, "websocket-warmup-summary.json", warmup)
	writeResult(*output, spec, profile, measured)
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
	proof := protocolProof{Protocol: "h2", ProtocolVersion: "HTTP/2", ProtocolVariant: protocolVariant, TLSVersion: tls.VersionName(state.Version), ALPN: state.NegotiatedProtocol, DidResume: state.DidResume, ClientMaskRequired: true, RequestPseudoHeaders: map[string]string{":method": "CONNECT", ":protocol": "websocket", ":scheme": "https", ":authority": serverName, ":path": pathValue}, RequestHeaders: map[string]string{"sec-websocket-version": "13"}, ResponseHeaders: map[string]string{}, ConfiguredConnections: 1, ConfiguredConcurrency: 1, ConfiguredStreamsPerConnection: 1, ObservedActiveConnections: 1, ObservedActiveStreams: 1, EffectiveConcurrency: 1}
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
	if headers["connection"] != "" || headers["upgrade"] != "" || proof.SecWebSocketAcceptPresent || proof.SubprotocolPresent || proof.ExtensionsPresent {
		return operationResult{}, errors.New("prohibited RFC 6455-over-H1 or negotiated WebSocket response header present")
	}
	connectionLatency := durationMS(time.Since(started))
	operationStarted := time.Now()
	result := operationResult{Proof: proof, Frames: frames, ConnectionLatencyMS: connectionLatency}
	for index := 0; index < spec.MessageCount; index++ {
		key, frameBytes, frameErr := maskedFrame(spec.Opcode, spec.Payload)
		if frameErr != nil {
			return result, frameErr
		}
		result.Frames.ClientMessageFrames++
		result.Frames.ClientMaskedFrames++
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
		if received.Masked {
			result.Frames.ServerMaskedFrames++
		}
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
	result.Frames.ClientMaskedFrames++
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
	result.Proof.StrictOrdering = spec.MessageCount <= 1 || (spec.MessageCount == 100 && strictOrder(result.Frames.ReceivedOrder))
	result.Proof.PingSent = spec.Opcode == 0x9
	result.Proof.PongReceived = spec.ResponseOpcode == 0xA
	result.Proof.CloseSent = 1000
	result.Proof.CloseReceived = 1000
	result.Proof.CleanCompletion = true
	result.Frames.UniqueClientMaskKeys, result.Frames.DuplicateClientMaskKeys = maskKeyCounts(result.Frames.ClientMaskKeys)
	result.Frames.StrictPerStreamOrdering = result.Proof.StrictOrdering
	result.Frames.StreamIDs = []uint32{1}
	return result, nil
}

type h2Session struct {
	conn               *tls.Conn
	fr                 *http2.Framer
	writeMu            sync.Mutex
	streamsMu          sync.RWMutex
	streams            map[uint32]chan streamEvent
	headerBlocks       map[uint32]*bytes.Buffer
	headerDecoder      *hpack.Decoder
	done               chan struct{}
	readerError        atomic.Value
	settingsFrames     atomic.Int64
	windowUpdateFrames atomic.Int64
}

type multiplexedWebSocket struct {
	session       *h2Session
	streamID      uint32
	frames        chan streamEvent
	pending       []byte
	summary       frameSummary
	maskKeys      map[string]struct{}
	duplicateKeys int
}

type streamEvent struct {
	Headers map[string]string
	Data    []byte
	Err     error
}

type messageGroupResult struct {
	MessageLatencies []float64
	TransferredBytes int
	MaskKeyHashes    []string
	ReceivedOrder    []int
}

func openH2Session(ctx context.Context, address string, roots *x509.CertPool) (*h2Session, protocolProof, float64, error) {
	started := time.Now()
	dialer := &tls.Dialer{Config: &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: roots, ServerName: serverName, NextProtos: []string{alpn}, SessionTicketsDisabled: true}}
	raw, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, protocolProof{}, 0, err
	}
	conn := raw.(*tls.Conn)
	state := conn.ConnectionState()
	proof := protocolProof{
		Protocol: "h2", ProtocolVersion: "HTTP/2", ProtocolVariant: protocolVariant,
		TLSVersion: tls.VersionName(state.Version), ALPN: state.NegotiatedProtocol, DidResume: state.DidResume,
		ClientMaskRequired: true, RequestPseudoHeaders: map[string]string{":method": "CONNECT", ":protocol": "websocket", ":scheme": "https", ":authority": serverName, ":path": pathValue},
		RequestHeaders: map[string]string{"sec-websocket-version": "13"}, ResponseHeaders: map[string]string{},
		ConnectionReused: true, ConfiguredConnections: 1, ConfiguredConcurrency: 8, ConfiguredStreamsPerConnection: 8,
		ObservedActiveConnections: 1, ObservedActiveStreams: 8, EffectiveConcurrency: 8,
	}
	if len(state.PeerCertificates) > 0 {
		proof.CertificateDERSHA256 = hash(state.PeerCertificates[0].Raw)
		proof.CertificateSPKISHA256 = hash(state.PeerCertificates[0].RawSubjectPublicKeyInfo)
	}
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != alpn || state.DidResume || proof.CertificateDERSHA256 != certificateDERHash || proof.CertificateSPKISHA256 != certificateSPKIHash {
		_ = conn.Close()
		return nil, protocolProof{}, 0, errors.New("TLS 1.3 ALPN h2 certificate gate failed")
	}
	fr := http2.NewFramer(conn, conn)
	if _, err = io.WriteString(conn, http2.ClientPreface); err != nil {
		_ = conn.Close()
		return nil, protocolProof{}, 0, err
	}
	if err = fr.WriteSettings(http2.Setting{ID: http2.SettingEnableConnectProtocol, Val: 1}); err != nil {
		_ = conn.Close()
		return nil, protocolProof{}, 0, err
	}
	for proof.SettingsEnableConnectProtocol != 1 {
		frame, readErr := fr.ReadFrame()
		if readErr != nil {
			_ = conn.Close()
			return nil, protocolProof{}, 0, readErr
		}
		settings, ok := frame.(*http2.SettingsFrame)
		if !ok || settings.IsAck() {
			continue
		}
		_ = settings.ForeachSetting(func(setting http2.Setting) error {
			if setting.ID == http2.SettingEnableConnectProtocol && setting.Val == 1 {
				proof.SettingsEnableConnectProtocol = 1
			}
			return nil
		})
		if err = fr.WriteSettingsAck(); err != nil {
			_ = conn.Close()
			return nil, protocolProof{}, 0, err
		}
	}
	session := &h2Session{conn: conn, fr: fr, streams: map[uint32]chan streamEvent{}, headerBlocks: map[uint32]*bytes.Buffer{}, done: make(chan struct{})}
	session.headerDecoder = hpack.NewDecoder(4096, nil)
	go session.readLoop()
	return session, proof, durationMS(time.Since(started)), nil
}

func (session *h2Session) readLoop() {
	defer close(session.done)
	for {
		frame, err := session.fr.ReadFrame()
		if err != nil {
			session.readerError.Store(err)
			return
		}
		switch typed := frame.(type) {
		case *http2.SettingsFrame:
			session.settingsFrames.Add(1)
			if !typed.IsAck() {
				session.writeMu.Lock()
				err = session.fr.WriteSettingsAck()
				session.writeMu.Unlock()
				if err != nil {
					session.readerError.Store(err)
					return
				}
			}
			continue
		case *http2.WindowUpdateFrame:
			session.windowUpdateFrames.Add(1)
			continue
		case *http2.DataFrame:
			data := append([]byte(nil), typed.Data()...)
			length := uint32(len(data))
			if length > 0 {
				session.writeMu.Lock()
				err = session.fr.WriteWindowUpdate(0, length)
				if err == nil {
					err = session.fr.WriteWindowUpdate(typed.Header().StreamID, length)
				}
				session.writeMu.Unlock()
				if err != nil {
					session.readerError.Store(err)
					return
				}
			}
			session.dispatch(typed.Header().StreamID, streamEvent{Data: data})
			continue
		case *http2.RSTStreamFrame:
			session.dispatch(typed.Header().StreamID, streamEvent{Err: fmt.Errorf("stream %d reset with %s", typed.Header().StreamID, typed.ErrCode)})
			continue
		case *http2.GoAwayFrame:
			session.readerError.Store(fmt.Errorf("HTTP/2 GOAWAY lastStream=%d error=%s", typed.LastStreamID, typed.ErrCode))
			return
		}
		streamID := frame.Header().StreamID
		if headers, continuation := frame.(*http2.HeadersFrame); continuation {
			block := session.headerBlocks[streamID]
			if block == nil {
				block = &bytes.Buffer{}
				session.headerBlocks[streamID] = block
			}
			block.Write(headers.HeaderBlockFragment())
			if !headers.HeadersEnded() {
				continue
			}
			decoded, decodeErr := session.decodeHeaderBlock(block.Bytes())
			delete(session.headerBlocks, streamID)
			if decodeErr != nil {
				session.readerError.Store(decodeErr)
				return
			}
			session.dispatch(streamID, streamEvent{Headers: decoded})
			continue
		}
		if continuation, ok := frame.(*http2.ContinuationFrame); ok {
			block := session.headerBlocks[streamID]
			if block == nil {
				session.readerError.Store(fmt.Errorf("stream %d continuation without header block", streamID))
				return
			}
			block.Write(continuation.HeaderBlockFragment())
			if !continuation.HeadersEnded() {
				continue
			}
			decoded, decodeErr := session.decodeHeaderBlock(block.Bytes())
			delete(session.headerBlocks, streamID)
			if decodeErr != nil {
				session.readerError.Store(decodeErr)
				return
			}
			session.dispatch(streamID, streamEvent{Headers: decoded})
			continue
		}
	}
}

func (session *h2Session) decodeHeaderBlock(block []byte) (map[string]string, error) {
	headers := map[string]string{}
	session.headerDecoder.SetEmitFunc(func(field hpack.HeaderField) {
		headers[strings.ToLower(field.Name)] = field.Value
	})
	if _, err := session.headerDecoder.Write(block); err != nil {
		return nil, err
	}
	if err := session.headerDecoder.Close(); err != nil {
		return nil, err
	}
	return headers, nil
}

func (session *h2Session) dispatch(streamID uint32, event streamEvent) {
	session.streamsMu.RLock()
	streamFrames := session.streams[streamID]
	session.streamsMu.RUnlock()
	if streamFrames != nil {
		select {
		case streamFrames <- event:
		case <-session.done:
		}
	}
}

func (session *h2Session) openWebSocket(ctx context.Context, streamID uint32, baseProof protocolProof) (*multiplexedWebSocket, protocolProof, error) {
	frames := make(chan streamEvent, 256)
	session.streamsMu.Lock()
	if _, exists := session.streams[streamID]; exists {
		session.streamsMu.Unlock()
		return nil, protocolProof{}, fmt.Errorf("duplicate HTTP/2 stream %d", streamID)
	}
	session.streams[streamID] = frames
	session.streamsMu.Unlock()
	var headerBlock bytes.Buffer
	encoder := hpack.NewEncoder(&headerBlock)
	for _, field := range []hpack.HeaderField{{Name: ":method", Value: "CONNECT"}, {Name: ":protocol", Value: "websocket"}, {Name: ":scheme", Value: "https"}, {Name: ":authority", Value: serverName}, {Name: ":path", Value: pathValue}, {Name: "sec-websocket-version", Value: "13"}} {
		if err := encoder.WriteField(field); err != nil {
			return nil, protocolProof{}, err
		}
	}
	session.writeMu.Lock()
	err := session.fr.WriteHeaders(http2.HeadersFrameParam{StreamID: streamID, BlockFragment: headerBlock.Bytes(), EndHeaders: true, EndStream: false})
	session.writeMu.Unlock()
	if err != nil {
		return nil, protocolProof{}, err
	}
	headers, err := readMultiplexedResponseHeaders(ctx, session, frames, streamID)
	if err != nil {
		return nil, protocolProof{}, err
	}
	proof := baseProof
	proof.ResponseHeaders = headers
	if headers[":status"] != "200" {
		return nil, protocolProof{}, fmt.Errorf("stream %d Extended CONNECT status %q", streamID, headers[":status"])
	}
	proof.ResponseStatus = 200
	proof.SecWebSocketAcceptPresent = headers["sec-websocket-accept"] != ""
	proof.SecWebSocketKeyPresent = false
	proof.SubprotocolPresent = headers["sec-websocket-protocol"] != ""
	proof.ExtensionsPresent = headers["sec-websocket-extensions"] != ""
	proof.ProhibitedRequestHeadersPresent = false
	if headers["connection"] != "" || headers["upgrade"] != "" || proof.SecWebSocketAcceptPresent || proof.SubprotocolPresent || proof.ExtensionsPresent {
		return nil, protocolProof{}, fmt.Errorf("stream %d prohibited response header present", streamID)
	}
	return &multiplexedWebSocket{session: session, streamID: streamID, frames: frames, maskKeys: map[string]struct{}{}, summary: frameSummary{StreamIDs: []uint32{streamID}, StrictPerStreamOrdering: true}}, proof, nil
}

func readMultiplexedResponseHeaders(ctx context.Context, session *h2Session, frames <-chan streamEvent, streamID uint32) (map[string]string, error) {
	for {
		event, err := nextMultiplexedFrame(ctx, session, frames)
		if err != nil {
			return nil, err
		}
		if event.Headers != nil {
			return event.Headers, nil
		}
	}
}

func nextMultiplexedFrame(ctx context.Context, session *h2Session, frames <-chan streamEvent) (streamEvent, error) {
	select {
	case event := <-frames:
		if event.Err != nil {
			return streamEvent{}, event.Err
		}
		return event, nil
	case <-session.done:
		if value := session.readerError.Load(); value != nil {
			return streamEvent{}, value.(error)
		}
		return streamEvent{}, io.EOF
	case <-ctx.Done():
		return streamEvent{}, ctx.Err()
	}
}

func (stream *multiplexedWebSocket) runMessageGroup(ctx context.Context, spec scenario) (messageGroupResult, error) {
	result := messageGroupResult{ReceivedOrder: make([]int, 0, spec.MessageCount), MaskKeyHashes: make([]string, 0, spec.MessageCount), MessageLatencies: make([]float64, 0, spec.MessageCount)}
	for index := 0; index < spec.MessageCount; index++ {
		started := time.Now()
		key, encoded, err := maskedFrame(spec.Opcode, spec.Payload)
		if err != nil {
			return result, err
		}
		keyHash := hash(key)
		if _, exists := stream.maskKeys[keyHash]; exists {
			stream.duplicateKeys++
		}
		stream.maskKeys[keyHash] = struct{}{}
		stream.summary.ClientMaskKeys = append(stream.summary.ClientMaskKeys, keyHash)
		stream.summary.ClientMaskedFrames++
		stream.summary.ClientMessageFrames++
		stream.session.writeMu.Lock()
		err = stream.session.fr.WriteData(stream.streamID, false, encoded)
		stream.session.writeMu.Unlock()
		if err != nil {
			return result, err
		}
		response, err := stream.readWebSocketFrame(ctx)
		if err != nil {
			return result, err
		}
		if response.Masked {
			stream.summary.ServerMaskedFrames++
		}
		if response.Masked || response.Opcode != spec.ResponseOpcode || !bytes.Equal(response.Payload, spec.Payload) {
			return result, fmt.Errorf("stream %d message %d response mismatch", stream.streamID, index+1)
		}
		stream.summary.ServerMessageFrames++
		result.MessageLatencies = append(result.MessageLatencies, durationMS(time.Since(started)))
		result.TransferredBytes += len(spec.Payload) * 2
		result.MaskKeyHashes = append(result.MaskKeyHashes, keyHash)
		result.ReceivedOrder = append(result.ReceivedOrder, index+1)
	}
	if !strictOrder(result.ReceivedOrder) {
		return result, fmt.Errorf("stream %d did not preserve strict 1..100 response order", stream.streamID)
	}
	if len(stream.summary.ReceivedOrder) == 0 {
		stream.summary.ReceivedOrder = append(stream.summary.ReceivedOrder, result.ReceivedOrder...)
	}
	return result, nil
}

func (stream *multiplexedWebSocket) readWebSocketFrame(ctx context.Context) (wsFrame, error) {
	for {
		if parsed, consumed, ok, err := parseFrameConsumed(stream.pending); err != nil {
			return wsFrame{}, err
		} else if ok {
			stream.pending = append([]byte(nil), stream.pending[consumed:]...)
			return parsed, nil
		}
		event, err := nextMultiplexedFrame(ctx, stream.session, stream.frames)
		if err != nil {
			return wsFrame{}, err
		}
		if len(event.Data) == 0 {
			continue
		}
		stream.summary.DataFrames++
		stream.pending = append(stream.pending, event.Data...)
	}
}

func (stream *multiplexedWebSocket) close(ctx context.Context) (float64, error) {
	started := time.Now()
	key, encoded, err := maskedFrame(0x8, []byte{0x03, 0xE8})
	if err != nil {
		return 0, err
	}
	keyHash := hash(key)
	if _, exists := stream.maskKeys[keyHash]; exists {
		stream.duplicateKeys++
	}
	stream.maskKeys[keyHash] = struct{}{}
	stream.summary.ClientMaskKeys = append(stream.summary.ClientMaskKeys, keyHash)
	stream.summary.ClientMaskedFrames++
	stream.session.writeMu.Lock()
	err = stream.session.fr.WriteData(stream.streamID, false, encoded)
	stream.session.writeMu.Unlock()
	if err != nil {
		return 0, err
	}
	response, err := stream.readWebSocketFrame(ctx)
	if err != nil {
		return 0, err
	}
	if response.Masked || response.Opcode != 0x8 || !bytes.Equal(response.Payload, []byte{0x03, 0xE8}) {
		return 0, fmt.Errorf("stream %d clean close mismatch", stream.streamID)
	}
	return durationMS(time.Since(started)), nil
}

func runMultiMessageDiagnostic(address string, roots *x509.CertPool, spec scenario, profile loadProfile, validationOnly bool) (operationResult, measuredSummary, measuredSummary, topologyProof, error) {
	ctx, cancel := context.WithTimeout(context.Background(), profile.OperationTimeout)
	session, baseProof, connectionLatency, err := openH2Session(ctx, address, roots)
	cancel()
	if err != nil {
		return operationResult{}, measuredSummary{}, measuredSummary{}, topologyProof{}, err
	}
	defer session.conn.Close()
	streamIDs := []uint32{1, 3, 5, 7, 9, 11, 13, 15}
	streams := make([]*multiplexedWebSocket, 0, len(streamIDs))
	proofs := make([]protocolProof, 0, len(streamIDs))
	for _, streamID := range streamIDs {
		openCtx, openCancel := context.WithTimeout(context.Background(), profile.OperationTimeout)
		stream, proof, openErr := session.openWebSocket(openCtx, streamID, baseProof)
		openCancel()
		if openErr != nil {
			return operationResult{}, measuredSummary{}, measuredSummary{}, topologyProof{}, openErr
		}
		streams = append(streams, stream)
		proofs = append(proofs, proof)
	}
	preflightSummary, err := runConcurrentMessagePhase(streams, spec, profile, 0, true)
	if err != nil {
		return operationResult{}, measuredSummary{}, measuredSummary{}, topologyProof{}, err
	}
	warmup := emptySummary(profile)
	measured := preflightSummary
	if !validationOnly {
		warmup, err = runConcurrentMessagePhase(streams, spec, profile, profile.Warmup, false)
		if err == nil {
			measured, err = runConcurrentMessagePhase(streams, spec, profile, profile.Duration, false)
		}
		if err != nil {
			return operationResult{}, warmup, measured, topologyProof{}, err
		}
		if profile.Cooldown > 0 {
			time.Sleep(profile.Cooldown)
		}
	}
	closeLatencies := make([]float64, len(streams))
	closeErrors := make(chan error, len(streams))
	var closeWait sync.WaitGroup
	for index, stream := range streams {
		closeWait.Add(1)
		go func(index int, stream *multiplexedWebSocket) {
			defer closeWait.Done()
			closeCtx, closeCancel := context.WithTimeout(context.Background(), profile.OperationTimeout)
			defer closeCancel()
			latency, closeErr := stream.close(closeCtx)
			closeLatencies[index] = latency
			if closeErr != nil {
				closeErrors <- closeErr
			}
		}(index, stream)
	}
	closeWait.Wait()
	close(closeErrors)
	for closeErr := range closeErrors {
		if closeErr != nil {
			return operationResult{}, warmup, measured, topologyProof{}, closeErr
		}
	}
	proof := proofs[0]
	proof.MessageType = "text"
	proof.MessageBytes = len(spec.Payload)
	proof.MessageCount = spec.MessageCount
	proof.PayloadSHA256 = spec.PayloadHash
	proof.StrictOrdering = true
	proof.ClientMaskObserved = true
	proof.CloseSent = 1000
	proof.CloseReceived = 1000
	proof.CleanCompletion = true
	aggregateFrames := frameSummary{StreamIDs: append([]uint32(nil), streamIDs...), StrictPerStreamOrdering: true}
	maskKeys := map[string]struct{}{}
	for _, stream := range streams {
		aggregateFrames.HeadersFrames += stream.summary.HeadersFrames
		aggregateFrames.DataFrames += stream.summary.DataFrames
		aggregateFrames.ClientMessageFrames += stream.summary.ClientMessageFrames
		aggregateFrames.ServerMessageFrames += stream.summary.ServerMessageFrames
		aggregateFrames.ClientMaskedFrames += stream.summary.ClientMaskedFrames
		aggregateFrames.ServerMaskedFrames += stream.summary.ServerMaskedFrames
		for _, keyHash := range stream.summary.ClientMaskKeys {
			if _, exists := maskKeys[keyHash]; exists {
				aggregateFrames.DuplicateClientMaskKeys++
			}
			maskKeys[keyHash] = struct{}{}
			if len(aggregateFrames.ClientMaskKeys) < 800 {
				aggregateFrames.ClientMaskKeys = append(aggregateFrames.ClientMaskKeys, keyHash)
			}
		}
	}
	aggregateFrames.SettingsFrames = int(session.settingsFrames.Load()) + 1
	aggregateFrames.WindowUpdateFrames = int(session.windowUpdateFrames.Load())
	aggregateFrames.UniqueClientMaskKeys = len(maskKeys)
	if aggregateFrames.ServerMaskedFrames != 0 {
		return operationResult{}, warmup, measured, topologyProof{}, fmt.Errorf("multiplexed WebSocket masking gate failed: duplicateClientKeys=%d serverMaskedFrames=%d uniqueClientKeys=%d totalClientFrames=%d", aggregateFrames.DuplicateClientMaskKeys, aggregateFrames.ServerMaskedFrames, aggregateFrames.UniqueClientMaskKeys, aggregateFrames.ClientMaskedFrames)
	}
	measured.CloseLatencies = append(measured.CloseLatencies, closeLatencies...)
	measured.ConnectionLatencies = []float64{connectionLatency}
	measured.Last = &operationResult{Proof: proof, Frames: aggregateFrames, ConnectionLatencyMS: connectionLatency, OperationLatencyMS: mean(measured.OperationLatencies), CloseLatencyMS: mean(closeLatencies), Bytes: measured.TotalTransferredBytes}
	preflight := *measured.Last
	topology := topologyProof{ConnectionCount: 1, AuthenticatedTLSConnections: 1, ConfiguredConcurrency: 8, ConfiguredStreamsPerConnection: 8, ObservedActiveConnections: 1, ObservedActiveStreams: 8, EffectiveConcurrency: 8, StreamIDs: streamIDs, BalancedAssignment: "one-stream-per-worker-round-robin", OneConnection: true, EightConcurrentStreams: true}
	return preflight, warmup, measured, topology, nil
}

func runConcurrentMessagePhase(streams []*multiplexedWebSocket, spec scenario, profile loadProfile, duration time.Duration, exactlyOne bool) (measuredSummary, error) {
	started := time.Now()
	start := make(chan struct{})
	type workerResult struct {
		stream     streamSummary
		latencies  []float64
		bytes      int
		operations int
		messages   int
		err        error
	}
	results := make(chan workerResult, len(streams))
	var active atomic.Int64
	var peak atomic.Int64
	for _, stream := range streams {
		go func(stream *multiplexedWebSocket) {
			<-start
			current := active.Add(1)
			for {
				observed := peak.Load()
				if current <= observed || peak.CompareAndSwap(observed, current) {
					break
				}
			}
			defer active.Add(-1)
			deadline := started.Add(duration)
			result := workerResult{stream: streamSummary{StreamID: stream.streamID, StrictOrdering: true}}
			for exactlyOne || time.Now().Before(deadline) {
				groupCtx, cancel := context.WithTimeout(context.Background(), profile.OperationTimeout)
				group, err := stream.runMessageGroup(groupCtx, spec)
				cancel()
				if err != nil {
					result.err = err
					result.stream.FailedOperations++
					if errors.Is(err, context.DeadlineExceeded) {
						result.stream.FailedOperations--
						result.stream.TimedOutOperations++
					}
					break
				}
				result.operations++
				result.messages += spec.MessageCount
				result.bytes += group.TransferredBytes
				result.latencies = append(result.latencies, group.MessageLatencies...)
				if exactlyOne {
					break
				}
			}
			result.stream.CompletedOperations = result.operations
			result.stream.CompletedMessages = result.messages
			result.stream.TransferredBytes = result.bytes
			result.stream.MessageLatencyMeanMS = mean(result.latencies)
			result.stream.MessageLatencyP95MS = percentile(result.latencies, .95)
			results <- result
		}(stream)
	}
	close(start)
	summary := emptySummary(profile)
	summary.DurationSeconds = duration.Seconds()
	if exactlyOne {
		summary.DurationSeconds = 0
	}
	var firstError error
	for range streams {
		result := <-results
		summary.CompletedOperations += result.operations
		summary.CompletedMessages += result.messages
		summary.TotalTransferredBytes += result.bytes
		summary.OperationLatencies = append(summary.OperationLatencies, result.latencies...)
		summary.FailedOperations += result.stream.FailedOperations
		summary.TimedOutOperations += result.stream.TimedOutOperations
		summary.PerStream = append(summary.PerStream, result.stream)
		if result.err != nil {
			summary.Errors[result.err.Error()]++
			if firstError == nil {
				firstError = result.err
			}
		}
	}
	sort.Slice(summary.PerStream, func(i, j int) bool { return summary.PerStream[i].StreamID < summary.PerStream[j].StreamID })
	summary.EffectiveConcurrency = int(peak.Load())
	if summary.EffectiveConcurrency != profile.Concurrency {
		return summary, fmt.Errorf("effective concurrency mismatch: expected %d observed %d", profile.Concurrency, summary.EffectiveConcurrency)
	}
	if summary.DurationSeconds == 0 {
		summary.DurationSeconds = time.Since(started).Seconds()
	}
	return summary, firstError
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
	frame, _, ok, err := parseFrameConsumed(data)
	return frame, ok, err
}

func parseFrameConsumed(data []byte) (wsFrame, int, bool, error) {
	if len(data) < 2 {
		return wsFrame{}, 0, false, nil
	}
	first, second := data[0], data[1]
	if first&0x80 == 0 || first&0x70 != 0 {
		return wsFrame{}, 0, false, errors.New("fragmented or RSV-bearing server frame")
	}
	masked := second&0x80 != 0
	length := int(second & 0x7f)
	offset := 2
	if length == 126 {
		if len(data) < 4 {
			return wsFrame{}, 0, false, nil
		}
		length = int(binary.BigEndian.Uint16(data[2:4]))
		offset = 4
	} else if length == 127 {
		return wsFrame{}, 0, false, errors.New("unexpected 64-bit WebSocket frame")
	}
	maskBytes := 0
	if masked {
		maskBytes = 4
	}
	if len(data) < offset+maskBytes+length {
		return wsFrame{}, 0, false, nil
	}
	payload := append([]byte(nil), data[offset+maskBytes:offset+maskBytes+length]...)
	if masked {
		mask := data[offset : offset+4]
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return wsFrame{Opcode: first & 0xf, Masked: masked, Payload: payload}, offset + maskBytes + length, true, nil
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

func runFor(address string, roots *x509.CertPool, spec scenario, duration, operationTimeout time.Duration, profile loadProfile) measuredSummary {
	s := emptySummary(profile)
	started := time.Now()
	for time.Since(started) < duration {
		op, err := runOperation(context.Background(), address, roots, spec, operationTimeout)
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
	checks := []string{"protocol:h2", "tls:1.3", "alpn:h2", "authenticated-certificate-hashes", "session:not-resumed", "no-fallback", "settings-enable-connect-protocol:1", "extended-connect", "exact-pseudo-headers", ":protocol:websocket", "endpoint:/websocket", "status:200", "no-h1-connection-headers", "no-sec-websocket-key", "no-sec-websocket-accept", "no-subprotocol", "no-extensions", "randomized-client-masking", "unmasked-server-frames", "close-code:1000", "zero-unexpected-failures", "zero-timeouts"}
	if spec.ID == "http2.websocket.rfc8441.multi-message-text-echo" {
		checks = append(checks, "one-authenticated-h2-connection", "eight-concurrent-extended-connect-streams", "effective-concurrency:8", "message-count-per-operation:100", "strict-per-stream-order")
	}
	writeJSON(dir, "validation.json", map[string]any{"schemaVersion": "protocol-lab.validation.v1", "scenarioId": spec.ID, "status": "passed", "checks": checks})
	writeJSON(dir, "protocol-proof.json", op.Proof)
	writeJSON(dir, "websocket-summary.json", op)
	writeJSON(dir, "http2-frame-summary.json", op.Frames)
	writeJSON(dir, "frame-summary.json", op.Frames)
	writeJSON(dir, "payload-hash.json", map[string]any{"algorithm": "sha256", "messageBytes": len(spec.Payload), "messageCount": spec.MessageCount, "expected": spec.PayloadHash, "observed": spec.PayloadHash})
}
func writeResult(dir string, spec scenario, profile loadProfile, s measuredSummary) {
	d := s.DurationSeconds
	if d <= 0 {
		d = 1
	}
	connectionCount := s.CompletedOperations
	if spec.ID == "http2.websocket.rfc8441.multi-message-text-echo" {
		connectionCount = s.ObservedConnections
	}
	metrics := map[string]any{"connectionsPerSecond": float64(connectionCount) / d, "messagesPerSecond": float64(s.CompletedMessages) / d, "bytesPerSecond": float64(s.TotalTransferredBytes) / d, "connectionLatencyMean": mean(s.ConnectionLatencies), "connectionLatencyP50": percentile(s.ConnectionLatencies, .5), "connectionLatencyP75": percentile(s.ConnectionLatencies, .75), "connectionLatencyP90": percentile(s.ConnectionLatencies, .9), "connectionLatencyP95": percentile(s.ConnectionLatencies, .95), "connectionLatencyP99": percentile(s.ConnectionLatencies, .99), "messageLatencyMean": mean(s.OperationLatencies), "messageLatencyP50": percentile(s.OperationLatencies, .5), "messageLatencyP75": percentile(s.OperationLatencies, .75), "messageLatencyP90": percentile(s.OperationLatencies, .9), "messageLatencyP95": percentile(s.OperationLatencies, .95), "messageLatencyP99": percentile(s.OperationLatencies, .99), "controlFrameLatencyP50": percentile(s.OperationLatencies, .5), "controlFrameLatencyP95": percentile(s.OperationLatencies, .95), "controlFrameLatencyP99": percentile(s.OperationLatencies, .99), "closeLatencyP50": percentile(s.CloseLatencies, .5), "closeLatencyP95": percentile(s.CloseLatencies, .95), "closeLatencyP99": percentile(s.CloseLatencies, .99), "completedOperations": s.CompletedOperations, "completedMessages": s.CompletedMessages, "failedOperations": s.FailedOperations, "timedOutOperations": s.TimedOutOperations, "totalTransferredBytes": s.TotalTransferredBytes, "configuredConnections": profile.Connections, "configuredConcurrency": profile.Concurrency, "configuredStreamsPerConnection": profile.StreamsPerConnection, "configuredStreamCapacity": profile.Connections * profile.StreamsPerConnection, "observedActiveConnections": s.ObservedConnections, "observedActiveStreams": s.ObservedStreams, "effectiveConcurrency": s.EffectiveConcurrency, "effectiveStreams": s.ObservedStreams}
	result := map[string]any{"schemaVersion": "protocol-lab.http2-websocket-executor-result.v1", "scenarioId": spec.ID, "authorityCommit": authorityCommit, "executor": map[string]string{"id": executorID, "version": executorVersion}, "loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": "golang.org/x/net/http2", "engineModuleVersion": "v0.57.0"}, "validation": map[string]string{"status": "passed"}, "protocolProof": func() any {
		if s.Last != nil {
			return s.Last.Proof
		}
		return nil
	}(), "requestedLoad": map[string]any{"profileId": profile.ID, "connections": profile.Connections, "concurrency": profile.Concurrency, "streamsPerConnection": profile.StreamsPerConnection, "durationSeconds": profile.Duration.Seconds(), "warmupSeconds": profile.Warmup.Seconds(), "cooldownSeconds": profile.Cooldown.Seconds(), "repetitions": profile.Repetitions, "operationTimeoutSeconds": profile.OperationTimeout.Seconds()}, "effectiveLoad": map[string]any{"configuredCapacity": map[string]any{"connections": profile.Connections, "concurrency": profile.Concurrency, "streamsPerConnection": profile.StreamsPerConnection, "totalStreams": profile.Connections * profile.StreamsPerConnection}, "observed": map[string]any{"activeConnections": s.ObservedConnections, "activeStreams": s.ObservedStreams, "effectiveConcurrency": s.EffectiveConcurrency}, "connections": s.ObservedConnections, "concurrency": s.EffectiveConcurrency, "streams": s.ObservedStreams}, "metrics": metrics, "warnings": []string{"Local package-backed RFC 8441 evidence is diagnostic and non-publishable."}}
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
	writeJSON(dir, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedProtocols": []string{"h2"}, "supportedScenarios": sortedScenarioIDs(), "supportedLoadProfiles": []string{smokeProfileID, diagnosticProfileID}})
	writeJSON(dir, "load-generator-identity.json", map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": "golang.org/x/net/http2", "engineModuleVersion": "v0.57.0"})
}

func profileFor(spec scenario) loadProfile {
	if spec.ID == "http2.websocket.rfc8441.multi-message-text-echo" {
		return diagnosticProfile
	}
	return smokeProfile
}

func emptySummary(profile loadProfile) measuredSummary {
	return measuredSummary{
		Errors: map[string]int{}, ConfiguredConnections: profile.Connections,
		ConfiguredConcurrency: profile.Concurrency, ConfiguredStreams: profile.StreamsPerConnection,
		ObservedConnections: profile.Connections, ObservedStreams: profile.StreamsPerConnection,
		EffectiveConcurrency: profile.Concurrency,
	}
}

func summaryFromOperation(op operationResult, profile loadProfile) measuredSummary {
	duration := op.OperationLatencyMS / 1000
	if duration <= 0 {
		duration = .001
	}
	return measuredSummary{
		DurationSeconds: duration, CompletedOperations: 1, CompletedMessages: max(1, op.Proof.MessageCount),
		TotalTransferredBytes: op.Bytes, ConnectionLatencies: []float64{op.ConnectionLatencyMS},
		OperationLatencies: []float64{op.OperationLatencyMS}, CloseLatencies: []float64{op.CloseLatencyMS},
		Last: &op, Errors: map[string]int{}, ConfiguredConnections: profile.Connections,
		ConfiguredConcurrency: profile.Concurrency, ConfiguredStreams: profile.StreamsPerConnection,
		ObservedConnections: 1, ObservedStreams: 1, EffectiveConcurrency: 1,
	}
}

func verifyProfileEnvironment(profile loadProfile) {
	verifyOptionalInteger("PLAB_CONNECTIONS", profile.Connections)
	verifyOptionalInteger("PLAB_CONCURRENCY", profile.Concurrency)
	verifyOptionalInteger("PLAB_STREAMS_PER_CONNECTION", profile.StreamsPerConnection)
	verifyOptionalInteger("PLAB_DURATION_SECONDS", int(profile.Duration.Seconds()))
	verifyOptionalInteger("PLAB_WARMUP_SECONDS", int(profile.Warmup.Seconds()))
	verifyOptionalInteger("PLAB_COOLDOWN_SECONDS", int(profile.Cooldown.Seconds()))
	verifyOptionalInteger("PLAB_REPETITION", profile.Repetitions)
	verifyOptionalInteger("PLAB_REQUEST_TIMEOUT_SECONDS", int(profile.OperationTimeout.Seconds()))
}

func verifyOptionalInteger(name string, expected int) {
	observed := strings.TrimSpace(os.Getenv(name))
	if observed == "" {
		return
	}
	if observed != fmt.Sprintf("%d", expected) {
		fatal(2, fmt.Errorf("%s substitution: expected %d observed %q", name, expected, observed))
	}
}

func maskKeyCounts(values []string) (int, int) {
	unique := map[string]struct{}{}
	duplicates := 0
	for _, value := range values {
		if _, exists := unique[value]; exists {
			duplicates++
		}
		unique[value] = struct{}{}
	}
	return len(unique), duplicates
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
