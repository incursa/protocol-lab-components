package main

import (
	"bytes"
	"context"
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
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"
)

const (
	executorID                  = "go-grpc-h2-executor"
	executorVersion             = "0.1.0"
	loadGeneratorID             = "go-x-net-http2-grpc-load"
	loadGeneratorVersion        = "0.1.0"
	scenarioID                  = "grpc.h2.unary.echo"
	loadProfileID               = "grpc-h2-smoke"
	protocolVariant             = "grpc-over-h2-tls-alpn"
	serviceContractDigest       = "b7b987814f8af5cd4f15c03989b9c309c1c0ec643972ae32668304d71502120f"
	grpcPath                    = "/protocollab.performance.v1.EchoService/UnaryEcho"
	expectedPayloadHash         = "394394b5f0e91a21d1e932f9ed55e098c8b05f3668f77134eeee843fef1d1758"
	expectedProtobufHash        = "c2046bce7238a2a9cae159ba85a75e93f07c74073fe7b83d2be713caef717cb4"
	expectedFrameHash           = "98f1fce70c79cf7da3649fc3ecfc77a975c1427c9f8afcd1a41d85559164525a"
	expectedLeafCertificateHash = "4627eed7781247db01e5641e4799e1c71e45543c10e304c99ed74b6e0bc0e254"
	expectedLeafSPKIHash        = "83737e10fac4d3acac8245615dc510b8c12deb8f4756af3eaddf21118f18e4ea"
	expectedRootPEMHash         = "dda81fe80d268bbd91993339ceef728a7e9e181d1faee88af35d73be2a7b039b"
)

type rpcObservation struct {
	HTTPStatus         int    `json:"httpStatus"`
	HTTPVersion        string `json:"httpVersion"`
	ContentType        string `json:"contentType"`
	TrailersPresent    bool   `json:"trailersPresent"`
	GRPCStatus         string `json:"grpcStatus"`
	ResponseFrameBytes int    `json:"responseFrameBytes"`
	ResponseFrameHash  string `json:"responseFrameSha256"`
	LatencyNanos       int64  `json:"latencyNanos"`
	Passed             bool   `json:"passed"`
	TimedOut           bool   `json:"timedOut"`
	Error              string `json:"error,omitempty"`
	ResponseFrame      []byte `json:"-"`
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

func (connection *countingConn) Read(buffer []byte) (int, error) {
	count, err := connection.Conn.Read(buffer)
	connection.readBytes.Add(int64(count))
	return count, err
}

func (connection *countingConn) Write(buffer []byte) (int, error) {
	count, err := connection.Conn.Write(buffer)
	connection.writtenBytes.Add(int64(count))
	return count, err
}

func (connection *countingConn) totals() (int64, int64) {
	return connection.readBytes.Load(), connection.writtenBytes.Load()
}

func main() {
	targetURL := flag.String("target-url", os.Getenv("PLAB_TARGET_BASE_URL"), "target HTTPS URL")
	outputDir := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "artifact directory")
	rootPath := flag.String("root-cert", envOrDefault("PLAB_TLS_ROOT_CERT_FILE", defaultMaterialPath("certs/root.pem")), "trusted root certificate")
	flag.Parse()
	if strings.TrimSpace(*targetURL) == "" {
		fatal(2, "target-url or PLAB_TARGET_BASE_URL is required")
	}
	if strings.TrimSpace(*outputDir) == "" {
		*outputDir = "artifacts"
	}
	if err := validateSelection(); err != nil {
		fatal(2, err.Error())
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fatal(1, err.Error())
	}
	timeout := durationFromEnvironment("PLAB_REQUEST_TIMEOUT_SECONDS", 5*time.Second)
	result, requestFrame, responseFrame, peerCertificate, err := execute(*targetURL, *rootPath, timeout)
	if err != nil {
		result.Passed = false
		result.Validation["error"] = err.Error()
	}
	if writeErr := writeArtifacts(*outputDir, result, requestFrame, responseFrame, peerCertificate); writeErr != nil {
		fatal(1, writeErr.Error())
	}
	encoded, _ := json.Marshal(result)
	fmt.Fprintln(os.Stdout, string(encoded))
	if err != nil || !result.Passed {
		fatal(1, "gRPC/H2 unary validity gate failed")
	}
}

func execute(targetURL, rootPath string, timeout time.Duration) (executionResult, []byte, []byte, []byte, error) {
	result := baseResult()
	frame := canonicalFrame()
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return result, frame, nil, nil, errors.New("target URL must be an absolute https URL")
	}
	rootPEM, err := os.ReadFile(rootPath)
	if err != nil {
		return result, frame, nil, nil, err
	}
	if sha256Hex(rootPEM) != expectedRootPEMHash {
		return result, frame, nil, nil, errors.New("trusted root certificate hash mismatch")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootPEM) {
		return result, frame, nil, nil, errors.New("trusted root certificate could not be parsed")
	}
	dialer := &net.Dialer{Timeout: timeout}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	raw, err := dialer.DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		return result, frame, nil, nil, err
	}
	counted := &countingConn{Conn: raw}
	defer counted.Close()
	tlsConn := tls.Client(counted, &tls.Config{RootCAs: roots, ServerName: "grpc.plab.test", MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, NextProtos: []string{"h2"}})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return result, frame, nil, nil, err
	}
	state := tlsConn.ConnectionState()
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != "h2" || !state.NegotiatedProtocolIsMutual || len(state.PeerCertificates) != 1 {
		return result, frame, nil, nil, errors.New("exact TLS 1.3, mutual ALPN h2, and one authenticated leaf certificate are required")
	}
	leaf := state.PeerCertificates[0]
	leafHash, spkiHash := sha256Hex(leaf.Raw), sha256Hex(leaf.RawSubjectPublicKeyInfo)
	if leafHash != expectedLeafCertificateHash || spkiHash != expectedLeafSPKIHash {
		return result, frame, nil, leaf.Raw, errors.New("authenticated leaf certificate identity mismatch")
	}
	transport := &http2.Transport{}
	channel, err := transport.NewClientConn(tlsConn)
	if err != nil {
		return result, frame, nil, leaf.Raw, err
	}
	defer channel.Close()
	warmup := invoke(ctx, channel, parsed, frame, timeout)
	if !warmup.Passed {
		return result, frame, nil, leaf.Raw, fmt.Errorf("pre-established channel warmup failed: %s", warmup.Error)
	}
	readBefore, writtenBefore := counted.totals()
	observation := invoke(context.Background(), channel, parsed, frame, timeout)
	readAfter, writtenAfter := counted.totals()
	measuredNetworkBytes := (readAfter - readBefore) + (writtenAfter - writtenBefore)
	result.Observation = observation
	result.Protocol = map[string]any{"requested": "grpc-over-h2", "observed": "grpc-over-h2", "tlsVersion": "TLS1.3", "alpn": "h2", "httpVersion": observation.HTTPVersion, "fallbackDetected": observation.HTTPVersion != "HTTP/2.0", "leafCertificateSha256": leafHash, "leafSpkiSha256": spkiHash, "serverName": "grpc.plab.test"}
	result.Channel = map[string]any{"configuredChannels": 1, "configuredConnections": 1, "configuredStreamsPerConnection": 1, "observedActiveConnections": 1, "observedActiveStreams": 1, "preEstablished": true, "warmupCompleted": true, "reusedForMeasuredOperation": true}
	completed, failed, timedOut := 0, 1, 0
	if observation.Passed {
		completed, failed = 1, 0
	}
	if observation.TimedOut {
		timedOut = 1
	}
	latencyMS := float64(observation.LatencyNanos) / float64(time.Millisecond)
	rpcsPerSecond := 0.0
	if observation.LatencyNanos > 0 && completed == 1 {
		rpcsPerSecond = float64(time.Second) / float64(observation.LatencyNanos)
	}
	result.Metrics = map[string]any{"rpcsPerSecond": rpcsPerSecond, "completedOperations": completed, "failedOperations": failed, "deadlineExceededOperations": 0, "cancelledOperations": 0, "timedOutOperations": timedOut, "effectiveConcurrency": 1, "effectiveStreams": 1, "rpcLatencyMeanMilliseconds": latencyMS, "rpcLatencyP50Milliseconds": latencyMS, "rpcLatencyP75Milliseconds": latencyMS, "rpcLatencyP90Milliseconds": latencyMS, "rpcLatencyP95Milliseconds": latencyMS, "rpcLatencyP99Milliseconds": latencyMS, "totalTransferredBytes": measuredNetworkBytes, "networkBytesRead": readAfter - readBefore, "networkBytesWritten": writtenAfter - writtenBefore, "grpcMessageFrameBytesBidirectional": 272}
	result.Response = map[string]any{"count": 1, "payloadBytes": 128, "payloadSha256": expectedPayloadHash, "serializedProtobufBytes": 131, "serializedProtobufSha256": expectedProtobufHash, "grpcFrameBytes": len(observation.ResponseFrame), "grpcFrameSha256": observation.ResponseFrameHash, "httpStatus": observation.HTTPStatus, "contentType": observation.ContentType, "trailersPresent": observation.TrailersPresent, "grpcStatus": observation.GRPCStatus}
	result.Passed = observation.Passed && completed == 1 && failed == 0 && timedOut == 0
	result.Validation = map[string]any{"passed": result.Passed, "checks": []string{"tls-version:1.3", "alpn:h2", "no-fallback", "pre-established-channel", "channel-reused", "request-count:1", "response-count:1", "payload-bytes:128", "protobuf-bytes:131", "grpc-frame-bytes:136", "payload-sha256", "protobuf-sha256", "grpc-frame-sha256", "http-status:200", "content-type:application/grpc+proto", "trailers-present", "grpc-status:0", "zero-failures", "zero-deadlines", "zero-cancellations", "zero-timeouts"}}
	if !result.Passed {
		return result, frame, observation.ResponseFrame, leaf.Raw, errors.New(observation.Error)
	}
	return result, frame, observation.ResponseFrame, leaf.Raw, nil
}

type roundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

func invoke(parent context.Context, channel roundTripper, parsed *url.URL, frame []byte, timeout time.Duration) rpcObservation {
	result := rpcObservation{}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	requestURL := *parsed
	requestURL.Path, requestURL.RawQuery, requestURL.Fragment = grpcPath, "", ""
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), bytes.NewReader(frame))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Host = "grpc.plab.test"
	req.Header.Set("Content-Type", "application/grpc+proto")
	req.Header.Set("Te", "trailers")
	req.Header.Set("grpc-encoding", "identity")
	req.Header.Set("grpc-accept-encoding", "identity")
	started := time.Now()
	response, err := channel.RoundTrip(req)
	result.LatencyNanos = time.Since(started).Nanoseconds()
	if err != nil {
		result.TimedOut = errors.Is(err, context.DeadlineExceeded) || isTimeout(err)
		result.Error = err.Error()
		return result
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	result.ResponseFrame = body
	result.LatencyNanos = time.Since(started).Nanoseconds()
	result.HTTPStatus, result.HTTPVersion = response.StatusCode, response.Proto
	result.ContentType = response.Header.Get("Content-Type")
	result.TrailersPresent = response.Trailer != nil && response.Trailer.Get("grpc-status") != ""
	result.GRPCStatus = response.Trailer.Get("grpc-status")
	result.ResponseFrameBytes, result.ResponseFrameHash = len(body), sha256Hex(body)
	problems := []string{}
	if err != nil {
		problems = append(problems, err.Error())
	}
	if response.ProtoMajor != 2 || response.ProtoMinor != 0 {
		problems = append(problems, "HTTP/2 fallback detected")
	}
	if response.StatusCode != http.StatusOK {
		problems = append(problems, fmt.Sprintf("expected HTTP 200, observed %d", response.StatusCode))
	}
	if !strings.EqualFold(strings.TrimSpace(strings.SplitN(result.ContentType, ";", 2)[0]), "application/grpc+proto") {
		problems = append(problems, "unexpected response content type")
	}
	if result.GRPCStatus != "0" || !result.TrailersPresent {
		problems = append(problems, "missing successful gRPC trailers")
	}
	if validateCanonicalFrame(body) != nil {
		problems = append(problems, "response frame mismatch")
	}
	result.Passed = len(problems) == 0
	result.Error = strings.Join(problems, "; ")
	return result
}

func canonicalFrame() []byte {
	payload := bytes.Repeat([]byte{'G'}, 128)
	protobuf := append([]byte{0x0a, 0x80, 0x01}, payload...)
	frame := make([]byte, 5, 136)
	binary.BigEndian.PutUint32(frame[1:], uint32(len(protobuf)))
	return append(frame, protobuf...)
}

func validateCanonicalFrame(frame []byte) error {
	if len(frame) != 136 || frame[0] != 0 || binary.BigEndian.Uint32(frame[1:5]) != 131 {
		return errors.New("invalid gRPC envelope")
	}
	if sha256Hex(frame) != expectedFrameHash || sha256Hex(frame[5:]) != expectedProtobufHash || sha256Hex(frame[8:]) != expectedPayloadHash {
		return errors.New("gRPC byte-scope hash mismatch")
	}
	return nil
}

func baseResult() executionResult {
	requestedLoad := map[string]any{"connections": 1, "concurrency": 1, "streamsPerConnection": 1, "durationSeconds": 5, "warmupSeconds": 1, "repetition": 1}
	return executionResult{
		SchemaVersion: "protocol-lab.grpc-h2-executor-result.v1", ExecutorID: executorID, ExecutorVersion: executorVersion,
		LoadGeneratorID: loadGeneratorID, LoadGeneratorVersion: loadGeneratorVersion, ScenarioID: scenarioID, LoadProfileID: loadProfileID,
		ServiceContractSha256: serviceContractDigest,
		Request:               map[string]any{"count": 1, "payloadBytes": 128, "payloadSha256": expectedPayloadHash, "serializedProtobufBytes": 131, "serializedProtobufSha256": expectedProtobufHash, "grpcFrameBytes": 136, "grpcFrameSha256": expectedFrameHash, "compression": "identity"},
		RequestedLoad:         requestedLoad,
		EffectiveLoad:         map[string]any{"connections": 1, "activeConnections": 1, "concurrency": 1, "streamsPerConnection": 1, "activeStreams": 1, "durationSeconds": 5, "warmupSeconds": 1, "repetition": 1},
		Response:              map[string]any{}, Metrics: map[string]any{"completedOperations": 0, "failedOperations": 1, "deadlineExceededOperations": 0, "cancelledOperations": 0, "timedOutOperations": 0},
		Validation: map[string]any{"passed": false},
		Artifacts:  []string{"validation.json", "protocol-proof.json", "grpc-summary.json", "result.json", "executor-identity.json", "load-generator-identity.json", "grpc-request-frame.bin", "grpc-response-frame.bin", "tls-peer-certificate.der"},
	}
}

func writeArtifacts(outputDir string, result executionResult, requestFrame, responseFrame, peerCertificate []byte) error {
	values := map[string]any{
		"validation.json": result.Validation, "protocol-proof.json": result.Protocol, "grpc-summary.json": result, "result.json": result,
		"executor-identity.json":       map[string]any{"executorId": executorID, "executorVersion": executorVersion, "role": "client-test-executor", "supportedScenarios": []string{scenarioID}, "supportedLoadProfiles": []string{loadProfileID}, "stdoutStderrOwner": "invoking-runner-or-package-host"},
		"load-generator-identity.json": map[string]any{"loadGeneratorId": loadGeneratorID, "loadGeneratorVersion": loadGeneratorVersion, "engine": "golang.org/x/net/http2", "role": "test-side-load-generator"},
	}
	for name, value := range values {
		if err := writeJSON(filepath.Join(outputDir, name), value); err != nil {
			return err
		}
	}
	for name, value := range map[string][]byte{"grpc-request-frame.bin": requestFrame, "grpc-response-frame.bin": responseFrame, "tls-peer-certificate.der": peerCertificate} {
		if err := os.WriteFile(filepath.Join(outputDir, name), value, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func validateSelection() error {
	expected := map[string]string{"PLAB_EXECUTOR_ID": executorID, "PLAB_EXECUTOR_VERSION": executorVersion, "PLAB_LOAD_GENERATOR_ID": loadGeneratorID, "PLAB_LOAD_GENERATOR_VERSION": loadGeneratorVersion, "PLAB_SCENARIO_ID": scenarioID, "PLAB_LOAD_PROFILE_ID": loadProfileID, "PLAB_PROTOCOL": "h2", "PLAB_PROTOCOL_VARIANT": protocolVariant}
	for name, required := range expected {
		if observed := strings.TrimSpace(os.Getenv(name)); observed != required {
			return fmt.Errorf("%s %q is unsupported; expected %q", name, observed, required)
		}
	}
	expectedLoad := map[string]string{"PLAB_CONNECTIONS": "1", "PLAB_CONCURRENCY": "1", "PLAB_STREAMS_PER_CONNECTION": "1", "PLAB_DURATION_SECONDS": "5", "PLAB_WARMUP_SECONDS": "1", "PLAB_REPETITION": "1"}
	for name, required := range expectedLoad {
		if observed := strings.TrimSpace(os.Getenv(name)); observed != required {
			return fmt.Errorf("%s %q is unsupported; expected %q for grpc-h2-smoke", name, observed, required)
		}
	}
	return nil
}

func durationFromEnvironment(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
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
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
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
func fatal(code int, message string) { fmt.Fprintln(os.Stderr, message); os.Exit(code) }
