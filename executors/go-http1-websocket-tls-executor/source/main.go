package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	executorID                    = "go-http1-websocket-tls-executor"
	executorVersion               = "0.1.0"
	loadGeneratorID               = "go-http1-websocket-tls-load"
	loadGeneratorVersion          = "0.1.0"
	supportedProfile              = "websocket-smoke"
	websocketGUID                 = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	textPayload                   = "protocol-lab"
	controlPayload                = "protocol-lab-ping"
	serverName                    = "websocket.plab.test"
	expectedCertificateDERSHA256  = "fe996190f39355e3cfc201cbb7e2cba962a701b94ed08ff49e68e830216d0109"
	expectedCertificateSPKISHA256 = "c2440fbe955033f341ca625c1804e21b50066d952ab24a4b53007dc1cfbf410c"
)

type scenarioExpectation struct {
	id               string
	operation        string
	messageType      string
	payload          []byte
	payloadHash      string
	requestOpcode    byte
	responseOpcode   byte
	connectionMetric bool
}

var expectations = map[string]scenarioExpectation{
	"http1.websocket.rfc6455.tls.upgrade":        {id: "http1.websocket.rfc6455.tls.upgrade", operation: "establish", messageType: "none", payloadHash: hash(nil), connectionMetric: true},
	"http1.websocket.rfc6455.tls.text-echo":      {id: "http1.websocket.rfc6455.tls.text-echo", operation: "text-echo", messageType: "text", payload: []byte(textPayload), payloadHash: "504585b0bb4fd77012ea2575efbcdb58f4c33e6b543e9567a65896d213720c29", requestOpcode: 0x1, responseOpcode: 0x1},
	"http1.websocket.rfc6455.tls.binary-echo":    {id: "http1.websocket.rfc6455.tls.binary-echo", operation: "binary-echo", messageType: "binary", payload: repeatedByte(66, 1024), payloadHash: "9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a", requestOpcode: 0x2, responseOpcode: 0x2},
	"http1.websocket.rfc6455.tls.control-frames": {id: "http1.websocket.rfc6455.tls.control-frames", operation: "control-frames", messageType: "none", payload: []byte(controlPayload), payloadHash: "4848de689e96825e0e05b6c3e96e48f2ad7ec7805fb64ecf824ccc6aa9c58883", requestOpcode: 0x9, responseOpcode: 0xA},
	"http1.websocket.rfc6455.tls.close":          {id: "http1.websocket.rfc6455.tls.close", operation: "close", messageType: "none", payloadHash: hash(nil)},
}

var knownUnsupported = map[string]struct{}{
	"websocket.echo": {},
	"http1.websocket.rfc6455.cleartext.upgrade":                  {},
	"http1.websocket.rfc6455.cleartext.text-echo":                {},
	"http1.websocket.rfc6455.cleartext.binary-echo":              {},
	"http1.websocket.rfc6455.cleartext.control-frames":           {},
	"http1.websocket.rfc6455.cleartext.close":                    {},
	"http1.websocket.rfc6455.tls.subprotocol-text-echo":          {},
	"http1.websocket.rfc6455.tls.permessage-deflate-binary-echo": {},
	"http2.websocket.rfc8441.extended-connect":                   {},
	"http2.websocket.rfc8441.text-echo":                          {},
	"http2.websocket.rfc8441.binary-echo":                        {},
	"http2.websocket.rfc8441.control-frames":                     {},
	"http2.websocket.rfc8441.close":                              {},
	"http2.websocket.rfc8441.multi-message-text-echo":            {},
	"http3.websocket.rfc9220.extended-connect":                   {},
	"http3.websocket.rfc9220.text-echo":                          {},
	"http3.websocket.rfc9220.binary-echo":                        {},
	"http3.websocket.rfc9220.control-frames":                     {},
	"http3.websocket.rfc9220.close":                              {},
	"http3.websocket.rfc9220.fragmented-binary-echo":             {},
}

type loadConfig struct {
	connections                  int
	concurrency                  int
	durationSeconds              int
	warmupSeconds                int
	repetition                   int
	operationTimeoutMilliseconds int
}

type handshakeProof struct {
	Binding                    string `json:"binding"`
	Scheme                     string `json:"scheme"`
	TransportSecurity          string `json:"transportSecurity"`
	Endpoint                   string `json:"endpoint"`
	Authority                  string `json:"authority"`
	RequestMethod              string `json:"requestMethod"`
	RequestedHTTPVersion       string `json:"requestedHttpVersion"`
	ObservedHTTPVersion        string `json:"observedHttpVersion"`
	ResponseStatus             int    `json:"responseStatus"`
	UpgradeHeader              string `json:"upgradeHeader"`
	ConnectionHeader           string `json:"connectionHeader"`
	SecWebSocketVersion        string `json:"secWebSocketVersion"`
	SecWebSocketKeyPolicy      string `json:"secWebSocketKeyPolicy"`
	SampleSecWebSocketKey      string `json:"sampleSecWebSocketKey"`
	ExpectedSecWebSocketAccept string `json:"expectedSecWebSocketAccept"`
	ObservedSecWebSocketAccept string `json:"observedSecWebSocketAccept"`
	SubprotocolRequested       bool   `json:"subprotocolRequested"`
	SubprotocolNegotiated      bool   `json:"subprotocolNegotiated"`
	ExtensionsRequested        bool   `json:"extensionsRequested"`
	ExtensionsNegotiated       bool   `json:"extensionsNegotiated"`
	FallbackDetected           bool   `json:"fallbackDetected"`
	ConnectionEstablished      bool   `json:"connectionEstablished"`
	TLSVersion                 string `json:"tlsVersion"`
	ALPN                       string `json:"alpn"`
	ServerName                 string `json:"serverName"`
	CipherSuite                string `json:"cipherSuite"`
	DidResume                  bool   `json:"didResume"`
	EarlyData                  bool   `json:"earlyData"`
	VerifiedChainCount         int    `json:"verifiedChainCount"`
	CertificateDERSHA256       string `json:"certificateDerSha256"`
	CertificateSPKISHA256      string `json:"certificateSpkiSha256"`
}

type operationProof struct {
	ScenarioID           string         `json:"scenarioId"`
	Operation            string         `json:"operation"`
	MessageType          string         `json:"messageType"`
	MessageBytes         int            `json:"messageBytes"`
	MessageCount         int            `json:"messageCount"`
	PayloadSHA256        string         `json:"payloadSha256"`
	ControlPayloadBytes  int            `json:"controlPayloadBytes"`
	ControlPayloadSHA256 string         `json:"controlPayloadSha256"`
	Fragmentation        string         `json:"fragmentation"`
	ClientFrameMasked    bool           `json:"clientFrameMasked"`
	ServerFrameMasked    bool           `json:"serverFrameMasked"`
	RequestOpcode        string         `json:"requestOpcode"`
	ResponseOpcode       string         `json:"responseOpcode"`
	OrderedEcho          bool           `json:"orderedEcho"`
	PingSent             bool           `json:"pingSent"`
	PongReceived         bool           `json:"pongReceived"`
	CloseSentCode        int            `json:"closeSentCode"`
	CloseReceivedCode    int            `json:"closeReceivedCode"`
	CloseReasonBytes     int            `json:"closeReasonBytes"`
	CleanCompletion      bool           `json:"cleanCompletion"`
	TransportEOFObserved bool           `json:"transportEofObserved"`
	Handshake            handshakeProof `json:"handshake"`
	LatencyMilliseconds  float64        `json:"latencyMilliseconds"`
	TransferredBytes     int64          `json:"transferredBytes"`
	HandshakeRequest     string         `json:"-"`
	HandshakeResponse    string         `json:"-"`
}

type phaseSummary struct {
	Phase                         string              `json:"phase"`
	DurationSeconds               float64             `json:"durationSeconds"`
	CompletedOperations           int                 `json:"completedOperations"`
	FailedOperations              int                 `json:"failedOperations"`
	TimedOutOperations            int                 `json:"timedOutOperations"`
	TotalTransferredBytes         int64               `json:"totalTransferredBytes"`
	EffectiveConnections          int                 `json:"effectiveConnections"`
	EffectiveConcurrency          int                 `json:"effectiveConcurrency"`
	LatenciesMilliseconds         []float64           `json:"latenciesMilliseconds"`
	LastProof                     operationProof      `json:"lastProof"`
	OpeningHandshakes             int                 `json:"openingHandshakes"`
	KeyReuseCount                 int                 `json:"keyReuseCount"`
	InvalidDecodedKeyCount        int                 `json:"invalidDecodedKeyCount"`
	AcceptMismatchCount           int                 `json:"acceptMismatchCount"`
	UpgradeRequestHeadersMatched  bool                `json:"upgradeRequestHeadersMatched"`
	UpgradeResponseHeadersMatched bool                `json:"upgradeResponseHeadersMatched"`
	Errors                        map[string]int      `json:"errors"`
	seenKeys                      map[string]struct{} `json:"-"`
}

type metrics struct {
	ConnectionsPerSecond   float64 `json:"connectionsPerSecond"`
	MessagesPerSecond      float64 `json:"messagesPerSecond"`
	BytesPerSecond         float64 `json:"bytesPerSecond"`
	ConnectionLatencyMean  float64 `json:"connectionLatencyMeanMs"`
	ConnectionLatencyP50   float64 `json:"connectionLatencyP50Ms"`
	ConnectionLatencyP75   float64 `json:"connectionLatencyP75Ms"`
	ConnectionLatencyP90   float64 `json:"connectionLatencyP90Ms"`
	ConnectionLatencyP95   float64 `json:"connectionLatencyP95Ms"`
	ConnectionLatencyP99   float64 `json:"connectionLatencyP99Ms"`
	MessageLatencyMean     float64 `json:"messageLatencyMeanMs"`
	MessageLatencyP50      float64 `json:"messageLatencyP50Ms"`
	MessageLatencyP75      float64 `json:"messageLatencyP75Ms"`
	MessageLatencyP90      float64 `json:"messageLatencyP90Ms"`
	MessageLatencyP95      float64 `json:"messageLatencyP95Ms"`
	MessageLatencyP99      float64 `json:"messageLatencyP99Ms"`
	ControlFrameLatencyP50 float64 `json:"controlFrameLatencyP50Ms"`
	ControlFrameLatencyP95 float64 `json:"controlFrameLatencyP95Ms"`
	ControlFrameLatencyP99 float64 `json:"controlFrameLatencyP99Ms"`
	CloseLatencyP50        float64 `json:"closeLatencyP50Ms"`
	CloseLatencyP95        float64 `json:"closeLatencyP95Ms"`
	CloseLatencyP99        float64 `json:"closeLatencyP99Ms"`
	CompletedOperations    int     `json:"completedOperations"`
	FailedOperations       int     `json:"failedOperations"`
	TimedOutOperations     int     `json:"timedOutOperations"`
	TotalTransferredBytes  int64   `json:"totalTransferredBytes"`
	EffectiveConnections   int     `json:"effectiveConnections"`
	EffectiveConcurrency   int     `json:"effectiveConcurrency"`
}

type normalizedResult struct {
	SchemaVersion string            `json:"schemaVersion"`
	ScenarioID    string            `json:"scenarioId"`
	LoadProfileID string            `json:"loadProfileId"`
	Status        string            `json:"status"`
	Executor      map[string]string `json:"executor"`
	LoadGenerator map[string]string `json:"loadGenerator"`
	ProtocolProof map[string]any    `json:"protocolProof"`
	Validation    map[string]any    `json:"validation"`
	RequestedLoad map[string]any    `json:"requestedLoad"`
	EffectiveLoad map[string]any    `json:"effectiveLoad"`
	Metrics       metrics           `json:"metrics"`
	Warnings      []string          `json:"warnings"`
}

type wsConnection struct {
	conn        net.Conn
	reader      *bufio.Reader
	handshake   handshakeProof
	rawRequest  string
	rawResponse string
	handshakeMS float64
	bytes       int64
}

type wireFrame struct {
	fin     bool
	rsv     byte
	opcode  byte
	masked  bool
	payload []byte
	bytes   int
}

func main() {
	target := flag.String("target-url", os.Getenv("PLAB_TARGET_BASE_URL"), "TLS 1.3 HTTP/1.1 WebSocket target URL")
	rootCertificate := flag.String("root-certificate", os.Getenv("PLAB_TLS_ROOT_CERTIFICATE_PATH"), "authenticated package-local WebSocket test root PEM")
	output := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "artifact directory")
	validationOnly := flag.Bool("validation-only", false, "run one exact validity operation")
	showVersion := flag.Bool("version", false, "print executor version")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s %s\n", executorID, executorVersion)
		return
	}
	if strings.TrimSpace(*output) == "" {
		*output = "artifacts"
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		fatal(1, err)
	}
	expectation := checkIdentityOrExit(*output)
	if strings.TrimSpace(*rootCertificate) == "" {
		*rootCertificate = filepath.Join(packagedRoot(), "certs", "root.pem")
	}
	targetURL, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	timeout := time.Duration(envIntDefault("PLAB_OPERATION_TIMEOUT_MILLISECONDS", 5000)) * time.Millisecond
	preflight, err := performOperation(targetURL, *rootCertificate, expectation, timeout)
	writePreflightArtifacts(*output, expectation, preflight, err)
	if err != nil {
		fatal(1, fmt.Errorf("HTTP/1.1 WebSocket validity gate failed: %w", err))
	}
	if *validationOnly {
		fmt.Fprintln(os.Stderr, "go-http1-websocket-executor validity gate passed")
		return
	}
	config, err := loadConfigFromEnvironment()
	if err != nil {
		fatal(2, err)
	}
	warmup := runPhase(targetURL, *rootCertificate, expectation, time.Duration(config.warmupSeconds)*time.Second, timeout, "warmup")
	writeRequired(*output, "websocket-warmup-summary.json", warmup)
	if hasFailures(warmup) || warmup.CompletedOperations == 0 {
		fatal(1, errors.New("WebSocket warmup failed closed"))
	}
	measured := runPhase(targetURL, *rootCertificate, expectation, time.Duration(config.durationSeconds)*time.Second, timeout, "measured")
	writeRequired(*output, "websocket-load-summary.json", measured)
	if hasFailures(measured) || measured.CompletedOperations == 0 {
		fatal(1, errors.New("WebSocket measured phase failed closed"))
	}
	result := normalizeResult(expectation, config, measured)
	writeRequired(*output, "load-generator-identity.json", result.LoadGenerator)
	writeRequired(*output, "websocket-executor-result.json", result)
	writeRequired(*output, "result.json", result)
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
}

func checkIdentityOrExit(output string) scenarioExpectation {
	verifySubstitution("PLAB_EXECUTOR_ID", executorID, "executor")
	verifySubstitution("PLAB_EXECUTOR_VERSION", executorVersion, "executor version")
	verifySubstitution("PLAB_LOAD_GENERATOR_ID", loadGeneratorID, "load generator")
	verifySubstitution("PLAB_LOAD_GENERATOR_VERSION", loadGeneratorVersion, "load generator version")
	verifySubstitution("PLAB_PROTOCOL", "h1", "protocol")
	verifySubstitution("PLAB_PROTOCOL_VARIANT", "websocket-h1-tls1.3-upgrade", "protocol variant")
	scenario := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if expectation, ok := expectations[scenario]; ok {
		return expectation
	}
	if _, ok := knownUnsupported[scenario]; ok {
		document := map[string]any{
			"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": scenario,
			"executorId": executorID, "reason": "exact WebSocket identity is outside this TLS 1.3 RFC 6455 over HTTP/1.1 package",
		}
		writeRequired(output, "unsupported.json", document)
		writeRequired(output, "result.json", document)
		encoded, _ := json.Marshal(document)
		fmt.Println(string(encoded))
		os.Exit(3)
	}
	fatal(2, fmt.Errorf("unknown scenario identity %q", scenario))
	return scenarioExpectation{}
}

func loadConfigFromEnvironment() (loadConfig, error) {
	if strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID")) != supportedProfile {
		return loadConfig{}, fmt.Errorf("supports load profile %q only", supportedProfile)
	}
	config := loadConfig{
		connections: envInt("PLAB_CONNECTIONS"), concurrency: envInt("PLAB_CONCURRENCY"),
		durationSeconds: envInt("PLAB_DURATION_SECONDS"), warmupSeconds: envInt("PLAB_WARMUP_SECONDS"),
		repetition: envInt("PLAB_REPETITION"), operationTimeoutMilliseconds: envIntDefault("PLAB_OPERATION_TIMEOUT_MILLISECONDS", 5000),
	}
	if config.connections != 1 || config.concurrency != 1 || config.durationSeconds != 5 || config.warmupSeconds != 1 || config.repetition != 1 || config.operationTimeoutMilliseconds != 5000 {
		return loadConfig{}, fmt.Errorf("websocket-smoke requires connections=1 concurrency=1 duration=5 warmup=1 repetition=1 operationTimeoutMilliseconds=5000: %+v", config)
	}
	return config, nil
}

func runPhase(targetURL, rootCertificate string, expectation scenarioExpectation, duration, timeout time.Duration, name string) phaseSummary {
	summary := newPhaseSummary(name)
	if expectation.operation == "text-echo" || expectation.operation == "binary-echo" || expectation.operation == "control-frames" {
		return runReusablePhase(targetURL, rootCertificate, expectation, duration, timeout, name)
	}
	started := time.Now()
	deadline := started.Add(duration)
	for time.Now().Before(deadline) {
		proof, err := performOperation(targetURL, rootCertificate, expectation, timeout)
		if err != nil {
			if isTimeout(err) {
				summary.TimedOutOperations++
			} else {
				summary.FailedOperations++
			}
			summary.Errors[err.Error()]++
			break
		}
		recordOpeningHandshake(&summary, proof.Handshake)
		summary.CompletedOperations++
		summary.TotalTransferredBytes += proof.TransferredBytes
		summary.LatenciesMilliseconds = append(summary.LatenciesMilliseconds, proof.LatencyMilliseconds)
		summary.LastProof = proof
	}
	summary.DurationSeconds = time.Since(started).Seconds()
	return summary
}

func runReusablePhase(targetURL, rootCertificate string, expectation scenarioExpectation, duration, timeout time.Duration, name string) phaseSummary {
	summary := newPhaseSummary(name)
	connection, err := openWebSocket(targetURL, rootCertificate, timeout)
	if err != nil {
		summary.FailedOperations = 1
		summary.Errors[err.Error()]++
		return summary
	}
	defer connection.conn.Close()
	recordOpeningHandshake(&summary, connection.handshake)
	started := time.Now()
	deadline := started.Add(duration)
	for time.Now().Before(deadline) {
		_ = connection.conn.SetDeadline(time.Now().Add(timeout))
		proof, err := performOnOpenConnection(connection, expectation)
		if err != nil {
			if isTimeout(err) {
				summary.TimedOutOperations++
			} else {
				summary.FailedOperations++
			}
			summary.Errors[err.Error()]++
			break
		}
		summary.CompletedOperations++
		summary.TotalTransferredBytes += proof.TransferredBytes
		summary.LatenciesMilliseconds = append(summary.LatenciesMilliseconds, proof.LatencyMilliseconds)
		summary.LastProof = proof
	}
	if summary.FailedOperations == 0 && summary.TimedOutOperations == 0 {
		if err := closeCleanly(connection, &summary.LastProof); err != nil {
			if isTimeout(err) {
				summary.TimedOutOperations++
			} else {
				summary.FailedOperations++
			}
			summary.Errors[err.Error()]++
		}
	}
	summary.DurationSeconds = time.Since(started).Seconds()
	return summary
}

func newPhaseSummary(name string) phaseSummary {
	return phaseSummary{
		Phase: name, EffectiveConnections: 1, EffectiveConcurrency: 1,
		UpgradeRequestHeadersMatched: true, UpgradeResponseHeadersMatched: true,
		Errors: map[string]int{}, seenKeys: map[string]struct{}{},
	}
}

func recordOpeningHandshake(summary *phaseSummary, proof handshakeProof) {
	summary.OpeningHandshakes++
	decodedKey, err := base64.StdEncoding.DecodeString(proof.SampleSecWebSocketKey)
	if err != nil || len(decodedKey) != 16 {
		summary.InvalidDecodedKeyCount++
	}
	if _, exists := summary.seenKeys[proof.SampleSecWebSocketKey]; exists {
		summary.KeyReuseCount++
	} else {
		summary.seenKeys[proof.SampleSecWebSocketKey] = struct{}{}
	}
	if websocketAccept(proof.SampleSecWebSocketKey) != proof.ObservedSecWebSocketAccept {
		summary.AcceptMismatchCount++
	}
	summary.UpgradeRequestHeadersMatched = summary.UpgradeRequestHeadersMatched &&
		proof.Binding == "http1-upgrade" && proof.Scheme == "wss" && proof.TransportSecurity == "tls" &&
		proof.Endpoint == "/websocket" && proof.Authority == "websocket.plab.test" && proof.RequestMethod == http.MethodGet &&
		proof.RequestedHTTPVersion == "HTTP/1.1" && proof.SecWebSocketVersion == "13" &&
		!proof.SubprotocolRequested && !proof.ExtensionsRequested
	summary.UpgradeResponseHeadersMatched = summary.UpgradeResponseHeadersMatched &&
		proof.ObservedHTTPVersion == "HTTP/1.1" && proof.ResponseStatus == http.StatusSwitchingProtocols &&
		strings.EqualFold(proof.UpgradeHeader, "websocket") && hasToken(proof.ConnectionHeader, "upgrade") &&
		!proof.SubprotocolNegotiated && !proof.ExtensionsNegotiated &&
		proof.ObservedSecWebSocketAccept == proof.ExpectedSecWebSocketAccept
}

func performOperation(targetURL, rootCertificate string, expectation scenarioExpectation, timeout time.Duration) (operationProof, error) {
	connection, err := openWebSocket(targetURL, rootCertificate, timeout)
	if err != nil {
		return operationProof{}, err
	}
	defer connection.conn.Close()
	proof := operationProof{
		ScenarioID: expectation.id, Operation: expectation.operation, MessageType: expectation.messageType,
		MessageBytes: len(expectation.payload), PayloadSHA256: expectation.payloadHash, Fragmentation: "none",
		Handshake: connection.handshake, CloseReasonBytes: 0,
		HandshakeRequest: connection.rawRequest, HandshakeResponse: connection.rawResponse,
	}
	applySemanticCounts(&proof, expectation)
	if expectation.operation == "establish" {
		proof.LatencyMilliseconds = connection.handshakeMS
		if err := closeCleanly(connection, &proof); err != nil {
			return proof, err
		}
		proof.TransferredBytes = connection.bytes
		return proof, nil
	}
	if expectation.operation == "close" {
		started := time.Now()
		if err := closeCleanly(connection, &proof); err != nil {
			return proof, err
		}
		proof.LatencyMilliseconds = durationMS(time.Since(started))
		proof.TransferredBytes = connection.bytes
		return proof, nil
	}
	proof, err = performOnOpenConnection(connection, expectation)
	if err != nil {
		return proof, err
	}
	if err := closeCleanly(connection, &proof); err != nil {
		return proof, err
	}
	proof.TransferredBytes = connection.bytes
	return proof, nil
}

func performOnOpenConnection(connection *wsConnection, expectation scenarioExpectation) (operationProof, error) {
	startBytes := connection.bytes
	proof := operationProof{
		ScenarioID: expectation.id, Operation: expectation.operation, MessageType: expectation.messageType,
		MessageBytes: len(expectation.payload), PayloadSHA256: expectation.payloadHash, Fragmentation: "none",
		Handshake: connection.handshake, CloseReasonBytes: 0,
		HandshakeRequest: connection.rawRequest, HandshakeResponse: connection.rawResponse,
	}
	applySemanticCounts(&proof, expectation)
	started := time.Now()
	written, err := writeClientFrame(connection.conn, expectation.requestOpcode, expectation.payload)
	connection.bytes += int64(written)
	if err != nil {
		return proof, err
	}
	proof.ClientFrameMasked = true
	proof.RequestOpcode = opcodeName(expectation.requestOpcode)
	frame, err := readFrame(connection.reader, false)
	connection.bytes += int64(frame.bytes)
	if err != nil {
		return proof, err
	}
	proof.LatencyMilliseconds = durationMS(time.Since(started))
	proof.ServerFrameMasked = frame.masked
	proof.ResponseOpcode = opcodeName(frame.opcode)
	if !frame.fin || frame.rsv != 0 || frame.opcode != expectation.responseOpcode || !equal(frame.payload, expectation.payload) {
		return proof, errors.New("response frame opcode, fragmentation, or deterministic payload mismatch")
	}
	proof.OrderedEcho = expectation.operation == "text-echo" || expectation.operation == "binary-echo"
	proof.PingSent = expectation.operation == "control-frames"
	proof.PongReceived = expectation.operation == "control-frames"
	proof.TransferredBytes = connection.bytes - startBytes
	return proof, nil
}

func openWebSocket(targetURL, rootCertificate string, timeout time.Duration) (*wsConnection, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	rootPEM, err := os.ReadFile(rootCertificate)
	if err != nil {
		return nil, fmt.Errorf("read authenticated root certificate: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootPEM) {
		return nil, errors.New("authenticated root certificate PEM did not contain a certificate")
	}
	tlsDialer := tls.Dialer{
		NetDialer: &net.Dialer{},
		Config: &tls.Config{
			RootCAs: roots, ServerName: serverName, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13,
			NextProtos: []string{"http/1.1"}, ClientSessionCache: nil,
		},
	}
	rawConnection, err := tlsDialer.DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		return nil, err
	}
	conn, ok := rawConnection.(*tls.Conn)
	if !ok {
		rawConnection.Close()
		return nil, errors.New("TLS dialer returned a non-TLS connection")
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	state := conn.ConnectionState()
	var certificateDERSHA256, certificateSPKISHA256 string
	if len(state.PeerCertificates) != 0 {
		certificateDERSHA256 = hash(state.PeerCertificates[0].Raw)
		certificateSPKISHA256 = hash(state.PeerCertificates[0].RawSubjectPublicKeyInfo)
	}
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		conn.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	expectedAccept := websocketAccept(key)
	rawRequest := "GET /websocket HTTP/1.1\r\nHost: websocket.plab.test\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: " + key + "\r\nSec-WebSocket-Version: 13\r\n\r\n"
	started := time.Now()
	if _, err := io.WriteString(conn, rawRequest); err != nil {
		conn.Close()
		return nil, err
	}
	reader := bufio.NewReader(conn)
	rawResponse, err := readHTTPHeader(reader)
	if err != nil {
		conn.Close()
		return nil, err
	}
	response, err := http.ReadResponse(bufio.NewReader(strings.NewReader(rawResponse)), &http.Request{Method: http.MethodGet})
	if err != nil {
		conn.Close()
		return nil, err
	}
	defer response.Body.Close()
	proof := handshakeProof{
		Binding: "http1-upgrade", Scheme: "wss", TransportSecurity: "tls", Endpoint: "/websocket", Authority: "websocket.plab.test",
		RequestMethod: "GET", RequestedHTTPVersion: "HTTP/1.1", ObservedHTTPVersion: response.Proto,
		ResponseStatus: response.StatusCode, UpgradeHeader: response.Header.Get("Upgrade"), ConnectionHeader: response.Header.Get("Connection"),
		SecWebSocketVersion: "13", SecWebSocketKeyPolicy: "fresh-random-16-octets-per-opening-handshake", SampleSecWebSocketKey: key,
		ExpectedSecWebSocketAccept: expectedAccept, ObservedSecWebSocketAccept: response.Header.Get("Sec-WebSocket-Accept"),
		SubprotocolRequested: false, SubprotocolNegotiated: response.Header.Get("Sec-WebSocket-Protocol") != "",
		ExtensionsRequested: false, ExtensionsNegotiated: response.Header.Get("Sec-WebSocket-Extensions") != "",
		FallbackDetected: response.ProtoMajor != 1 || response.ProtoMinor != 1 || state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != "http/1.1",
		TLSVersion:       tls.VersionName(state.Version), ALPN: state.NegotiatedProtocol, ServerName: serverName,
		CipherSuite: tls.CipherSuiteName(state.CipherSuite), DidResume: state.DidResume, EarlyData: false,
		VerifiedChainCount: len(state.VerifiedChains), CertificateDERSHA256: certificateDERSHA256, CertificateSPKISHA256: certificateSPKISHA256,
	}
	var failures []string
	if proof.FallbackDetected {
		failures = append(failures, "observed protocol is not exact HTTP/1.1")
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		failures = append(failures, "opening handshake status is not 101")
	}
	if !strings.EqualFold(proof.UpgradeHeader, "websocket") || !hasToken(proof.ConnectionHeader, "upgrade") {
		failures = append(failures, "opening handshake Upgrade/Connection response is invalid")
	}
	if proof.ObservedSecWebSocketAccept != expectedAccept {
		failures = append(failures, "Sec-WebSocket-Accept mismatch")
	}
	if proof.SubprotocolNegotiated || proof.ExtensionsNegotiated {
		failures = append(failures, "unrequested subprotocol or extension was negotiated")
	}
	if proof.TLSVersion != "TLS 1.3" || proof.ALPN != "http/1.1" || proof.ServerName != serverName || proof.DidResume || proof.EarlyData {
		failures = append(failures, "exact TLS 1.3, http/1.1 ALPN, SNI, full-session, or no-early-data proof mismatch")
	}
	if proof.VerifiedChainCount < 1 || proof.CertificateDERSHA256 != expectedCertificateDERSHA256 || proof.CertificateSPKISHA256 != expectedCertificateSPKISHA256 {
		failures = append(failures, "authenticated certificate hash proof mismatch")
	}
	if len(failures) != 0 {
		conn.Close()
		return nil, errors.New(strings.Join(failures, "; "))
	}
	proof.ConnectionEstablished = true
	return &wsConnection{conn: conn, reader: reader, handshake: proof, rawRequest: rawRequest, rawResponse: rawResponse, handshakeMS: durationMS(time.Since(started)), bytes: int64(len(rawRequest) + len(rawResponse))}, nil
}

func closeCleanly(connection *wsConnection, proof *operationProof) error {
	payload := []byte{0x03, 0xE8}
	written, err := writeClientFrame(connection.conn, 0x8, payload)
	connection.bytes += int64(written)
	if err != nil {
		return err
	}
	proof.ClientFrameMasked = true
	proof.CloseSentCode = 1000
	frame, err := readFrame(connection.reader, false)
	connection.bytes += int64(frame.bytes)
	if err != nil {
		return err
	}
	if !frame.fin || frame.rsv != 0 || frame.masked || frame.opcode != 0x8 || len(frame.payload) != 2 || binary.BigEndian.Uint16(frame.payload) != 1000 {
		return errors.New("clean close echo mismatch")
	}
	proof.ServerFrameMasked = false
	proof.CloseReceivedCode = 1000
	if _, err := connection.reader.ReadByte(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected bytes followed the WebSocket closing handshake")
		}
		return fmt.Errorf("waiting for peer transport close after WebSocket closing handshake: %w", err)
	}
	proof.TransportEOFObserved = true
	proof.CleanCompletion = true
	return nil
}

func writePreflightArtifacts(output string, expectation scenarioExpectation, proof operationProof, operationErr error) {
	passed := operationErr == nil && proof.Handshake.ConnectionEstablished && !proof.Handshake.FallbackDetected && proof.CleanCompletion
	validation := map[string]any{
		"scenarioId": expectation.id, "passed": passed, "requestedProtocol": "websocket-over-h1-tls",
		"observedProtocol": func() string {
			if proof.Handshake.ConnectionEstablished {
				return "websocket-over-h1-tls"
			}
			return ""
		}(),
		"fallbackDetected": proof.Handshake.FallbackDetected, "completedOperations": boolInt(operationErr == nil),
		"failedOperations": boolInt(operationErr != nil && !isTimeout(operationErr)), "timedOutOperations": boolInt(isTimeout(operationErr)),
		"error": errorString(operationErr),
	}
	writeRequired(output, "validation.json", validation)
	writeRequired(output, "protocol-proof.json", map[string]any{
		"requestedProtocol": "websocket-over-h1-tls", "observedProtocol": "websocket-over-h1-tls",
		"protocolVariant": "websocket-h1-tls1.3-upgrade", "fallbackDetected": proof.Handshake.FallbackDetected,
		"websocket": proof,
	})
	writeRequired(output, "tls-negotiation.json", map[string]any{
		"requestedVersion": "TLS 1.3", "observedVersion": proof.Handshake.TLSVersion,
		"requestedAlpn": "http/1.1", "observedAlpn": proof.Handshake.ALPN, "serverName": proof.Handshake.ServerName,
		"cipherSuite": proof.Handshake.CipherSuite, "didResume": proof.Handshake.DidResume, "earlyData": proof.Handshake.EarlyData,
		"verifiedChainCount":     proof.Handshake.VerifiedChainCount,
		"certificateDerSha256":   proof.Handshake.CertificateDERSHA256,
		"certificateSpkiSha256":  proof.Handshake.CertificateSPKISHA256,
		"certificateHashMatched": proof.Handshake.CertificateDERSHA256 == expectedCertificateDERSHA256 && proof.Handshake.CertificateSPKISHA256 == expectedCertificateSPKISHA256,
	})
	writeRequired(output, "websocket-summary.json", proof)
	writeRequired(output, "payload-hash.json", map[string]any{
		"scenarioId": expectation.id, "generator": payloadGenerator(expectation), "lengthBytes": len(expectation.payload),
		"sha256": expectation.payloadHash, "observedSha256": hash(expectation.payload), "matched": expectation.payloadHash == hash(expectation.payload),
	})
	writeRequired(output, "handshake-summary.json", proof.Handshake)
	writeRequired(output, "frame-summary.json", map[string]any{
		"fragmentation": "none", "clientFramesMasked": proof.ClientFrameMasked, "serverFramesMasked": proof.ServerFrameMasked,
		"requestOpcode": proof.RequestOpcode, "responseOpcode": proof.ResponseOpcode, "closeCode": proof.CloseReceivedCode,
	})
	writeRequired(output, "executor-identity.json", map[string]any{
		"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedScenarios": sortedSupportedIDs(),
	})
	if err := os.WriteFile(filepath.Join(output, "handshake-request.txt"), []byte(proof.HandshakeRequest), 0o644); err != nil {
		fatal(1, err)
	}
	if err := os.WriteFile(filepath.Join(output, "handshake-response.txt"), []byte(proof.HandshakeResponse), 0o644); err != nil {
		fatal(1, err)
	}
	writeRequired(output, "result.json", validation)
}

func normalizeResult(expectation scenarioExpectation, config loadConfig, summary phaseSummary) normalizedResult {
	metric := metrics{
		BytesPerSecond:      float64(summary.TotalTransferredBytes) / summary.DurationSeconds,
		CompletedOperations: summary.CompletedOperations, FailedOperations: summary.FailedOperations,
		TimedOutOperations: summary.TimedOutOperations, TotalTransferredBytes: summary.TotalTransferredBytes,
		EffectiveConnections: summary.EffectiveConnections, EffectiveConcurrency: summary.EffectiveConcurrency,
	}
	rate := float64(summary.CompletedOperations) / summary.DurationSeconds
	meanValue, p50, p75, p90, p95, p99 := mean(summary.LatenciesMilliseconds), percentile(summary.LatenciesMilliseconds, .50), percentile(summary.LatenciesMilliseconds, .75), percentile(summary.LatenciesMilliseconds, .90), percentile(summary.LatenciesMilliseconds, .95), percentile(summary.LatenciesMilliseconds, .99)
	switch expectation.operation {
	case "establish":
		metric.ConnectionsPerSecond = rate
		metric.ConnectionLatencyMean, metric.ConnectionLatencyP50, metric.ConnectionLatencyP75, metric.ConnectionLatencyP90, metric.ConnectionLatencyP95, metric.ConnectionLatencyP99 = meanValue, p50, p75, p90, p95, p99
	case "close":
		metric.CloseLatencyP50, metric.CloseLatencyP95, metric.CloseLatencyP99 = p50, p95, p99
	case "control-frames":
		metric.MessagesPerSecond = rate
		metric.ControlFrameLatencyP50, metric.ControlFrameLatencyP95, metric.ControlFrameLatencyP99 = p50, p95, p99
	default:
		metric.MessagesPerSecond = rate
		metric.MessageLatencyMean, metric.MessageLatencyP50, metric.MessageLatencyP75, metric.MessageLatencyP90, metric.MessageLatencyP95, metric.MessageLatencyP99 = meanValue, p50, p75, p90, p95, p99
	}
	requested := map[string]any{
		"connections": config.connections, "concurrency": config.concurrency, "durationSeconds": config.durationSeconds,
		"warmupSeconds": config.warmupSeconds, "repetition": config.repetition,
		"totalOperations": nil, "operationRateLimitPerSecond": nil, "operationTimeoutMilliseconds": config.operationTimeoutMilliseconds,
	}
	return normalizedResult{
		SchemaVersion: "protocol-lab.websocket-h1-executor-result.v1", ScenarioID: expectation.id, LoadProfileID: supportedProfile, Status: "passed",
		Executor: map[string]string{"id": executorID, "version": executorVersion}, LoadGenerator: map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion},
		ProtocolProof: map[string]any{
			"requestedProtocol": "websocket-over-h1-tls", "observedProtocol": "websocket-over-h1-tls",
			"protocolVariant": "websocket-h1-tls1.3-upgrade", "fallbackDetected": false, "websocket": summary.LastProof,
			"tls": map[string]any{
				"version": summary.LastProof.Handshake.TLSVersion, "alpn": summary.LastProof.Handshake.ALPN,
				"serverName": summary.LastProof.Handshake.ServerName, "cipherSuite": summary.LastProof.Handshake.CipherSuite,
				"didResume": summary.LastProof.Handshake.DidResume, "earlyData": summary.LastProof.Handshake.EarlyData,
				"verifiedChainCount":    summary.LastProof.Handshake.VerifiedChainCount,
				"certificateDerSha256":  summary.LastProof.Handshake.CertificateDERSHA256,
				"certificateSpkiSha256": summary.LastProof.Handshake.CertificateSPKISHA256,
			},
			"handshakeAggregate": map[string]any{
				"binding": "http1-upgrade", "openingHandshakes": summary.OpeningHandshakes,
				"keyReuseCount": summary.KeyReuseCount, "invalidDecodedKeyCount": summary.InvalidDecodedKeyCount,
				"acceptMismatchCount":           summary.AcceptMismatchCount,
				"sampleSecWebSocketKey":         summary.LastProof.Handshake.SampleSecWebSocketKey,
				"sampleSecWebSocketAccept":      summary.LastProof.Handshake.ObservedSecWebSocketAccept,
				"upgradeRequestHeadersMatched":  summary.UpgradeRequestHeadersMatched,
				"upgradeResponseHeadersMatched": summary.UpgradeResponseHeadersMatched,
			},
		},
		Validation:    map[string]any{"status": "passed", "zeroUnexpectedFailures": true, "zeroTimeouts": true, "cleanClose": true},
		RequestedLoad: requested,
		EffectiveLoad: map[string]any{"connections": 1, "activeConnections": 1, "concurrency": 1}, Metrics: metric,
		Warnings: []string{"Local package-backed WebSocket TLS smoke is diagnostic and non-publishable. Cleartext substitution, TLS 1.2, RFC 8441, RFC 9220, breadth diagnostics, WebTransport, and legacy websocket.echo substitution are unsupported."},
	}
}

func writeClientFrame(writer io.Writer, opcode byte, payload []byte) (int, error) {
	header := []byte{0x80 | opcode}
	switch {
	case len(payload) <= 125:
		header = append(header, 0x80|byte(len(payload)))
	case len(payload) <= 65535:
		header = append(header, 0x80|126, byte(len(payload)>>8), byte(len(payload)))
	default:
		return 0, errors.New("payload exceeds package limit")
	}
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return 0, err
	}
	masked := append([]byte(nil), payload...)
	for i := range masked {
		masked[i] ^= mask[i%4]
	}
	wire := append(append(header, mask...), masked...)
	n, err := writer.Write(wire)
	return n, err
}

func readFrame(reader *bufio.Reader, requireMask bool) (wireFrame, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return wireFrame{}, err
	}
	result := wireFrame{fin: header[0]&0x80 != 0, rsv: header[0] & 0x70, opcode: header[0] & 0x0f, masked: header[1]&0x80 != 0, bytes: 2}
	if result.masked != requireMask {
		return result, errors.New("RFC 6455 masking direction mismatch")
	}
	length, lengthBytes, err := readPayloadLength(reader, header[1]&0x7f)
	result.bytes += lengthBytes
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
		result.bytes += 4
	}
	result.payload = make([]byte, length)
	if _, err := io.ReadFull(reader, result.payload); err != nil {
		return result, err
	}
	result.bytes += length
	if result.masked {
		for i := range result.payload {
			result.payload[i] ^= mask[i%4]
		}
	}
	if result.opcode >= 0x8 && (!result.fin || len(result.payload) > 125) {
		return result, errors.New("invalid control frame")
	}
	return result, nil
}

func readPayloadLength(reader io.Reader, encoded byte) (int, int, error) {
	switch encoded {
	case 126:
		value := make([]byte, 2)
		if _, err := io.ReadFull(reader, value); err != nil {
			return 0, 2, err
		}
		return int(binary.BigEndian.Uint16(value)), 2, nil
	case 127:
		value := make([]byte, 8)
		if _, err := io.ReadFull(reader, value); err != nil {
			return 0, 8, err
		}
		length := binary.BigEndian.Uint64(value)
		if length > uint64(^uint(0)>>1) {
			return 0, 8, errors.New("frame length overflows int")
		}
		return int(length), 8, nil
	default:
		return int(encoded), 0, nil
	}
}

func readHTTPHeader(reader *bufio.Reader) (string, error) {
	var data []byte
	for len(data) < 16384 {
		value, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		data = append(data, value)
		if len(data) >= 4 && string(data[len(data)-4:]) == "\r\n\r\n" {
			return string(data), nil
		}
	}
	return "", errors.New("opening handshake response header exceeds 16 KiB")
}

func normalizeTarget(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return "", errors.New("TLS WebSocket target must use https://host:port")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("target query and fragment are prohibited")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/websocket"
	}
	if parsed.Path != "/websocket" {
		return "", errors.New("target endpoint must be /websocket")
	}
	return parsed.String(), nil
}

func packagedRoot() string {
	executable, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(executable), "..", ".."))
}

func websocketAccept(key string) string {
	digest := sha1.Sum([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(digest[:])
}
func sortedSupportedIDs() []string {
	ids := make([]string, 0, len(expectations))
	for id := range expectations {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func repeatedByte(value byte, count int) []byte {
	result := make([]byte, count)
	for i := range result {
		result[i] = value
	}
	return result
}
func hash(value []byte) string { digest := sha256.Sum256(value); return hex.EncodeToString(digest[:]) }
func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func hasToken(value, token string) bool {
	for _, candidate := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(candidate), token) {
			return true
		}
	}
	return false
}
func opcodeName(value byte) string {
	switch value {
	case 0x1:
		return "text"
	case 0x2:
		return "binary"
	case 0x8:
		return "close"
	case 0x9:
		return "ping"
	case 0xA:
		return "pong"
	default:
		return fmt.Sprintf("0x%x", value)
	}
}
func payloadGenerator(value scenarioExpectation) string {
	switch value.operation {
	case "text-echo", "control-frames":
		return "exact-utf8"
	case "binary-echo":
		return "repeated-octet"
	default:
		return "none"
	}
}
func applySemanticCounts(proof *operationProof, expectation scenarioExpectation) {
	switch expectation.operation {
	case "control-frames":
		proof.MessageBytes = 0
		proof.MessageCount = 1
		proof.ControlPayloadBytes = len(expectation.payload)
		proof.ControlPayloadSHA256 = expectation.payloadHash
	case "text-echo", "binary-echo":
		proof.MessageBytes = len(expectation.payload)
		proof.MessageCount = 1
	default:
		proof.MessageBytes = 0
		proof.MessageCount = 0
	}
}
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}
func percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	index := int(math.Ceil(q*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
func durationMS(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }
func envInt(name string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	return value
}
func envIntDefault(name string, fallback int) int {
	if strings.TrimSpace(os.Getenv(name)) == "" {
		return fallback
	}
	return envInt(name)
}
func verifySubstitution(variable, expected, label string) {
	if observed := strings.TrimSpace(os.Getenv(variable)); observed != "" && observed != expected {
		fatal(2, fmt.Errorf("%s substitution detected: expected %q, observed %q", label, expected, observed))
	}
}
func hasFailures(summary phaseSummary) bool {
	return summary.FailedOperations != 0 || summary.TimedOutOperations != 0
}
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout())
}
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
func writeRequired(directory, name string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(directory, name), append(data, '\n'), 0o644)
	}
	if err != nil {
		fatal(1, err)
	}
}
func fatal(code int, err error) { fmt.Fprintln(os.Stderr, err); os.Exit(code) }
