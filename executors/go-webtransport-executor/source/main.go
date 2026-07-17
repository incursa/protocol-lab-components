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
	executorID              = "go-webtransport-executor"
	executorVersion         = "0.2.1"
	loadGeneratorID         = "webtransport-go-load"
	loadGeneratorVersion    = "0.2.1"
	engineModule            = "github.com/quic-go/webtransport-go"
	engineModuleVersion     = "v0.11.1"
	authorityCommit         = "dd518aee19d73fb1477320644785fa070b1b62f1"
	streamScenarioID        = "webtransport.session-bidi-echo"
	datagramScenarioID      = "webtransport.session-datagram-echo"
	streamProfileID         = "webtransport-smoke"
	datagramProfileID       = "webtransport-datagram-smoke"
	protocolVariant         = "webtransport-over-h3"
	authority               = "webtransport.plab.test"
	pathValue               = "/webtransport/echo"
	payloadBytes            = 65536
	payloadSHA256           = "4b640d85ab3ba30fd02c9fc9db4a8928f416322ad27022ea58a65aaee68a4df2"
	datagramCount           = 32
	payloadBytesPerDatagram = 256
	payloadSetSHA256        = "2e975a37b4bff0a8022c0f89ab19e9a8e2599300e557e9b8ce3eff364cd33e8b"
	certificateDERSHA256    = "de0f805e043ced1ca4742ad65ada6eb23d2338180f6129a3a72bfd367f932b9c"
	certificateSPKISHA256   = "431113d9234c136aad960a67d7a9788b7c95ae7fa53d482dd3b48a0bc5aa69cc"
	warmupDuration          = time.Second
	measurementDuration     = 5 * time.Second
	operationTimeout        = 10 * time.Second
)

type scenarioSpec struct {
	ID        string
	ProfileID string
	Datagram  bool
}

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
	DatagramCount                int    `json:"datagramCount,omitempty"`
	PayloadBytesPerDatagram      int    `json:"payloadBytesPerDatagram,omitempty"`
	PayloadGenerator             string `json:"payloadGenerator,omitempty"`
	PayloadSetSHA256             string `json:"payloadSetSha256,omitempty"`
	EchoedDatagramCount          int    `json:"echoedDatagramCount,omitempty"`
	EchoedPayloadSetSHA256       string `json:"echoedPayloadSetSha256,omitempty"`
	OrderedDatagramEcho          bool   `json:"orderedDatagramEcho,omitempty"`
	LostDatagrams                int    `json:"lostDatagrams"`
}

type operationResult struct {
	Proof               protocolProof `json:"protocolProof"`
	SessionLatencyMS    float64       `json:"sessionLatencyMilliseconds"`
	StreamLatencyMS     float64       `json:"streamLatencyMilliseconds"`
	DatagramLatenciesMS []float64     `json:"datagramLatencyMilliseconds,omitempty"`
	TransferredBytes    int           `json:"transferredBytes"`
}

type summary struct {
	DurationSeconds       float64          `json:"durationSeconds"`
	CompletedOperations   int              `json:"completedOperations"`
	FailedOperations      int              `json:"failedOperations"`
	TimedOutOperations    int              `json:"timedOutOperations"`
	TotalTransferredBytes int64            `json:"totalTransferredBytes"`
	SessionLatencies      []float64        `json:"sessionLatencyMilliseconds"`
	StreamLatencies       []float64        `json:"streamLatencyMilliseconds"`
	DatagramLatencies     []float64        `json:"datagramLatencyMilliseconds"`
	SentDatagrams         int              `json:"sentDatagrams"`
	ReceivedDatagrams     int              `json:"receivedDatagrams"`
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
	spec, err := selectedScenario()
	if err != nil {
		emitUnsupported(outputOrDefault(*output), strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")))
		os.Exit(3)
	}
	verifyOptional("PLAB_LOAD_PROFILE_ID", spec.ProfileID)
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
	payloads := makeDatagramPayloadSet()
	writeIdentity(*output)

	preflight, err := runOperation(actualAddress, logicalURL, roots, spec, payload, payloads)
	if err != nil {
		writeFailure(*output, err)
		fatal(1, err)
	}
	var warmup, measured summary
	if *validationOnly {
		measured = summaryFromOperation(preflight)
	} else {
		warmup = runFor(actualAddress, logicalURL, roots, spec, payload, payloads, warmupDuration)
		if warmup.FailedOperations != 0 || warmup.TimedOutOperations != 0 {
			err = errors.New("warmup contained failed or timed-out operations")
		}
		if err == nil {
			measured = runFor(actualAddress, logicalURL, roots, spec, payload, payloads, measurementDuration)
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
	writeProofArtifacts(*output, spec, preflight)
	writeResult(*output, spec, measured)
	data, _ := os.ReadFile(filepath.Join(*output, "result.json"))
	fmt.Print(string(data))
}

func selectedScenario() (scenarioSpec, error) {
	id := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if id == "" || id == streamScenarioID {
		return scenarioSpec{ID: streamScenarioID, ProfileID: streamProfileID}, nil
	}
	if id == datagramScenarioID {
		return scenarioSpec{ID: datagramScenarioID, ProfileID: datagramProfileID, Datagram: true}, nil
	}
	return scenarioSpec{}, fmt.Errorf("unsupported WebTransport scenario %q", id)
}

func runOperation(actualAddress, logicalURL string, roots *x509.CertPool, spec scenarioSpec, payload []byte, payloads [][]byte) (operationResult, error) {
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

	proof := protocolProof{
		Protocol: "webtransport", ProtocolVersion: "draft-ietf-webtrans-http3", ProtocolVariant: protocolVariant,
		TLSVersion: "TLS 1.3", ALPN: "h3", DidResume: state.ConnectionState.TLS.DidResume,
		CertificateDERSHA256: certificateDERSHA256, CertificateSPKISHA256: certificateSPKISHA256,
		RequestMethod: "CONNECT", RequestProtocol: "webtransport", RequestScheme: "https", RequestAuthority: authority, RequestPath: pathValue,
		ResponseStatus: 200, DatagramsEnabled: true, SessionEstablished: true,
		SessionCount: 1, SessionCloseCode: 0, CleanCompletion: true,
		ConfiguredSessions: 1, ConfiguredConcurrency: 1, ObservedActiveSessions: 1, EffectiveConcurrency: 1,
	}
	if !spec.Datagram {
		streamStarted := time.Now()
		stream, openErr := session.OpenStreamSync(ctx)
		if openErr != nil {
			return operationResult{}, openErr
		}
		_ = stream.SetDeadline(time.Now().Add(operationTimeout))
		if _, err = stream.Write(payload); err != nil {
			return operationResult{}, err
		}
		if err = stream.Close(); err != nil {
			return operationResult{}, err
		}
		echoed, readErr := io.ReadAll(io.LimitReader(stream, payloadBytes+1))
		if readErr != nil {
			return operationResult{}, readErr
		}
		streamLatency := time.Since(streamStarted)
		if len(echoed) != payloadBytes || !bytes.Equal(payload, echoed) || hash(echoed) != payloadSHA256 {
			return operationResult{}, errors.New("echoed payload identity mismatch")
		}
		proof.ClientInitiatedBidirectional = true
		proof.BidirectionalStreamCount = 1
		proof.PayloadBytes = payloadBytes
		proof.PayloadSHA256 = payloadSHA256
		proof.EchoedBytes = len(echoed)
		proof.EchoedSHA256 = hash(echoed)
		proof.ConfiguredStreamsPerSession = 1
		proof.ObservedActiveStreams = 1
		if err = session.CloseWithError(0, ""); err != nil {
			return operationResult{}, err
		}
		return operationResult{Proof: proof, SessionLatencyMS: durationMS(sessionLatency), StreamLatencyMS: durationMS(streamLatency), TransferredBytes: payloadBytes * 2}, nil
	}

	latencies := make([]float64, 0, len(payloads))
	receivedPayloads := make([][]byte, 0, len(payloads))
	for index, datagram := range payloads {
		started := time.Now()
		if err = session.SendDatagram(datagram); err != nil {
			return operationResult{}, fmt.Errorf("datagram %d write failed: %w", index, err)
		}
		echoed, receiveErr := session.ReceiveDatagram(ctx)
		latencies = append(latencies, durationMS(time.Since(started)))
		if receiveErr != nil {
			return operationResult{}, fmt.Errorf("datagram %d read failed: %w", index, receiveErr)
		}
		if !bytes.Equal(datagram, echoed) {
			return operationResult{}, fmt.Errorf("datagram %d exact echo mismatch", index)
		}
		receivedPayloads = append(receivedPayloads, echoed)
	}
	observedHash := hashPayloadSet(receivedPayloads)
	if observedHash != payloadSetSHA256 {
		return operationResult{}, errors.New("echoed datagram payload-set hash mismatch")
	}
	proof.DatagramCount = len(payloads)
	proof.PayloadBytesPerDatagram = payloadBytesPerDatagram
	proof.PayloadGenerator = "datagram-index-plus-octet-mod-251"
	proof.PayloadSetSHA256 = payloadSetSHA256
	proof.EchoedDatagramCount = len(receivedPayloads)
	proof.EchoedPayloadSetSHA256 = observedHash
	proof.OrderedDatagramEcho = true
	proof.LostDatagrams = 0
	if err = session.CloseWithError(0, ""); err != nil {
		return operationResult{}, err
	}
	return operationResult{Proof: proof, SessionLatencyMS: durationMS(sessionLatency), DatagramLatenciesMS: latencies, TransferredBytes: len(payloads) * payloadBytesPerDatagram * 2}, nil
}

func runFor(actualAddress, logicalURL string, roots *x509.CertPool, spec scenarioSpec, payload []byte, payloads [][]byte, duration time.Duration) summary {
	s := summary{Errors: map[string]int{}}
	started := time.Now()
	for time.Since(started) < duration {
		op, err := runOperation(actualAddress, logicalURL, roots, spec, payload, payloads)
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
		s.DatagramLatencies = append(s.DatagramLatencies, op.DatagramLatenciesMS...)
		if spec.Datagram {
			s.SentDatagrams += datagramCount
			s.ReceivedDatagrams += op.Proof.EchoedDatagramCount
		}
		s.Last = &op
	}
	s.DurationSeconds = time.Since(started).Seconds()
	return s
}

func summaryFromOperation(op operationResult) summary {
	duration := (op.StreamLatencyMS + sum(op.DatagramLatenciesMS)) / 1000
	if duration <= 0 {
		duration = .001
	}
	value := summary{DurationSeconds: duration, CompletedOperations: 1, TotalTransferredBytes: int64(op.TransferredBytes), SessionLatencies: []float64{op.SessionLatencyMS}, StreamLatencies: []float64{op.StreamLatencyMS}, DatagramLatencies: op.DatagramLatenciesMS, Last: &op, Errors: map[string]int{}}
	if op.Proof.DatagramCount > 0 {
		value.SentDatagrams = op.Proof.DatagramCount
		value.ReceivedDatagrams = op.Proof.EchoedDatagramCount
	}
	return value
}

func writeProofArtifacts(dir string, spec scenarioSpec, op operationResult) {
	checks := []string{"protocol:webtransport-over-h3", "tls:1.3", "alpn:h3", "no-fallback", "extended-connect", "pseudo-protocol:webtransport", "response-status:200", "session-established"}
	if spec.Datagram {
		checks = append(checks, "webtransport-datagrams-negotiated", "datagram-count:32", "datagram-bytes:256", "datagram-payload-set-sha256", "ordered-datagram-echo", "session-close-code:0", "zero-lost-datagrams", "zero-unexpected-failures", "zero-timeouts")
	} else {
		checks = append(checks, "client-initiated-bidirectional-stream", "stream-count:1", "payload-bytes:65536", "payload-sha256", "ordered-echo", "session-close-code:0", "zero-unexpected-failures", "zero-timeouts")
	}
	writeJSON(dir, "validation.json", map[string]any{"schemaVersion": "protocol-lab.validation.v1", "scenarioId": spec.ID, "status": "passed", "checks": checks})
	writeJSON(dir, "protocol-proof.json", op.Proof)
	writeJSON(dir, "webtransport-summary.json", op)
	if spec.Datagram {
		writeJSON(dir, "payload-hash.json", map[string]any{"algorithm": "sha256", "datagramCount": datagramCount, "payloadBytesPerDatagram": payloadBytesPerDatagram, "expected": payloadSetSHA256, "observed": op.Proof.EchoedPayloadSetSHA256})
	} else {
		writeJSON(dir, "payload-hash.json", map[string]any{"algorithm": "sha256", "payloadBytes": payloadBytes, "expected": payloadSHA256, "observed": op.Proof.EchoedSHA256})
	}
}

func writeResult(dir string, spec scenarioSpec, s summary) {
	metrics := metricsForSummary(spec, s)
	requestedLoad := map[string]any{"profileId": spec.ProfileID, "sessions": 1, "concurrency": 1, "durationSeconds": 5, "warmupSeconds": 1, "repetitions": 1, "operationTimeoutMilliseconds": 10000}
	effectiveLoad := map[string]any{"sessions": 1, "concurrency": 1, "observed": map[string]int{"activeSessions": 1, "effectiveConcurrency": 1}}
	if spec.Datagram {
		requestedLoad["datagramsPerSession"] = datagramCount
		effectiveLoad["datagramsPerSession"] = datagramCount
	} else {
		requestedLoad["bidirectionalStreamsPerSession"] = 1
		effectiveLoad["streams"] = 1
		effectiveLoad["observed"].(map[string]int)["activeStreams"] = 1
	}
	result := map[string]any{
		"schemaVersion": "protocol-lab.webtransport-executor-result.v1", "scenarioId": spec.ID, "authorityCommit": authorityCommit,
		"executor":      map[string]string{"id": executorID, "version": executorVersion},
		"loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": engineModule, "engineModuleVersion": engineModuleVersion},
		"validation":    map[string]string{"status": "passed"}, "protocolProof": s.Last.Proof,
		"requestedLoad": requestedLoad, "effectiveLoad": effectiveLoad,
		"metrics": metrics, "warnings": []string{"Single-host WebTransport smoke evidence is diagnostic and non-rankable."},
	}
	writeJSON(dir, "webtransport-load-summary.json", s)
	writeJSON(dir, "webtransport-executor-result.json", result)
	writeJSON(dir, "result.json", result)
}

func metricsForSummary(spec scenarioSpec, s summary) map[string]any {
	d := s.DurationSeconds
	if d <= 0 {
		d = 1
	}
	metrics := map[string]any{
		"sessionsPerSecond": float64(s.CompletedOperations) / d, "bytesPerSecond": float64(s.TotalTransferredBytes) / d,
		"sessionLatencyMean": mean(s.SessionLatencies), "sessionLatencyP50": percentile(s.SessionLatencies, .5), "sessionLatencyP95": percentile(s.SessionLatencies, .95), "sessionLatencyP99": percentile(s.SessionLatencies, .99),
		"completedOperations": s.CompletedOperations, "failedOperations": s.FailedOperations, "timedOutOperations": s.TimedOutOperations, "totalTransferredBytes": s.TotalTransferredBytes,
		"configuredSessions": 1, "configuredConcurrency": 1, "observedActiveSessions": 1, "effectiveConcurrency": 1, "effectiveSessions": 1,
	}
	if spec.Datagram {
		metrics["datagramsPerSecond"] = float64(s.ReceivedDatagrams) / d
		metrics["datagramLatencyMean"] = mean(s.DatagramLatencies)
		metrics["datagramLatencyP50"] = percentile(s.DatagramLatencies, .5)
		metrics["datagramLatencyP75"] = percentile(s.DatagramLatencies, .75)
		metrics["datagramLatencyP90"] = percentile(s.DatagramLatencies, .9)
		metrics["datagramLatencyP95"] = percentile(s.DatagramLatencies, .95)
		metrics["datagramLatencyP99"] = percentile(s.DatagramLatencies, .99)
		metrics["sentDatagrams"] = s.SentDatagrams
		metrics["receivedDatagrams"] = s.ReceivedDatagrams
		metrics["lostDatagrams"] = s.SentDatagrams - s.ReceivedDatagrams
		metrics["configuredDatagramsPerSession"] = datagramCount
	} else {
		metrics["streamLatencyMean"] = mean(s.StreamLatencies)
		metrics["streamLatencyP50"] = percentile(s.StreamLatencies, .5)
		metrics["streamLatencyP75"] = percentile(s.StreamLatencies, .75)
		metrics["streamLatencyP90"] = percentile(s.StreamLatencies, .9)
		metrics["streamLatencyP95"] = percentile(s.StreamLatencies, .95)
		metrics["streamLatencyP99"] = percentile(s.StreamLatencies, .99)
		metrics["configuredStreamsPerSession"] = 1
		metrics["observedActiveStreams"] = 1
		metrics["effectiveStreams"] = 1
	}
	return metrics
}

func makePayload() []byte {
	data := make([]byte, payloadBytes)
	for i := range data {
		data[i] = byte(i % 251)
	}
	return data
}

func makeDatagramPayloadSet() [][]byte {
	payloads := make([][]byte, datagramCount)
	for datagramIndex := range payloads {
		payload := make([]byte, payloadBytesPerDatagram)
		for octetIndex := range payload {
			payload[octetIndex] = byte((datagramIndex + octetIndex) % 251)
		}
		payloads[datagramIndex] = payload
	}
	return payloads
}

func hashPayloadSet(payloads [][]byte) string {
	hasher := sha256.New()
	for _, payload := range payloads {
		_, _ = hasher.Write(payload)
	}
	return hex.EncodeToString(hasher.Sum(nil))
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
func sum(values []float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}
	return total
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
	writeJSON(dir, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedProtocols": []string{"webtransport", "h3"}, "supportedScenarios": []string{streamScenarioID, datagramScenarioID}, "supportedLoadProfiles": []string{streamProfileID, datagramProfileID}})
	writeJSON(dir, "load-generator-identity.json", map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": engineModule, "engineModuleVersion": engineModuleVersion})
}
func writeFailure(dir string, err error) {
	_ = os.MkdirAll(dir, 0o755)
	scenario := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if scenario == "" {
		scenario = streamScenarioID
	}
	value := map[string]any{"schemaVersion": "protocol-lab.validation.v1", "status": "failed", "scenarioId": scenario, "message": err.Error()}
	writeJSON(dir, "validation.json", value)
	writeJSON(dir, "result.json", value)
	writeIdentity(dir)
}
func emitUnsupported(dir, id string) {
	_ = os.MkdirAll(dir, 0o755)
	value := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": id, "reasonCode": "scenario-not-implemented", "authorityCommit": authorityCommit, "executor": map[string]string{"id": executorID, "version": executorVersion}, "loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, "supportedScenarios": []string{streamScenarioID, datagramScenarioID}}
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
