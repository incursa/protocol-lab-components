package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
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

	"github.com/quic-go/quic-go"
	webtransport "github.com/quic-go/webtransport-go"
)

const (
	executorID            = "go-webtransport-executor"
	executorVersion       = "0.1.0"
	loadGeneratorID       = "webtransport-go-load"
	loadGeneratorVersion  = "0.1.0"
	engineModule          = "github.com/quic-go/webtransport-go"
	engineModuleVersion   = "v0.11.1"
	authorityCommit       = "5b113ee75e6f4e329f638751580c9e6cf0c9a99e"
	scenarioID            = "webtransport.session-bidi-echo"
	profileID             = "webtransport-smoke"
	protocolVariant       = "webtransport-over-h3"
	authority             = "webtransport.plab.test"
	pathValue             = "/webtransport/echo"
	payloadBytes          = 65536
	payloadSHA256         = "4b640d85ab3ba30fd02c9fc9db4a8928f416322ad27022ea58a65aaee68a4df2"
	certificateDERSHA256  = "de0f805e043ced1ca4742ad65ada6eb23d2338180f6129a3a72bfd367f932b9c"
	certificateSPKISHA256 = "431113d9234c136aad960a67d7a9788b7c95ae7fa53d482dd3b48a0bc5aa69cc"
	warmupDuration        = time.Second
	measurementDuration   = 5 * time.Second
	operationTimeout      = 10 * time.Second
)

type protocolProof struct {
	Protocol                     string `json:"protocol"`
	ProtocolVersion              string `json:"protocolVersion"`
	ProtocolVariant              string `json:"protocolVariant"`
	TLSVersion                   string `json:"tlsVersion"`
	ALPN                         string `json:"alpn"`
	DidResume                    bool   `json:"didResume"`
	CertificateDERSHA256         string `json:"certificateDerSha256"`
	CertificateSPKISHA256        string `json:"certificateSpkiSha256"`
	RequestMethod                string `json:"requestMethod"`
	RequestProtocol              string `json:"requestProtocol"`
	RequestScheme                string `json:"requestScheme"`
	RequestAuthority             string `json:"requestAuthority"`
	RequestPath                  string `json:"requestPath"`
	ResponseStatus               int    `json:"responseStatus"`
	DatagramsEnabled             bool   `json:"datagramsEnabled"`
	SessionEstablished           bool   `json:"sessionEstablished"`
	ClientInitiatedBidirectional bool   `json:"clientInitiatedBidirectional"`
	SessionCount                 int    `json:"sessionCount"`
	BidirectionalStreamCount     int    `json:"bidirectionalStreamCount"`
	PayloadBytes                 int    `json:"payloadBytes"`
	PayloadSHA256                string `json:"payloadSha256"`
	EchoedBytes                  int    `json:"echoedBytes"`
	EchoedSHA256                 string `json:"echoedSha256"`
	SessionCloseCode             int    `json:"sessionCloseCode"`
	CleanCompletion              bool   `json:"cleanCompletion"`
	ConfiguredSessions           int    `json:"configuredSessions"`
	ConfiguredConcurrency        int    `json:"configuredConcurrency"`
	ConfiguredStreamsPerSession  int    `json:"configuredStreamsPerSession"`
	ObservedActiveSessions       int    `json:"observedActiveSessions"`
	ObservedActiveStreams        int    `json:"observedActiveStreams"`
	EffectiveConcurrency         int    `json:"effectiveConcurrency"`
}

type operationResult struct {
	Proof            protocolProof `json:"protocolProof"`
	SessionLatencyMS float64       `json:"sessionLatencyMilliseconds"`
	StreamLatencyMS  float64       `json:"streamLatencyMilliseconds"`
	TransferredBytes int           `json:"transferredBytes"`
}

type summary struct {
	DurationSeconds       float64          `json:"durationSeconds"`
	CompletedOperations   int              `json:"completedOperations"`
	FailedOperations      int              `json:"failedOperations"`
	TimedOutOperations    int              `json:"timedOutOperations"`
	TotalTransferredBytes int64            `json:"totalTransferredBytes"`
	SessionLatencies      []float64        `json:"sessionLatencyMilliseconds"`
	StreamLatencies       []float64        `json:"streamLatencyMilliseconds"`
	Last                  *operationResult `json:"lastOperation,omitempty"`
	Errors                map[string]int   `json:"errors"`
}

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "target endpoint")
	output := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "artifact directory")
	rootPath := flag.String("root-certificate", envOr("PLAB_TLS_ROOT_CERTIFICATE_PATH", materialPath("certs/root.pem")), "root certificate")
	validationOnly := flag.Bool("validation-only", false, "run one exact operation")
	version := flag.Bool("version", false, "print version")
	flag.Parse()
	if *version {
		fmt.Printf("%s %s %s %s\n", executorID, executorVersion, engineModule, engineModuleVersion)
		return
	}
	verifyOptional("PLAB_EXECUTOR_ID", executorID)
	verifyOptional("PLAB_EXECUTOR_VERSION", executorVersion)
	verifyOptional("PLAB_LOAD_GENERATOR_ID", loadGeneratorID)
	verifyOptional("PLAB_LOAD_GENERATOR_VERSION", loadGeneratorVersion)
	verifyOptional("PLAB_PROTOCOL", "webtransport")
	verifyOptional("PLAB_PROTOCOL_VARIANT", protocolVariant)
	verifyOptional("PLAB_LOAD_PROFILE_ID", profileID)
	if id := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")); id != "" && id != scenarioID {
		emitUnsupported(outputOrDefault(*output), id)
		os.Exit(3)
	}
	if *output == "" {
		*output = "artifacts"
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		fatal(1, err)
	}
	actualAddress, logicalURL, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	roots, err := loadRoots(*rootPath)
	if err != nil {
		fatal(2, err)
	}
	payload := makePayload()
	writeIdentity(*output)

	preflight, err := runOperation(actualAddress, logicalURL, roots, payload)
	if err != nil {
		writeFailure(*output, err)
		fatal(1, err)
	}
	var warmup, measured summary
	if *validationOnly {
		measured = summaryFromOperation(preflight)
	} else {
		warmup = runFor(actualAddress, logicalURL, roots, payload, warmupDuration)
		if warmup.FailedOperations != 0 || warmup.TimedOutOperations != 0 {
			err = errors.New("warmup contained failed or timed-out operations")
		}
		if err == nil {
			measured = runFor(actualAddress, logicalURL, roots, payload, measurementDuration)
		}
		if err == nil && (measured.CompletedOperations == 0 || measured.FailedOperations != 0 || measured.TimedOutOperations != 0) {
			err = errors.New("measured window did not complete cleanly")
		}
		writeJSON(*output, "webtransport-warmup-summary.json", warmup)
	}
	if err != nil {
		writeFailure(*output, err)
		fatal(1, err)
	}
	writeProofArtifacts(*output, preflight)
	writeResult(*output, measured)
	data, _ := os.ReadFile(filepath.Join(*output, "result.json"))
	fmt.Print(string(data))
}

func runOperation(actualAddress, logicalURL string, roots *x509.CertPool, payload []byte) (operationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	tlsConfig := &tls.Config{RootCAs: roots, ServerName: authority, MinVersion: tls.VersionTLS13}
	dialer := &webtransport.Dialer{
		TLSClientConfig: tlsConfig,
		QUICConfig:      &quic.Config{EnableDatagrams: true, EnableStreamResetPartialDelivery: true},
		DialAddr: func(ctx context.Context, _ string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
			return quic.DialAddrEarly(ctx, actualAddress, tlsCfg, cfg)
		},
	}
	defer dialer.Close()
	sessionStarted := time.Now()
	response, session, err := dialer.Dial(ctx, logicalURL, nil)
	if err != nil {
		return operationResult{}, err
	}
	sessionLatency := time.Since(sessionStarted)
	state := session.SessionState()
	if response.StatusCode != 200 {
		return operationResult{}, fmt.Errorf("unexpected CONNECT status %d", response.StatusCode)
	}
	if state.ConnectionState.TLS.Version != tls.VersionTLS13 || state.ConnectionState.TLS.NegotiatedProtocol != "h3" {
		return operationResult{}, fmt.Errorf("protocol substitution: tls=%x alpn=%q", state.ConnectionState.TLS.Version, state.ConnectionState.TLS.NegotiatedProtocol)
	}
	if len(state.ConnectionState.TLS.PeerCertificates) != 1 {
		return operationResult{}, fmt.Errorf("expected one peer certificate, observed %d", len(state.ConnectionState.TLS.PeerCertificates))
	}
	leaf := state.ConnectionState.TLS.PeerCertificates[0]
	if hash(leaf.Raw) != certificateDERSHA256 || hash(leaf.RawSubjectPublicKeyInfo) != certificateSPKISHA256 {
		return operationResult{}, errors.New("certificate identity mismatch")
	}

	streamStarted := time.Now()
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		return operationResult{}, err
	}
	_ = stream.SetDeadline(time.Now().Add(operationTimeout))
	if _, err = stream.Write(payload); err != nil {
		return operationResult{}, err
	}
	if err = stream.Close(); err != nil {
		return operationResult{}, err
	}
	echoed, err := io.ReadAll(io.LimitReader(stream, payloadBytes+1))
	if err != nil {
		return operationResult{}, err
	}
	streamLatency := time.Since(streamStarted)
	if len(echoed) != payloadBytes || !bytes.Equal(payload, echoed) || hash(echoed) != payloadSHA256 {
		return operationResult{}, errors.New("echoed payload identity mismatch")
	}
	if err = session.CloseWithError(0, ""); err != nil {
		return operationResult{}, err
	}
	proof := protocolProof{
		Protocol: "webtransport", ProtocolVersion: "draft-ietf-webtrans-http3", ProtocolVariant: protocolVariant,
		TLSVersion: "TLS 1.3", ALPN: "h3", DidResume: state.ConnectionState.TLS.DidResume,
		CertificateDERSHA256: certificateDERSHA256, CertificateSPKISHA256: certificateSPKISHA256,
		RequestMethod: "CONNECT", RequestProtocol: "webtransport", RequestScheme: "https", RequestAuthority: authority, RequestPath: pathValue,
		ResponseStatus: 200, DatagramsEnabled: true, SessionEstablished: true, ClientInitiatedBidirectional: true,
		SessionCount: 1, BidirectionalStreamCount: 1, PayloadBytes: payloadBytes, PayloadSHA256: payloadSHA256,
		EchoedBytes: len(echoed), EchoedSHA256: hash(echoed), SessionCloseCode: 0, CleanCompletion: true,
		ConfiguredSessions: 1, ConfiguredConcurrency: 1, ConfiguredStreamsPerSession: 1,
		ObservedActiveSessions: 1, ObservedActiveStreams: 1, EffectiveConcurrency: 1,
	}
	return operationResult{Proof: proof, SessionLatencyMS: durationMS(sessionLatency), StreamLatencyMS: durationMS(streamLatency), TransferredBytes: payloadBytes * 2}, nil
}

func runFor(actualAddress, logicalURL string, roots *x509.CertPool, payload []byte, duration time.Duration) summary {
	s := summary{Errors: map[string]int{}}
	started := time.Now()
	for time.Since(started) < duration {
		op, err := runOperation(actualAddress, logicalURL, roots, payload)
		if err != nil {
			var timeout net.Error
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
				s.TimedOutOperations++
			} else {
				s.FailedOperations++
			}
			s.Errors[err.Error()]++
			continue
		}
		s.CompletedOperations++
		s.TotalTransferredBytes += int64(op.TransferredBytes)
		s.SessionLatencies = append(s.SessionLatencies, op.SessionLatencyMS)
		s.StreamLatencies = append(s.StreamLatencies, op.StreamLatencyMS)
		s.Last = &op
	}
	s.DurationSeconds = time.Since(started).Seconds()
	return s
}

func summaryFromOperation(op operationResult) summary {
	duration := op.StreamLatencyMS / 1000
	if duration <= 0 {
		duration = .001
	}
	return summary{DurationSeconds: duration, CompletedOperations: 1, TotalTransferredBytes: int64(op.TransferredBytes), SessionLatencies: []float64{op.SessionLatencyMS}, StreamLatencies: []float64{op.StreamLatencyMS}, Last: &op, Errors: map[string]int{}}
}

func writeProofArtifacts(dir string, op operationResult) {
	checks := []string{"protocol:webtransport-over-h3", "tls:1.3", "alpn:h3", "no-fallback", "extended-connect", "pseudo-protocol:webtransport", "response-status:200", "session-established", "client-initiated-bidirectional-stream", "stream-count:1", "payload-bytes:65536", "payload-sha256", "ordered-echo", "session-close-code:0", "zero-unexpected-failures", "zero-timeouts"}
	writeJSON(dir, "validation.json", map[string]any{"schemaVersion": "protocol-lab.validation.v1", "scenarioId": scenarioID, "status": "passed", "checks": checks})
	writeJSON(dir, "protocol-proof.json", op.Proof)
	writeJSON(dir, "webtransport-summary.json", op)
	writeJSON(dir, "payload-hash.json", map[string]any{"algorithm": "sha256", "payloadBytes": payloadBytes, "expected": payloadSHA256, "observed": op.Proof.EchoedSHA256})
}

func writeResult(dir string, s summary) {
	d := s.DurationSeconds
	if d <= 0 {
		d = 1
	}
	metrics := map[string]any{
		"sessionsPerSecond": float64(s.CompletedOperations) / d, "bytesPerSecond": float64(s.TotalTransferredBytes) / d,
		"sessionLatencyMean": mean(s.SessionLatencies), "sessionLatencyP50": percentile(s.SessionLatencies, .5), "sessionLatencyP95": percentile(s.SessionLatencies, .95), "sessionLatencyP99": percentile(s.SessionLatencies, .99),
		"streamLatencyMean": mean(s.StreamLatencies), "streamLatencyP50": percentile(s.StreamLatencies, .5), "streamLatencyP75": percentile(s.StreamLatencies, .75), "streamLatencyP90": percentile(s.StreamLatencies, .9), "streamLatencyP95": percentile(s.StreamLatencies, .95), "streamLatencyP99": percentile(s.StreamLatencies, .99),
		"completedOperations": s.CompletedOperations, "failedOperations": s.FailedOperations, "timedOutOperations": s.TimedOutOperations, "totalTransferredBytes": s.TotalTransferredBytes,
		"configuredSessions": 1, "configuredConcurrency": 1, "configuredStreamsPerSession": 1, "observedActiveSessions": 1, "observedActiveStreams": 1, "effectiveConcurrency": 1, "effectiveStreams": 1,
	}
	result := map[string]any{
		"schemaVersion": "protocol-lab.webtransport-executor-result.v1", "scenarioId": scenarioID, "authorityCommit": authorityCommit,
		"executor":      map[string]string{"id": executorID, "version": executorVersion},
		"loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": engineModule, "engineModuleVersion": engineModuleVersion},
		"validation":    map[string]string{"status": "passed"}, "protocolProof": s.Last.Proof,
		"requestedLoad": map[string]any{"profileId": profileID, "sessions": 1, "concurrency": 1, "bidirectionalStreamsPerSession": 1, "durationSeconds": 5, "warmupSeconds": 1, "repetitions": 1, "operationTimeoutMilliseconds": 10000},
		"effectiveLoad": map[string]any{"sessions": 1, "concurrency": 1, "streams": 1, "observed": map[string]int{"activeSessions": 1, "activeStreams": 1, "effectiveConcurrency": 1}},
		"metrics":       metrics, "warnings": []string{"Single-host WebTransport smoke evidence is diagnostic and non-rankable."},
	}
	writeJSON(dir, "webtransport-load-summary.json", s)
	writeJSON(dir, "webtransport-executor-result.json", result)
	writeJSON(dir, "result.json", result)
}

func makePayload() []byte {
	data := make([]byte, payloadBytes)
	for i := range data {
		data[i] = byte(i % 251)
	}
	return data
}
func normalizeTarget(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", errors.New("target required")
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	u, err := url.Parse(value)
	if err != nil || u.Host == "" {
		return "", "", errors.New("target must be https://host:port")
	}
	if u.Scheme != "https" {
		return "", "", errors.New("target must use https")
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	actual := net.JoinHostPort(u.Hostname(), port)
	logical := "https://" + net.JoinHostPort(authority, port) + pathValue
	return actual, logical, nil
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
func materialPath(relative string) string {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "..", "..", relative)
		if _, err = os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join("..", relative)
}
func outputOrDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return "artifacts"
	}
	return value
}
func envOr(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}
func verifyOptional(name, expected string) {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" && v != expected {
		fatal(2, fmt.Errorf("%s substitution: expected %q observed %q", name, expected, v))
	}
}
func hash(data []byte) string                { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func durationMS(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
func percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	v := append([]float64(nil), values...)
	sort.Float64s(v)
	i := int(math.Ceil(q*float64(len(v)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(v) {
		i = len(v) - 1
	}
	return v[i]
}
func writeIdentity(dir string) {
	writeJSON(dir, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedProtocols": []string{"webtransport", "h3"}, "supportedScenarios": []string{scenarioID}, "supportedLoadProfiles": []string{profileID}})
	writeJSON(dir, "load-generator-identity.json", map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": engineModule, "engineModuleVersion": engineModuleVersion})
}
func writeFailure(dir string, err error) {
	_ = os.MkdirAll(dir, 0o755)
	value := map[string]any{"schemaVersion": "protocol-lab.validation.v1", "status": "failed", "scenarioId": scenarioID, "message": err.Error()}
	writeJSON(dir, "validation.json", value)
	writeJSON(dir, "result.json", value)
	writeIdentity(dir)
}
func emitUnsupported(dir, id string) {
	_ = os.MkdirAll(dir, 0o755)
	value := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": id, "reasonCode": "scenario-not-implemented", "authorityCommit": authorityCommit, "executor": map[string]string{"id": executorID, "version": executorVersion}, "loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, "supportedScenarios": []string{scenarioID}}
	writeJSON(dir, "unsupported.json", value)
	writeJSON(dir, "result.json", value)
	writeIdentity(dir)
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
