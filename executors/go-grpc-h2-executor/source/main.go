package main

import (
	"bytes"
	"compress/gzip"
	"context"
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
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"
)

const (
	executorID                  = "go-grpc-h2-executor"
	executorVersion             = "0.4.0"
	loadGeneratorID             = "go-x-net-http2-grpc-load"
	loadGeneratorVersion        = "0.4.0"
	loadProfileSmoke            = "grpc-h2-smoke"
	loadProfileDiagnostic       = "grpc-h2-diagnostic"
	loadProfileChannelChurn     = "grpc-h2-channel-churn"
	protocolVariant             = "grpc-over-h2-tls-alpn"
	protocolVariantNewChannel   = "grpc-over-h2-tls-new-channel"
	serviceContractDigest       = "b7b987814f8af5cd4f15c03989b9c309c1c0ec643972ae32668304d71502120f"
	expectedLeafCertificateHash = "4627eed7781247db01e5641e4799e1c71e45543c10e304c99ed74b6e0bc0e254"
	expectedLeafSPKIHash        = "83737e10fac4d3acac8245615dc510b8c12deb8f4756af3eaddf21118f18e4ea"
	expectedRootPEMHash         = "dda81fe80d268bbd91993339ceef728a7e9e181d1faee88af35d73be2a7b039b"
)

type scenarioSpec struct {
	id, method, path, rpcType, compression, metadataProfile  string
	payload                                                  []byte
	payloadHash, protobufHash, frameHash                     string
	protobufBytes, frameBytes                                int
	requestCount, responseCount                              int
	deadline                                                 time.Duration
	terminalMode, expectedStatus, expectedMessage, lifecycle string
}

var supportedScenarioIDs = []string{"grpc.h2.unary.echo", "grpc.h2.unary.empty", "grpc.h2.unary.fixed-metadata", "grpc.h2.unary.gzip", "grpc.h2.unary.large", "grpc.h2.server-streaming.echo", "grpc.h2.client-streaming.echo", "grpc.h2.bidi-streaming.echo", "grpc.h2.trailers-only-status", "grpc.h2.deadline-exceeded", "grpc.h2.client-cancellation", "grpc.h2.unary.echo-new-channel"}
var knownUnsupportedScenarioIDs = []string{}

func specFor(id string) (scenarioSpec, bool) {
	switch id {
	case "grpc.h2.unary.echo":
		return makeIdentitySpec(id, "UnaryEcho", bytes.Repeat([]byte{'G'}, 128), 5*time.Second), true
	case "grpc.h2.unary.empty":
		return makeIdentitySpec(id, "UnaryEcho", nil, 5*time.Second), true
	case "grpc.h2.unary.fixed-metadata":
		s := makeIdentitySpec(id, "UnaryFixedMetadata", bytes.Repeat([]byte{'G'}, 128), 5*time.Second)
		s.metadataProfile = "fixed-ascii-and-binary-metadata-v1"
		return s, true
	case "grpc.h2.unary.gzip":
		payload := bytes.Repeat([]byte{'B'}, 1024)
		protobuf := encodeProtobuf(payload)
		frame, _ := encodeFrame(protobuf, "gzip")
		return scenarioSpec{id: id, method: "UnaryGzip", path: "/protocollab.performance.v1.EchoService/UnaryGzip", rpcType: "unary", compression: "gzip", metadataProfile: "fixed-empty-user-metadata", payload: payload, payloadHash: sha256Hex(payload), protobufHash: sha256Hex(protobuf), frameHash: sha256Hex(frame), protobufBytes: len(protobuf), frameBytes: len(frame), requestCount: 1, responseCount: 1, deadline: 5 * time.Second}, true
	case "grpc.h2.unary.large":
		return makeIdentitySpec(id, "UnaryEcho", bytes.Repeat([]byte{'L'}, 1<<20), 10*time.Second), true
	case "grpc.h2.server-streaming.echo":
		return makeStreamingSpec(id, "ServerStreamingEcho", "server-streaming", 1, 100), true
	case "grpc.h2.client-streaming.echo":
		return makeStreamingSpec(id, "ClientStreamingEcho", "client-streaming", 100, 1), true
	case "grpc.h2.bidi-streaming.echo":
		return makeStreamingSpec(id, "BidirectionalStreamingEcho", "bidirectional-streaming", 100, 100), true
	case "grpc.h2.trailers-only-status":
		s := makeIdentitySpec(id, "TrailersOnlyStatus", bytes.Repeat([]byte{'G'}, 128), 5*time.Second)
		s.responseCount, s.terminalMode, s.expectedStatus, s.expectedMessage = 0, "trailers-only", "3", "plab invalid fixture"
		return s, true
	case "grpc.h2.deadline-exceeded":
		s := makeIdentitySpec(id, "DeadlineExceeded", bytes.Repeat([]byte{'G'}, 128), 50*time.Millisecond)
		s.responseCount, s.terminalMode, s.expectedStatus = 0, "deadline-exceeded", "4"
		return s, true
	case "grpc.h2.client-cancellation":
		s := makeIdentitySpec(id, "ClientCancellation", bytes.Repeat([]byte{'G'}, 128), 5*time.Second)
		s.rpcType, s.responseCount, s.terminalMode, s.expectedStatus = "server-streaming", 0, "client-cancellation", "1"
		return s, true
	case "grpc.h2.unary.echo-new-channel":
		s := makeIdentitySpec(id, "UnaryEcho", bytes.Repeat([]byte{'G'}, 128), 5*time.Second)
		s.lifecycle = "new-channel-per-operation"
		return s, true
	default:
		return scenarioSpec{}, false
	}
}

func makeIdentitySpec(id, method string, payload []byte, deadline time.Duration) scenarioSpec {
	protobuf := encodeProtobuf(payload)
	frame, _ := encodeFrame(protobuf, "identity")
	return scenarioSpec{id: id, method: method, path: "/protocollab.performance.v1.EchoService/" + method, rpcType: "unary", compression: "identity", metadataProfile: "fixed-empty-user-metadata", payload: payload, payloadHash: sha256Hex(payload), protobufHash: sha256Hex(protobuf), frameHash: sha256Hex(frame), protobufBytes: len(protobuf), frameBytes: len(frame), requestCount: 1, responseCount: 1, deadline: deadline, expectedStatus: "0", lifecycle: "pre-established-channel"}
}

func makeStreamingSpec(id, method, rpcType string, requestCount, responseCount int) scenarioSpec {
	payload := bytes.Repeat([]byte{'B'}, 1024)
	protobuf := encodeProtobuf(payload)
	frame, _ := encodeFrame(protobuf, "identity")
	return scenarioSpec{id: id, method: method, path: "/protocollab.performance.v1.EchoService/" + method, rpcType: rpcType, compression: "identity", metadataProfile: "fixed-empty-user-metadata", payload: payload, payloadHash: sha256Hex(payload), protobufHash: sha256Hex(protobuf), frameHash: sha256Hex(frame), protobufBytes: len(protobuf), frameBytes: len(frame), requestCount: requestCount, responseCount: responseCount, deadline: 15 * time.Second, expectedStatus: "0", lifecycle: "pre-established-channel"}
}

type rpcObservation struct {
	HTTPStatus              int     `json:"httpStatus"`
	HTTPVersion             string  `json:"httpVersion"`
	ContentType             string  `json:"contentType"`
	TrailersPresent         bool    `json:"trailersPresent"`
	GRPCStatus              string  `json:"grpcStatus"`
	GRPCEncoding            string  `json:"grpcEncoding"`
	ResponseFrameBytes      int     `json:"responseFrameBytes"`
	ResponseFrameHash       string  `json:"responseFrameSha256"`
	RequestMessageCount     int     `json:"requestMessageCount"`
	ResponseMessageCount    int     `json:"responseMessageCount"`
	ClientHalfClosed        bool    `json:"clientHalfClosed"`
	OrderedEcho             bool    `json:"orderedEcho"`
	StreamComplete          bool    `json:"streamComplete"`
	TimeToFirstMessageNanos int64   `json:"timeToFirstMessageNanos,omitempty"`
	MessageArrivalNanos     []int64 `json:"-"`
	ResponseInitialText     string  `json:"responseInitialTextMetadata,omitempty"`
	ResponseTrailingBinary  string  `json:"responseTrailingBinaryMetadata,omitempty"`
	GRPCMessage             string  `json:"grpcMessage,omitempty"`
	ReadyInitialMetadata    bool    `json:"readyInitialMetadata"`
	ClientCancelTriggered   bool    `json:"clientCancelTriggered"`
	DeadlineFired           bool    `json:"deadlineFired"`
	TrailersOnly            bool    `json:"trailersOnly"`
	NoResponseData          bool    `json:"noResponseData"`
	ExpectedTerminalOutcome bool    `json:"expectedTerminalOutcome"`
	LatencyNanos            int64   `json:"latencyNanos"`
	Passed                  bool    `json:"passed"`
	TimedOut                bool    `json:"timedOut"`
	Error                   string  `json:"error,omitempty"`
	ResponseFrame           []byte  `json:"-"`
}

type executionResult struct {
	SchemaVersion         string         `json:"schemaVersion"`
	ExecutorID            string         `json:"executorId"`
	ExecutorVersion       string         `json:"executorVersion"`
	LoadGeneratorID       string         `json:"loadGeneratorId"`
	LoadGeneratorVersion  string         `json:"loadGeneratorVersion"`
	ScenarioID            string         `json:"scenarioId"`
	LoadProfileID         string         `json:"loadProfileId"`
	ServiceContractSha256 string         `json:"serviceContractSha256"`
	Passed                bool           `json:"passed"`
	Protocol              map[string]any `json:"protocol"`
	Channel               map[string]any `json:"channel"`
	Request               map[string]any `json:"request"`
	Response              map[string]any `json:"response"`
	RequestedLoad         map[string]any `json:"requestedLoad"`
	EffectiveLoad         map[string]any `json:"effectiveLoad"`
	Metrics               map[string]any `json:"metrics"`
	Validation            map[string]any `json:"validation"`
	Observation           rpcObservation `json:"observation"`
	Artifacts             []string       `json:"artifacts"`
}

type countingConn struct {
	net.Conn
	readBytes    atomic.Int64
	writtenBytes atomic.Int64
}

func (c *countingConn) Read(p []byte) (int, error) {
	n, e := c.Conn.Read(p)
	c.readBytes.Add(int64(n))
	return n, e
}
func (c *countingConn) Write(p []byte) (int, error) {
	n, e := c.Conn.Write(p)
	c.writtenBytes.Add(int64(n))
	return n, e
}
func (c *countingConn) totals() (int64, int64) { return c.readBytes.Load(), c.writtenBytes.Load() }

type channelSession struct {
	counted   *countingConn
	tlsConn   *tls.Conn
	transport *http2.Transport
	channel   *http2.ClientConn
	leaf      *x509.Certificate
}

func (s *channelSession) close() {
	if s.channel != nil {
		s.channel.Close()
	}
	if s.tlsConn != nil {
		s.tlsConn.Close()
	}
}

func loadTargetAndRoots(targetURL, rootPath string) (*url.URL, *x509.CertPool, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, nil, errors.New("target URL must be an absolute https URL")
	}
	rootPEM, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, nil, err
	}
	if sha256Hex(rootPEM) != expectedRootPEMHash {
		return nil, nil, errors.New("trusted root certificate hash mismatch")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootPEM) {
		return nil, nil, errors.New("trusted root certificate could not be parsed")
	}
	return parsed, roots, nil
}

func openChannel(ctx context.Context, parsed *url.URL, roots *x509.CertPool, timeout time.Duration) (*channelSession, error) {
	raw, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		return nil, err
	}
	counted := &countingConn{Conn: raw}
	tlsConn := tls.Client(counted, &tls.Config{RootCAs: roots, ServerName: "grpc.plab.test", MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, NextProtos: []string{"h2"}, ClientSessionCache: nil})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		raw.Close()
		return nil, err
	}
	state := tlsConn.ConnectionState()
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != "h2" || !state.NegotiatedProtocolIsMutual || state.DidResume || len(state.PeerCertificates) != 1 {
		tlsConn.Close()
		return nil, errors.New("exact full TLS 1.3, mutual ALPN h2, and one authenticated leaf certificate are required")
	}
	leaf := state.PeerCertificates[0]
	if sha256Hex(leaf.Raw) != expectedLeafCertificateHash || sha256Hex(leaf.RawSubjectPublicKeyInfo) != expectedLeafSPKIHash {
		tlsConn.Close()
		return nil, errors.New("authenticated leaf certificate identity mismatch")
	}
	transport := &http2.Transport{}
	channel, err := transport.NewClientConn(tlsConn)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}
	return &channelSession{counted: counted, tlsConn: tlsConn, transport: transport, channel: channel, leaf: leaf}, nil
}

func main() {
	targetURL := flag.String("target-url", os.Getenv("PLAB_TARGET_BASE_URL"), "target HTTPS URL")
	outputDir := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "artifact directory")
	rootPath := flag.String("root-cert", envOrDefault("PLAB_TLS_ROOT_CERT_FILE", defaultMaterialPath("certs/root.pem")), "trusted root certificate")
	flag.Parse()
	if *outputDir == "" {
		*outputDir = "artifacts"
	}
	_ = os.MkdirAll(*outputDir, 0o755)
	scenarioID := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	spec, supported := specFor(scenarioID)
	if !supported {
		if contains(knownUnsupportedScenarioIDs, scenarioID) {
			writeUnsupported(*outputDir, scenarioID)
			fatal(3, "scenario is explicitly unsupported: "+scenarioID)
		}
		fatal(2, "unknown gRPC/H2 scenario: "+scenarioID)
	}
	if strings.TrimSpace(*targetURL) == "" {
		fatal(2, "target-url or PLAB_TARGET_BASE_URL is required")
	}
	loadProfileID, err := validateSelection(spec)
	if err != nil {
		fatal(2, err.Error())
	}
	timeoutFallback := spec.deadline
	if spec.terminalMode != "" || spec.lifecycle == "new-channel-per-operation" {
		timeoutFallback = 20 * time.Second
	}
	timeout := durationFromEnvironment("PLAB_REQUEST_TIMEOUT_SECONDS", timeoutFallback)
	var result executionResult
	var requestFrame, responseFrame, peer []byte
	if spec.lifecycle == "new-channel-per-operation" {
		result, requestFrame, responseFrame, peer, err = executeNewChannel(*targetURL, *rootPath, timeout, spec, loadProfileID)
	} else if spec.terminalMode != "" {
		result, requestFrame, responseFrame, peer, err = executeTerminal(*targetURL, *rootPath, timeout, spec, loadProfileID)
	} else {
		result, requestFrame, responseFrame, peer, err = execute(*targetURL, *rootPath, timeout, spec, loadProfileID)
	}
	if err != nil {
		result.Passed = false
		result.Validation["error"] = err.Error()
	}
	if e := writeArtifacts(*outputDir, result, requestFrame, responseFrame, peer); e != nil {
		fatal(1, e.Error())
	}
	encoded, _ := json.Marshal(result)
	fmt.Fprintln(os.Stdout, string(encoded))
	if err != nil || !result.Passed {
		fatal(1, "gRPC/H2 validity gate failed")
	}
}

func execute(targetURL, rootPath string, timeout time.Duration, spec scenarioSpec, loadProfileID string) (executionResult, []byte, []byte, []byte, error) {
	result := baseResult(spec, loadProfileID)
	frame, _ := encodeFrame(encodeProtobuf(spec.payload), spec.compression)
	requestFrames := bytes.Repeat(frame, spec.requestCount)
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return result, requestFrames, nil, nil, errors.New("target URL must be an absolute https URL")
	}
	rootPEM, err := os.ReadFile(rootPath)
	if err != nil {
		return result, requestFrames, nil, nil, err
	}
	if sha256Hex(rootPEM) != expectedRootPEMHash {
		return result, requestFrames, nil, nil, errors.New("trusted root certificate hash mismatch")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootPEM) {
		return result, requestFrames, nil, nil, errors.New("trusted root certificate could not be parsed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	raw, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		return result, requestFrames, nil, nil, err
	}
	counted := &countingConn{Conn: raw}
	defer counted.Close()
	tlsConn := tls.Client(counted, &tls.Config{RootCAs: roots, ServerName: "grpc.plab.test", MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, NextProtos: []string{"h2"}})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return result, requestFrames, nil, nil, err
	}
	state := tlsConn.ConnectionState()
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != "h2" || !state.NegotiatedProtocolIsMutual || len(state.PeerCertificates) != 1 {
		return result, requestFrames, nil, nil, errors.New("exact TLS 1.3, mutual ALPN h2, and one authenticated leaf certificate are required")
	}
	leaf := state.PeerCertificates[0]
	leafHash, spkiHash := sha256Hex(leaf.Raw), sha256Hex(leaf.RawSubjectPublicKeyInfo)
	if leafHash != expectedLeafCertificateHash || spkiHash != expectedLeafSPKIHash {
		return result, requestFrames, nil, leaf.Raw, errors.New("authenticated leaf certificate identity mismatch")
	}
	transport := &http2.Transport{}
	channel, err := transport.NewClientConn(tlsConn)
	if err != nil {
		return result, requestFrames, nil, leaf.Raw, err
	}
	defer channel.Close()
	warmupCompleted := false
	if loadProfileID == loadProfileSmoke {
		warmup := invoke(context.Background(), channel, parsed, requestFrames, timeout, spec)
		if !warmup.Passed {
			return result, requestFrames, nil, leaf.Raw, fmt.Errorf("pre-established channel warmup failed: %s", warmup.Error)
		}
		warmupCompleted = true
	}
	readBefore, writtenBefore := counted.totals()
	observation := invoke(context.Background(), channel, parsed, requestFrames, timeout, spec)
	readAfter, writtenAfter := counted.totals()
	networkBytes := (readAfter - readBefore) + (writtenAfter - writtenBefore)
	result.Observation = observation
	result.Protocol = map[string]any{"requested": "grpc-over-h2", "observed": "grpc-over-h2", "tlsVersion": "TLS1.3", "alpn": "h2", "httpVersion": observation.HTTPVersion, "fallbackDetected": observation.HTTPVersion != "HTTP/2.0", "leafCertificateSha256": leafHash, "leafSpkiSha256": spkiHash, "serverName": "grpc.plab.test"}
	result.Channel = map[string]any{"configuredChannels": 1, "configuredConnections": 1, "configuredStreamsPerConnection": 1, "observedActiveConnections": 1, "observedActiveStreams": 1, "preEstablished": true, "warmupCompleted": warmupCompleted, "reusedForMeasuredOperation": true}
	completed, failed, timedOut := 0, 1, 0
	if observation.Passed {
		completed, failed = 1, 0
	}
	if observation.TimedOut {
		timedOut = 1
	}
	latencyMS := float64(observation.LatencyNanos) / float64(time.Millisecond)
	rate := 0.0
	if observation.LatencyNanos > 0 && completed == 1 {
		rate = float64(time.Second) / float64(observation.LatencyNanos)
	}
	messageRate := 0.0
	if observation.LatencyNanos > 0 && completed == 1 {
		messageRate = float64(spec.requestCount+spec.responseCount) * float64(time.Second) / float64(observation.LatencyNanos)
	}
	messageLatencies := observation.MessageArrivalNanos
	if len(messageLatencies) == 0 {
		messageLatencies = []int64{observation.LatencyNanos}
	}
	result.Metrics = map[string]any{"rpcsPerSecond": rate, "messagesPerSecond": messageRate, "bytesPerSecond": func() float64 {
		if observation.LatencyNanos > 0 {
			return float64(networkBytes) * float64(time.Second) / float64(observation.LatencyNanos)
		}
		return 0
	}(), "completedOperations": completed, "failedOperations": failed, "deadlineExceededOperations": 0, "cancelledOperations": 0, "timedOutOperations": timedOut, "effectiveConcurrency": 1, "effectiveStreams": 1, "rpcLatencyMeanMilliseconds": latencyMS, "rpcLatencyP50Milliseconds": latencyMS, "rpcLatencyP75Milliseconds": latencyMS, "rpcLatencyP90Milliseconds": latencyMS, "rpcLatencyP95Milliseconds": latencyMS, "rpcLatencyP99Milliseconds": latencyMS, "messageLatencyMeanMilliseconds": meanMilliseconds(messageLatencies), "messageLatencyP50Milliseconds": percentileMilliseconds(messageLatencies, 0.50), "messageLatencyP75Milliseconds": percentileMilliseconds(messageLatencies, 0.75), "messageLatencyP90Milliseconds": percentileMilliseconds(messageLatencies, 0.90), "messageLatencyP95Milliseconds": percentileMilliseconds(messageLatencies, 0.95), "messageLatencyP99Milliseconds": percentileMilliseconds(messageLatencies, 0.99), "messageLatencyScope": "rpc-start-to-ordered-response-message-arrival", "timeToFirstMessageMilliseconds": float64(observation.TimeToFirstMessageNanos) / float64(time.Millisecond), "totalTransferredBytes": networkBytes, "networkBytesRead": readAfter - readBefore, "networkBytesWritten": writtenAfter - writtenBefore, "grpcMessageFrameBytesBidirectional": spec.requestCount*spec.frameBytes + observation.ResponseFrameBytes}
	result.Response = map[string]any{"count": observation.ResponseMessageCount, "payloadBytesPerMessage": len(spec.payload), "payloadSha256": spec.payloadHash, "serializedProtobufBytesPerMessage": spec.protobufBytes, "serializedProtobufSha256": spec.protobufHash, "grpcFrameBytesPerMessage": spec.frameBytes, "grpcFrameSha256": spec.frameHash, "aggregateGrpcFrameBytes": observation.ResponseFrameBytes, "aggregateGrpcFramesSha256": observation.ResponseFrameHash, "httpStatus": observation.HTTPStatus, "contentType": observation.ContentType, "trailersPresent": observation.TrailersPresent, "grpcStatus": observation.GRPCStatus, "grpcEncoding": observation.GRPCEncoding, "metadataProfile": spec.metadataProfile, "streamComplete": observation.StreamComplete, "orderedEcho": observation.OrderedEcho}
	if spec.responseCount == 1 {
		result.Response["payloadBytes"] = len(spec.payload)
		result.Response["serializedProtobufBytes"] = spec.protobufBytes
		result.Response["grpcFrameBytes"] = spec.frameBytes
	}
	if spec.metadataProfile == "fixed-ascii-and-binary-metadata-v1" {
		result.Response["responseInitialTextMetadata"] = observation.ResponseInitialText
		result.Response["responseTrailingBinaryMetadata"] = observation.ResponseTrailingBinary
		result.Response["responseTrailingBinaryMetadataDecodedSha256"] = sha256Hex([]byte{0, 1, 2, 3})
	}
	result.Passed = observation.Passed && completed == 1 && failed == 0 && timedOut == 0
	result.Validation = map[string]any{"passed": result.Passed, "checks": validationChecks(spec)}
	if !result.Passed {
		return result, requestFrames, observation.ResponseFrame, leaf.Raw, errors.New(observation.Error)
	}
	return result, requestFrames, observation.ResponseFrame, leaf.Raw, nil
}

func executeTerminal(targetURL, rootPath string, timeout time.Duration, spec scenarioSpec, loadProfileID string) (executionResult, []byte, []byte, []byte, error) {
	result := baseResult(spec, loadProfileID)
	frame, _ := encodeFrame(encodeProtobuf(spec.payload), spec.compression)
	parsed, roots, err := loadTargetAndRoots(targetURL, rootPath)
	if err != nil {
		return result, frame, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	session, err := openChannel(ctx, parsed, roots, timeout)
	if err != nil {
		return result, frame, nil, nil, err
	}
	defer session.close()
	readBefore, writtenBefore := session.counted.totals()
	observation := invokeTerminal(ctx, session.channel, parsed, frame, timeout, spec)
	readAfter, writtenAfter := session.counted.totals()
	networkBytes := (readAfter - readBefore) + (writtenAfter - writtenBefore)
	leafHash, spkiHash := sha256Hex(session.leaf.Raw), sha256Hex(session.leaf.RawSubjectPublicKeyInfo)
	result.Observation = observation
	result.Protocol = map[string]any{"requested": "grpc-over-h2", "observed": "grpc-over-h2", "tlsVersion": "TLS1.3", "alpn": "h2", "httpVersion": "HTTP/2.0", "fallbackDetected": false, "leafCertificateSha256": leafHash, "leafSpkiSha256": spkiHash, "serverName": "grpc.plab.test"}
	result.Channel = map[string]any{"configuredChannels": 1, "configuredConnections": 1, "configuredStreamsPerConnection": 1, "observedActiveConnections": 1, "observedActiveStreams": 1, "preEstablished": true, "warmupCompleted": false, "reusedForMeasuredOperation": true}
	completed, failed, deadlineExceeded, cancelled := 0, 1, 0, 0
	if observation.Passed {
		completed, failed = 1, 0
		if spec.terminalMode == "deadline-exceeded" {
			deadlineExceeded = 1
		}
		if spec.terminalMode == "client-cancellation" {
			cancelled = 1
		}
	}
	latencyMS := float64(observation.LatencyNanos) / float64(time.Millisecond)
	result.Metrics = map[string]any{"completedOperations": completed, "failedOperations": failed, "deadlineExceededOperations": deadlineExceeded, "cancelledOperations": cancelled, "timedOutOperations": 0, "effectiveConcurrency": 1, "effectiveStreams": 1, "rpcLatencyMeanMilliseconds": latencyMS, "rpcLatencyP50Milliseconds": latencyMS, "rpcLatencyP75Milliseconds": latencyMS, "rpcLatencyP90Milliseconds": latencyMS, "rpcLatencyP95Milliseconds": latencyMS, "rpcLatencyP99Milliseconds": latencyMS, "totalTransferredBytes": networkBytes, "networkBytesRead": readAfter - readBefore, "networkBytesWritten": writtenAfter - writtenBefore}
	result.Response = map[string]any{"count": 0, "httpStatus": observation.HTTPStatus, "contentType": observation.ContentType, "trailersPresent": observation.TrailersPresent, "trailersOnly": observation.TrailersOnly, "grpcStatus": observation.GRPCStatus, "grpcMessage": observation.GRPCMessage, "readyInitialMetadata": observation.ReadyInitialMetadata, "clientCancelTriggered": observation.ClientCancelTriggered, "deadlineFired": observation.DeadlineFired, "noResponseData": observation.NoResponseData, "expectedTerminalOutcome": observation.ExpectedTerminalOutcome}
	result.Passed = observation.Passed && completed == 1 && failed == 0
	result.Validation = map[string]any{"passed": result.Passed, "checks": terminalValidationChecks(spec)}
	if !result.Passed {
		return result, frame, nil, session.leaf.Raw, errors.New(observation.Error)
	}
	return result, frame, nil, session.leaf.Raw, nil
}

func invokeTerminal(parent context.Context, channel roundTripper, parsed *url.URL, requestFrame []byte, timeout time.Duration, spec scenarioSpec) rpcObservation {
	r := rpcObservation{RequestMessageCount: 1, ResponseMessageCount: 0}
	operationContext, operationCancel := context.WithTimeout(parent, timeout)
	if spec.terminalMode == "deadline-exceeded" {
		operationCancel()
		operationContext, operationCancel = context.WithTimeout(parent, 50*time.Millisecond)
	}
	if spec.terminalMode == "client-cancellation" {
		operationCancel()
		operationContext, operationCancel = context.WithCancel(parent)
	}
	defer operationCancel()
	u := *parsed
	u.Path, u.RawQuery, u.Fragment = spec.path, "", ""
	req, err := http.NewRequestWithContext(operationContext, http.MethodPost, u.String(), bytes.NewReader(requestFrame))
	if err != nil {
		r.Error = err.Error()
		return r
	}
	req.Host = "grpc.plab.test"
	req.Header.Set("Content-Type", "application/grpc+proto")
	req.Header.Set("Te", "trailers")
	req.Header.Set("grpc-encoding", "identity")
	req.Header.Set("grpc-accept-encoding", "identity")
	if spec.terminalMode == "deadline-exceeded" {
		req.Header.Set("grpc-timeout", "50m")
	}
	started := time.Now()
	resp, err := channel.RoundTrip(req)
	if spec.terminalMode == "deadline-exceeded" {
		r.LatencyNanos = time.Since(started).Nanoseconds()
		r.DeadlineFired = errors.Is(err, context.DeadlineExceeded) || errors.Is(operationContext.Err(), context.DeadlineExceeded)
		r.GRPCStatus, r.NoResponseData = "4", resp == nil
		r.ExpectedTerminalOutcome = r.DeadlineFired && r.NoResponseData && r.LatencyNanos >= int64(40*time.Millisecond) && r.LatencyNanos < int64(250*time.Millisecond)
		r.Passed = r.ExpectedTerminalOutcome
		if !r.Passed {
			r.Error = fmt.Sprintf("deadline outcome mismatch: err=%v elapsed=%s", err, time.Duration(r.LatencyNanos))
		}
		return r
	}
	if err != nil {
		r.LatencyNanos = time.Since(started).Nanoseconds()
		r.Error = err.Error()
		return r
	}
	defer resp.Body.Close()
	r.HTTPStatus, r.HTTPVersion = resp.StatusCode, resp.Proto
	r.ContentType, r.GRPCEncoding = resp.Header.Get("Content-Type"), resp.Header.Get("grpc-encoding")
	if spec.terminalMode == "client-cancellation" {
		r.ReadyInitialMetadata = resp.Header.Get("x-plab-ready") == "1"
		time.Sleep(10 * time.Millisecond)
		operationCancel()
		body, readErr := io.ReadAll(resp.Body)
		r.LatencyNanos = time.Since(started).Nanoseconds()
		r.ClientCancelTriggered = errors.Is(readErr, context.Canceled) || errors.Is(operationContext.Err(), context.Canceled)
		r.GRPCStatus, r.NoResponseData = "1", len(body) == 0
		r.ExpectedTerminalOutcome = r.ReadyInitialMetadata && r.ClientCancelTriggered && r.NoResponseData && resp.ProtoMajor == 2 && resp.StatusCode == http.StatusOK
		r.Passed = r.ExpectedTerminalOutcome
		if !r.Passed {
			r.Error = fmt.Sprintf("cancellation outcome mismatch: ready=%t cancel=%t bytes=%d err=%v", r.ReadyInitialMetadata, r.ClientCancelTriggered, len(body), readErr)
		}
		return r
	}
	body, readErr := io.ReadAll(resp.Body)
	r.LatencyNanos = time.Since(started).Nanoseconds()
	r.GRPCStatus, r.GRPCMessage = resp.Header.Get("grpc-status"), resp.Header.Get("grpc-message")
	r.TrailersOnly = r.GRPCStatus != "" && len(body) == 0
	r.TrailersPresent = r.TrailersOnly
	r.NoResponseData = len(body) == 0
	r.ExpectedTerminalOutcome = readErr == nil && resp.ProtoMajor == 2 && resp.StatusCode == http.StatusOK && strings.EqualFold(strings.TrimSpace(strings.SplitN(r.ContentType, ";", 2)[0]), "application/grpc+proto") && r.GRPCStatus == spec.expectedStatus && r.GRPCMessage == spec.expectedMessage && r.TrailersOnly
	r.Passed = r.ExpectedTerminalOutcome
	if !r.Passed {
		r.Error = fmt.Sprintf("trailers-only outcome mismatch: status=%q message=%q bytes=%d err=%v", r.GRPCStatus, r.GRPCMessage, len(body), readErr)
	}
	return r
}

func executeNewChannel(targetURL, rootPath string, timeout time.Duration, spec scenarioSpec, loadProfileID string) (executionResult, []byte, []byte, []byte, error) {
	result := baseResult(spec, loadProfileID)
	frame, _ := encodeFrame(encodeProtobuf(spec.payload), spec.compression)
	parsed, roots, err := loadTargetAndRoots(targetURL, rootPath)
	if err != nil {
		return result, frame, nil, nil, err
	}
	latencies := make([]int64, 0, 10)
	var totalRead, totalWritten int64
	var last rpcObservation
	var peer []byte
	completed := 0
	for operation := 0; operation < 10; operation++ {
		operationContext, cancel := context.WithTimeout(context.Background(), timeout)
		started := time.Now()
		session, openErr := openChannel(operationContext, parsed, roots, timeout)
		if openErr != nil {
			cancel()
			return result, frame, nil, peer, fmt.Errorf("new channel operation %d: %w", operation, openErr)
		}
		observation := invoke(operationContext, session.channel, parsed, frame, spec.deadline, spec)
		observation.LatencyNanos = time.Since(started).Nanoseconds()
		readBytes, writtenBytes := session.counted.totals()
		totalRead += readBytes
		totalWritten += writtenBytes
		peer = append([]byte(nil), session.leaf.Raw...)
		session.close()
		cancel()
		if !observation.Passed {
			return result, frame, observation.ResponseFrame, peer, fmt.Errorf("new channel operation %d: %s", operation, observation.Error)
		}
		completed++
		latencies = append(latencies, observation.LatencyNanos)
		last = observation
	}
	leaf, _ := x509.ParseCertificate(peer)
	result.Observation = last
	result.Protocol = map[string]any{"requested": "grpc-over-h2-new-channel", "observed": "grpc-over-h2-new-channel", "tlsVersion": "TLS1.3", "alpn": "h2", "httpVersion": last.HTTPVersion, "fallbackDetected": false, "leafCertificateSha256": sha256Hex(peer), "leafSpkiSha256": sha256Hex(leaf.RawSubjectPublicKeyInfo), "serverName": "grpc.plab.test"}
	result.Channel = map[string]any{"configuredChannels": 1, "configuredConnections": 1, "configuredStreamsPerConnection": 1, "channelsCreated": 10, "connectionsEstablished": 10, "observedActiveConnections": 1, "observedActiveStreams": 1, "preEstablished": false, "warmupCompleted": false, "reusedForMeasuredOperation": false, "newChannelPerOperation": true, "channelDisposedAfterEachOperation": true}
	result.Response = map[string]any{"count": 1, "payloadBytes": len(spec.payload), "payloadSha256": spec.payloadHash, "serializedProtobufBytes": spec.protobufBytes, "serializedProtobufSha256": spec.protobufHash, "grpcFrameBytes": spec.frameBytes, "grpcFrameSha256": spec.frameHash, "httpStatus": last.HTTPStatus, "contentType": last.ContentType, "trailersPresent": last.TrailersPresent, "grpcStatus": last.GRPCStatus}
	result.Metrics = map[string]any{"completedOperations": completed, "failedOperations": 0, "deadlineExceededOperations": 0, "cancelledOperations": 0, "timedOutOperations": 0, "effectiveConcurrency": 1, "effectiveStreams": 1, "channelAndRpcLatencyMeanMilliseconds": meanMilliseconds(latencies), "channelAndRpcLatencyP50Milliseconds": percentileMilliseconds(latencies, .50), "channelAndRpcLatencyP75Milliseconds": percentileMilliseconds(latencies, .75), "channelAndRpcLatencyP90Milliseconds": percentileMilliseconds(latencies, .90), "channelAndRpcLatencyP95Milliseconds": percentileMilliseconds(latencies, .95), "channelAndRpcLatencyP99Milliseconds": percentileMilliseconds(latencies, .99), "totalTransferredBytes": totalRead + totalWritten, "networkBytesRead": totalRead, "networkBytesWritten": totalWritten}
	result.Passed = completed == 10
	result.Validation = map[string]any{"passed": result.Passed, "checks": []string{"tls-version:1.3", "alpn:h2", "no-fallback", "fresh-tls-connection-per-operation", "fresh-http2-channel-per-operation", "settings-observed", "channel-disposed-after-operation", "request-count:1", "response-count:1", "payload-sha256", "grpc-status:0", "zero-failures", "zero-deadlines", "zero-cancellations", "zero-timeouts"}}
	return result, frame, last.ResponseFrame, peer, nil
}

func terminalValidationChecks(spec scenarioSpec) []string {
	base := []string{"tls-version:1.3", "alpn:h2", "no-fallback", "pre-established-channel", "retry-disabled", "hedging-disabled", "request-count:1", "response-count:0", "zero-unexpected-failures", "zero-timeouts"}
	switch spec.terminalMode {
	case "trailers-only":
		return append(base, "http-status:200", "trailers-only", "no-response-data", "grpc-status:3", "grpc-message:plab-invalid-fixture")
	case "deadline-exceeded":
		return append(base, "grpc-timeout:50m", "client-deadline-fired", "grpc-status:4", "no-response-data", "expected-terminal-outcome")
	case "client-cancellation":
		return append(base, "ready-initial-metadata", "client-cancel-after-ready", "grpc-status:1", "no-response-data", "expected-terminal-outcome")
	default:
		return base
	}
}

type roundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

func invoke(parent context.Context, channel roundTripper, parsed *url.URL, requestFrames []byte, timeout time.Duration, spec scenarioSpec) rpcObservation {
	r := rpcObservation{}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	u := *parsed
	u.Path, u.RawQuery, u.Fragment = spec.path, "", ""
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(requestFrames))
	if err != nil {
		r.Error = err.Error()
		return r
	}
	req.Host = "grpc.plab.test"
	req.Header.Set("Content-Type", "application/grpc+proto")
	req.Header.Set("Te", "trailers")
	req.Header.Set("grpc-encoding", spec.compression)
	req.Header.Set("grpc-accept-encoding", spec.compression)
	if spec.metadataProfile == "fixed-ascii-and-binary-metadata-v1" {
		req.Header.Set("x-plab-text", "protocol-lab")
		req.Header.Set("x-plab-bin-bin", "AAECAw==")
	}
	started := time.Now()
	resp, err := channel.RoundTrip(req)
	r.LatencyNanos = time.Since(started).Nanoseconds()
	if err != nil {
		r.TimedOut = errors.Is(err, context.DeadlineExceeded) || isTimeout(err)
		r.Error = err.Error()
		return r
	}
	defer resp.Body.Close()
	body, messageArrivalNanos, responseCount, readErr := readAndValidateResponseFrames(resp.Body, spec, started)
	r.ResponseFrame = body
	r.LatencyNanos = time.Since(started).Nanoseconds()
	r.HTTPStatus, r.HTTPVersion = resp.StatusCode, resp.Proto
	r.ContentType = resp.Header.Get("Content-Type")
	r.GRPCEncoding = resp.Header.Get("grpc-encoding")
	r.ResponseInitialText = resp.Header.Get("x-plab-text")
	r.TrailersPresent = resp.Trailer != nil && resp.Trailer.Get("grpc-status") != ""
	r.GRPCStatus = resp.Trailer.Get("grpc-status")
	r.ResponseTrailingBinary = resp.Trailer.Get("x-plab-bin-bin")
	r.ResponseFrameBytes, r.ResponseFrameHash = len(body), sha256Hex(body)
	r.RequestMessageCount = spec.requestCount
	r.ResponseMessageCount = responseCount
	r.ClientHalfClosed = spec.requestCount > 1
	r.OrderedEcho = responseCount == spec.responseCount
	r.StreamComplete = readErr == nil && responseCount == spec.responseCount
	r.MessageArrivalNanos = messageArrivalNanos
	if len(messageArrivalNanos) > 0 {
		r.TimeToFirstMessageNanos = messageArrivalNanos[0]
	}
	problems := []string{}
	if readErr != nil {
		problems = append(problems, readErr.Error())
	}
	if resp.ProtoMajor != 2 || resp.ProtoMinor != 0 {
		problems = append(problems, "HTTP/2 fallback detected")
	}
	if resp.StatusCode != http.StatusOK {
		problems = append(problems, fmt.Sprintf("expected HTTP 200, observed %d", resp.StatusCode))
	}
	if !strings.EqualFold(strings.TrimSpace(strings.SplitN(r.ContentType, ";", 2)[0]), "application/grpc+proto") {
		problems = append(problems, "unexpected response content type")
	}
	if r.GRPCStatus != "0" || !r.TrailersPresent {
		problems = append(problems, "missing successful gRPC trailers")
	}
	if responseCount != spec.responseCount {
		problems = append(problems, fmt.Sprintf("expected %d response messages, observed %d", spec.responseCount, responseCount))
	}
	if (spec.rpcType == "client-streaming" || spec.rpcType == "bidirectional-streaming") && !r.ClientHalfClosed {
		problems = append(problems, "client half-close was not proven")
	}
	if spec.compression == "gzip" && r.GRPCEncoding != "gzip" {
		problems = append(problems, "gzip response encoding not proven")
	}
	if spec.metadataProfile == "fixed-ascii-and-binary-metadata-v1" && (r.ResponseInitialText != "protocol-lab" || !matchesBinaryMetadata(r.ResponseTrailingBinary)) {
		problems = append(problems, "fixed response metadata mismatch")
	}
	r.Passed = len(problems) == 0
	r.Error = strings.Join(problems, "; ")
	return r
}

func matchesBinaryMetadata(value string) bool {
	decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(value, "="))
	return err == nil && bytes.Equal(decoded, []byte{0, 1, 2, 3})
}

func encodeProtobuf(payload []byte) []byte {
	if len(payload) == 0 {
		return []byte{}
	}
	out := []byte{0x0a}
	n := uint64(len(payload))
	for n >= 0x80 {
		out = append(out, byte(n)|0x80)
		n >>= 7
	}
	out = append(out, byte(n))
	return append(out, payload...)
}
func encodeFrame(protobuf []byte, compression string) ([]byte, error) {
	message := protobuf
	flag := byte(0)
	if compression == "gzip" {
		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		if _, err := w.Write(protobuf); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		message = b.Bytes()
		flag = 1
	}
	frame := make([]byte, 5, 5+len(message))
	frame[0] = flag
	binary.BigEndian.PutUint32(frame[1:], uint32(len(message)))
	return append(frame, message...), nil
}
func decodeFrame(frame []byte, compression string) ([]byte, []byte, error) {
	if len(frame) < 5 || int(binary.BigEndian.Uint32(frame[1:5])) != len(frame)-5 {
		return nil, nil, errors.New("invalid gRPC envelope")
	}
	message := frame[5:]
	if compression == "gzip" {
		if frame[0] != 1 {
			return nil, nil, errors.New("gzip compressed flag mismatch")
		}
		r, err := gzip.NewReader(bytes.NewReader(message))
		if err != nil {
			return nil, nil, err
		}
		decompressed, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			return nil, nil, err
		}
		message = decompressed
	} else if frame[0] != 0 {
		return nil, nil, errors.New("identity compressed flag mismatch")
	}
	payload, err := decodeProtobuf(message)
	return message, payload, err
}
func decodeProtobuf(p []byte) ([]byte, error) {
	if len(p) == 0 {
		return []byte{}, nil
	}
	if p[0] != 0x0a {
		return nil, errors.New("unexpected protobuf field")
	}
	var n uint64
	var shift uint
	idx := 1
	for {
		if idx >= len(p) || shift > 63 {
			return nil, errors.New("invalid protobuf length")
		}
		b := p[idx]
		idx++
		n |= uint64(b&0x7f) << shift
		if b < 0x80 {
			break
		}
		shift += 7
	}
	if int(n) != len(p)-idx {
		return nil, errors.New("protobuf payload length mismatch")
	}
	return p[idx:], nil
}
func validateResponseFrame(frame []byte, spec scenarioSpec) error {
	protobuf, payload, err := decodeFrame(frame, spec.compression)
	if err != nil {
		return err
	}
	if len(payload) != len(spec.payload) || sha256Hex(payload) != spec.payloadHash || len(protobuf) != spec.protobufBytes || sha256Hex(protobuf) != spec.protobufHash {
		return errors.New("response payload/protobuf byte-scope mismatch")
	}
	if spec.compression == "identity" && (len(frame) != spec.frameBytes || sha256Hex(frame) != spec.frameHash) {
		return errors.New("identity response frame mismatch")
	}
	return nil
}

func readAndValidateResponseFrames(reader io.Reader, spec scenarioSpec, started time.Time) ([]byte, []int64, int, error) {
	var aggregate bytes.Buffer
	arrivalNanos := make([]int64, 0, spec.responseCount)
	for index := 0; index < spec.responseCount; index++ {
		header := make([]byte, 5)
		if _, err := io.ReadFull(reader, header); err != nil {
			return aggregate.Bytes(), arrivalNanos, index, fmt.Errorf("response message %d header: %w", index, err)
		}
		length := int(binary.BigEndian.Uint32(header[1:]))
		if length < 0 || length > 2<<20 {
			return aggregate.Bytes(), arrivalNanos, index, fmt.Errorf("response message %d length is out of range", index)
		}
		message := make([]byte, length)
		if _, err := io.ReadFull(reader, message); err != nil {
			return aggregate.Bytes(), arrivalNanos, index, fmt.Errorf("response message %d body: %w", index, err)
		}
		frame := append(header, message...)
		if err := validateResponseFrame(frame, spec); err != nil {
			return aggregate.Bytes(), arrivalNanos, index, fmt.Errorf("response message %d: %w", index, err)
		}
		arrivalNanos = append(arrivalNanos, time.Since(started).Nanoseconds())
		aggregate.Write(frame)
	}
	var extra [1]byte
	if n, err := reader.Read(extra[:]); n != 0 || (err != nil && !errors.Is(err, io.EOF)) {
		return aggregate.Bytes(), arrivalNanos, spec.responseCount, errors.New("response stream contains unexpected data after the canonical message count")
	}
	return aggregate.Bytes(), arrivalNanos, spec.responseCount, nil
}

func baseResult(spec scenarioSpec, loadProfileID string) executionResult {
	duration, warmup, repetition, totalRpcs := 5, 1, 1, 1
	if loadProfileID == loadProfileDiagnostic {
		duration, warmup = 10, 0
	}
	if loadProfileID == loadProfileChannelChurn {
		duration, warmup, repetition, totalRpcs = 30, 0, 3, 10
	}
	frame, _ := encodeFrame(encodeProtobuf(spec.payload), spec.compression)
	request := map[string]any{"count": spec.requestCount, "payloadBytes": len(spec.payload), "payloadBytesPerMessage": len(spec.payload), "payloadSha256": spec.payloadHash, "serializedProtobufBytes": spec.protobufBytes, "serializedProtobufBytesPerMessage": spec.protobufBytes, "serializedProtobufSha256": spec.protobufHash, "grpcFrameBytes": len(frame), "grpcFrameBytesPerMessage": len(frame), "grpcFrameSha256": sha256Hex(frame), "aggregateGrpcFrameBytes": spec.requestCount * len(frame), "aggregateGrpcFramesSha256": sha256Hex(bytes.Repeat(frame, spec.requestCount)), "compression": spec.compression, "metadataProfile": spec.metadataProfile, "method": spec.method, "path": spec.path, "rpcType": spec.rpcType, "clientHalfCloseRequired": spec.requestCount > 1}
	if spec.metadataProfile == "fixed-ascii-and-binary-metadata-v1" {
		request["requestInitialTextMetadata"] = "protocol-lab"
		request["requestInitialBinaryMetadata"] = "AAECAw=="
		request["requestInitialBinaryMetadataDecodedSha256"] = sha256Hex([]byte{0, 1, 2, 3})
	}
	return executionResult{SchemaVersion: "protocol-lab.grpc-h2-executor-result.v1", ExecutorID: executorID, ExecutorVersion: executorVersion, LoadGeneratorID: loadGeneratorID, LoadGeneratorVersion: loadGeneratorVersion, ScenarioID: spec.id, LoadProfileID: loadProfileID, ServiceContractSha256: serviceContractDigest, Request: request, RequestedLoad: map[string]any{"connections": 1, "concurrency": 1, "streamsPerConnection": 1, "durationSeconds": duration, "warmupSeconds": warmup, "repetition": repetition, "totalRpcs": totalRpcs}, EffectiveLoad: map[string]any{"connections": 1, "activeConnections": 1, "concurrency": 1, "streamsPerConnection": 1, "activeStreams": 1, "durationSeconds": duration, "warmupSeconds": warmup, "repetition": repetition, "totalRpcs": totalRpcs}, Response: map[string]any{}, Metrics: map[string]any{"completedOperations": 0, "failedOperations": 1, "deadlineExceededOperations": 0, "cancelledOperations": 0, "timedOutOperations": 0}, Validation: map[string]any{"passed": false}, Artifacts: []string{"validation.json", "protocol-proof.json", "grpc-summary.json", "tls-negotiation.json", "result.json", "executor-identity.json", "load-generator-identity.json", "grpc-request-frame.bin", "grpc-response-frame.bin", "tls-peer-certificate.der", "gzip-encoder-provenance.json"}}
}
func validationChecks(s scenarioSpec) []string {
	checks := []string{"tls-version:1.3", "alpn:h2", "no-fallback", "pre-established-channel", "channel-reused", fmt.Sprintf("request-count:%d", s.requestCount), fmt.Sprintf("response-count:%d", s.responseCount), "payload-sha256", "protobuf-sha256", "http-status:200", "content-type:application/grpc+proto", "trailers-present", "grpc-status:0", "zero-failures", "zero-deadlines", "zero-cancellations", "zero-timeouts"}
	if s.compression == "gzip" {
		checks = append(checks, "grpc-encoding:gzip", "compressed-flag:1", "decompress-success", "gzip-encoder-provenance")
	} else {
		checks = append(checks, "grpc-frame-sha256")
	}
	if s.metadataProfile == "fixed-ascii-and-binary-metadata-v1" {
		checks = append(checks, "request-ascii-metadata", "request-binary-metadata-decoded-bytes", "response-initial-ascii-metadata", "response-trailing-binary-metadata-decoded-bytes")
	}
	if s.requestCount > 1 {
		checks = append(checks, "client-half-close")
	}
	if s.responseCount > 1 {
		checks = append(checks, "stream-complete")
	}
	if s.rpcType == "bidirectional-streaming" {
		checks = append(checks, "ordered-one-to-one-echo")
	}
	return checks
}

func writeArtifacts(dir string, result executionResult, request, response, peer []byte) error {
	values := map[string]any{"validation.json": result.Validation, "protocol-proof.json": result.Protocol, "grpc-summary.json": result, "tls-negotiation.json": result.Protocol, "result.json": result, "executor-identity.json": map[string]any{"executorId": executorID, "executorVersion": executorVersion, "role": "client-test-executor", "supportedScenarios": supportedScenarioIDs, "supportedLoadProfiles": []string{loadProfileSmoke, loadProfileDiagnostic, loadProfileChannelChurn}, "stdoutStderrOwner": "invoking-runner-or-package-host"}, "load-generator-identity.json": map[string]any{"loadGeneratorId": loadGeneratorID, "loadGeneratorVersion": loadGeneratorVersion, "engine": "golang.org/x/net/http2", "role": "test-side-load-generator"}, "gzip-encoder-provenance.json": map[string]any{"encoder": "Go standard library compress/gzip", "executorVersion": executorVersion, "semanticValidation": "decompress-then-compare-uncompressed-protobuf-sha256", "wireBytesComparable": false}}
	for name, value := range values {
		if err := writeJSON(filepath.Join(dir, name), value); err != nil {
			return err
		}
	}
	for name, value := range map[string][]byte{"grpc-request-frame.bin": request, "grpc-response-frame.bin": response, "tls-peer-certificate.der": peer} {
		if err := os.WriteFile(filepath.Join(dir, name), value, 0o644); err != nil {
			return err
		}
	}
	return nil
}
func writeUnsupported(dir, id string) {
	_ = writeJSON(filepath.Join(dir, "unsupported.json"), map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "scenarioId": id, "executorId": executorID, "executorVersion": executorVersion, "outcome": "unsupported", "reason": "exact committed scenario semantics are not implemented"})
}

func validateSelection(spec scenarioSpec) (string, error) {
	expectedVariant := protocolVariant
	if spec.lifecycle == "new-channel-per-operation" {
		expectedVariant = protocolVariantNewChannel
	}
	expected := map[string]string{"PLAB_EXECUTOR_ID": executorID, "PLAB_EXECUTOR_VERSION": executorVersion, "PLAB_LOAD_GENERATOR_ID": loadGeneratorID, "PLAB_LOAD_GENERATOR_VERSION": loadGeneratorVersion, "PLAB_SCENARIO_ID": spec.id, "PLAB_PROTOCOL": "h2", "PLAB_PROTOCOL_VARIANT": expectedVariant}
	for name, required := range expected {
		if observed := strings.TrimSpace(os.Getenv(name)); observed != required {
			return "", fmt.Errorf("%s %q is unsupported; expected %q", name, observed, required)
		}
	}
	profile := strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID"))
	if profile != loadProfileSmoke && profile != loadProfileDiagnostic && profile != loadProfileChannelChurn {
		return "", fmt.Errorf("PLAB_LOAD_PROFILE_ID %q is unsupported", profile)
	}
	if profile == loadProfileSmoke && spec.id != "grpc.h2.unary.echo" && spec.id != "grpc.h2.server-streaming.echo" && spec.id != "grpc.h2.client-streaming.echo" && spec.id != "grpc.h2.bidi-streaming.echo" {
		return "", fmt.Errorf("%s is not bound to %s", profile, spec.id)
	}
	if spec.terminalMode != "" && profile != loadProfileDiagnostic {
		return "", fmt.Errorf("%s requires %s", spec.id, loadProfileDiagnostic)
	}
	if spec.lifecycle == "new-channel-per-operation" && profile != loadProfileChannelChurn {
		return "", fmt.Errorf("%s requires %s", spec.id, loadProfileChannelChurn)
	}
	if profile == loadProfileDiagnostic && spec.terminalMode == "" && spec.lifecycle != "new-channel-per-operation" && spec.id != "grpc.h2.unary.empty" && spec.id != "grpc.h2.unary.fixed-metadata" && spec.id != "grpc.h2.unary.gzip" && spec.id != "grpc.h2.unary.large" {
		return "", fmt.Errorf("%s is not bound to %s", profile, spec.id)
	}
	expectedDuration, expectedWarmup, expectedRepetition := "5", "1", "1"
	if profile == loadProfileDiagnostic {
		expectedDuration, expectedWarmup = "10", "0"
	}
	if profile == loadProfileChannelChurn {
		expectedDuration, expectedWarmup, expectedRepetition = "30", "0", "3"
	}
	expectedLoad := map[string]string{"PLAB_CONNECTIONS": "1", "PLAB_CONCURRENCY": "1", "PLAB_STREAMS_PER_CONNECTION": "1", "PLAB_DURATION_SECONDS": expectedDuration, "PLAB_WARMUP_SECONDS": expectedWarmup, "PLAB_REPETITION": expectedRepetition}
	for name, required := range expectedLoad {
		if observed := strings.TrimSpace(os.Getenv(name)); observed != required {
			return "", fmt.Errorf("%s %q is unsupported; expected %q for %s", name, observed, required, profile)
		}
	}
	return profile, nil
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func durationFromEnvironment(name string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(v)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
func defaultMaterialPath(relative string) string {
	if executable, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(executable), "..", "..", relative)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", relative)
}
func envOrDefault(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}
func sha256Hex(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
func meanMilliseconds(values []int64) float64 {
	var total int64
	for _, value := range values {
		total += value
	}
	return float64(total) / float64(len(values)) / float64(time.Millisecond)
}
func percentileMilliseconds(values []int64, quantile float64) float64 {
	copyValues := append([]int64(nil), values...)
	slices.Sort(copyValues)
	index := int(math.Ceil(quantile*float64(len(copyValues)))) - 1
	if index < 0 {
		index = 0
	}
	return float64(copyValues[index]) / float64(time.Millisecond)
}
func fatal(code int, message string) { fmt.Fprintln(os.Stderr, message); os.Exit(code) }
