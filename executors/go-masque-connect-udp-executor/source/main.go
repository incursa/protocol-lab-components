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
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	masque "github.com/quic-go/masque-go"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/yosida95/uritemplate/v3"
)

const (
	executorID              = "go-masque-connect-udp-executor"
	executorVersion         = "0.1.1"
	loadGeneratorID         = "masque-go-load"
	loadGeneratorVersion    = "0.1.0"
	engineModule            = "github.com/quic-go/masque-go"
	engineModuleVersion     = "v0.4.0"
	authorityCommit         = "d8082f5ae7d872e9e556f9696177079929483c58"
	scenarioID              = "masque.connect-udp-tunnel"
	profileID               = "masque-connect-udp-comparison"
	protocolVariant         = "masque-connect-udp-over-h3"
	proxyAuthority          = "masque-proxy.plab.test"
	targetAuthority         = "masque-echo.plab.test:4433"
	pathTemplate            = "/.well-known/masque/udp/{target_host}/{target_port}/"
	expandedPath            = "/.well-known/masque/udp/masque-echo.plab.test/4433/"
	datagramCount           = 32
	payloadBytesPerDatagram = 256
	payloadSetSHA256        = "2e975a37b4bff0a8022c0f89ab19e9a8e2599300e557e9b8ce3eff364cd33e8b"
	certificateDERSHA256    = "f23269776cd1dc06fe36e68ab130de491a427a8596e5683e8aa4019091d97a83"
	certificateSPKISHA256   = "02fb58a89ebf6e7d63fd86cb668e67c31dd832c317795e2c04de13d6c01b85f2"
	repetitionCount         = 3
	configuredTunnels       = 1
	configuredConcurrency   = 1
)

var (
	warmupDuration      = 3 * time.Second
	measurementDuration = 15 * time.Second
	cooldownDuration    = 3 * time.Second
	operationTimeout    = 15 * time.Second
)

type protocolProof struct {
	Protocol                 string `json:"protocol"`
	ProtocolVersion          string `json:"protocolVersion"`
	ProtocolVariant          string `json:"protocolVariant"`
	TLSVersion               string `json:"tlsVersion"`
	ALPN                     string `json:"alpn"`
	DidResume                bool   `json:"didResume"`
	CertificateDERSHA256     string `json:"certificateDerSha256"`
	CertificateSPKISHA256    string `json:"certificateSpkiSha256"`
	RequestMethod            string `json:"requestMethod"`
	RequestProtocol          string `json:"requestProtocol"`
	RequestScheme            string `json:"requestScheme"`
	RequestAuthority         string `json:"requestAuthority"`
	RequestPathTemplate      string `json:"requestPathTemplate"`
	RequestExpandedPath      string `json:"requestExpandedPath"`
	ResponseStatus           int    `json:"responseStatus"`
	CapsuleProtocol          string `json:"capsuleProtocol"`
	DatagramsEnabled         bool   `json:"datagramsEnabled"`
	ContextID                int    `json:"contextId"`
	TunnelEstablished        bool   `json:"tunnelEstablished"`
	ProxyRole                string `json:"proxyRole"`
	TargetRole               string `json:"targetRole"`
	RequestedTargetAuthority string `json:"requestedTargetAuthority"`
	ObservedRemoteAddress    string `json:"observedRemoteAddress"`
	TunnelCount              int    `json:"tunnelCount"`
	DatagramCount            int    `json:"datagramCount"`
	PayloadBytesPerDatagram  int    `json:"payloadBytesPerDatagram"`
	PayloadGenerator         string `json:"payloadGenerator"`
	PayloadSetSHA256         string `json:"payloadSetSha256"`
	EchoedDatagramCount      int    `json:"echoedDatagramCount"`
	EchoedPayloadSetSHA256   string `json:"echoedPayloadSetSha256"`
	OrderedEcho              bool   `json:"orderedEcho"`
	LostDatagrams            int    `json:"lostDatagrams"`
	CleanCompletion          bool   `json:"cleanCompletion"`
	ConfiguredTunnels        int    `json:"configuredTunnels"`
	ConfiguredConcurrency    int    `json:"configuredConcurrency"`
	ConfiguredDatagrams      int    `json:"configuredDatagramsPerTunnel"`
	ObservedActiveTunnels    int    `json:"observedActiveTunnels"`
	EffectiveConcurrency     int    `json:"effectiveConcurrency"`
}

type operationResult struct {
	Proof               protocolProof `json:"protocolProof"`
	TunnelLatencyMS     float64       `json:"tunnelLatencyMilliseconds"`
	DatagramLatenciesMS []float64     `json:"datagramLatencyMilliseconds"`
	TransferredBytes    int           `json:"transferredBytes"`
}

type summary struct {
	DurationSeconds       float64          `json:"durationSeconds"`
	CompletedOperations   int              `json:"completedOperations"`
	FailedOperations      int              `json:"failedOperations"`
	TimedOutOperations    int              `json:"timedOutOperations"`
	SentDatagrams         int              `json:"sentDatagrams"`
	ReceivedDatagrams     int              `json:"receivedDatagrams"`
	TotalTransferredBytes int64            `json:"totalTransferredBytes"`
	TunnelLatencies       []float64        `json:"tunnelLatencyMilliseconds"`
	DatagramLatencies     []float64        `json:"datagramLatencyMilliseconds"`
	Last                  *operationResult `json:"lastOperation,omitempty"`
	Errors                map[string]int   `json:"errors"`
}

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "target proxy endpoint")
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
	verifyOptional("PLAB_PROTOCOL", "masque")
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
	actualAddress, logicalTemplate, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	roots, err := loadRoots(*rootPath)
	if err != nil {
		fatal(2, err)
	}
	payloads := makePayloadSet()
	writeIdentity(*output)

	preflight, err := runOperation(actualAddress, logicalTemplate, roots, payloads)
	if err != nil {
		writeFailure(*output, err)
		fatal(1, err)
	}
	if *validationOnly {
		measured := summaryFromOperation(preflight)
		writeProofArtifacts(*output, preflight)
		writeResult(*output, []summary{measured}, measured)
		printResult(*output)
		return
	}

	warmup := runFor(actualAddress, logicalTemplate, roots, payloads, warmupDuration)
	writeJSON(*output, "masque-warmup-summary.json", warmup)
	if warmup.CompletedOperations == 0 || warmup.FailedOperations != 0 || warmup.TimedOutOperations != 0 {
		err = errors.New("warmup did not complete cleanly")
	}
	repetitions := make([]summary, 0, repetitionCount)
	combined := newSummary()
	for repetition := 0; err == nil && repetition < repetitionCount; repetition++ {
		measured := runFor(actualAddress, logicalTemplate, roots, payloads, measurementDuration)
		if measured.CompletedOperations == 0 || measured.FailedOperations != 0 || measured.TimedOutOperations != 0 || measured.SentDatagrams != measured.ReceivedDatagrams {
			err = fmt.Errorf("repetition %d did not complete cleanly", repetition+1)
			break
		}
		repetitions = append(repetitions, measured)
		mergeSummary(&combined, measured)
		if repetition+1 < repetitionCount {
			time.Sleep(cooldownDuration)
		}
	}
	if err != nil {
		writeFailure(*output, err)
		fatal(1, err)
	}
	writeProofArtifacts(*output, preflight)
	writeJSON(*output, "masque-repetitions.json", repetitions)
	writeResult(*output, repetitions, combined)
	printResult(*output)
}

func runOperation(actualAddress, logicalTemplate string, roots *x509.CertPool, payloads [][]byte) (operationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	template, err := uritemplate.New(logicalTemplate)
	if err != nil {
		return operationResult{}, err
	}
	request, err := masque.NewRequest(ctx, template, targetAuthority)
	if err != nil {
		return operationResult{}, err
	}
	request.Header().Set("X-ProtocolLab-Scenario", scenarioID)
	tlsConfig := &tls.Config{RootCAs: roots, ServerName: proxyAuthority, MinVersion: tls.VersionTLS13, NextProtos: []string{http3.NextProtoH3}}
	var quicConnection *quic.Conn
	transport := &masque.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig:      &quic.Config{EnableDatagrams: true},
		DialAddr: func(ctx context.Context, _ string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
			conn, dialErr := quic.DialAddr(ctx, actualAddress, tlsCfg, cfg)
			if dialErr == nil {
				quicConnection = conn
			}
			return conn, dialErr
		},
	}
	tunnelStarted := time.Now()
	proxiedConn, response, err := transport.Dial(request)
	if err != nil {
		return operationResult{}, err
	}
	defer proxiedConn.Close()
	tunnelLatency := time.Since(tunnelStarted)
	if response.StatusCode != httpStatusOK {
		return operationResult{}, fmt.Errorf("unexpected CONNECT-UDP status %d", response.StatusCode)
	}
	if quicConnection == nil {
		return operationResult{}, errors.New("QUIC connection proof unavailable")
	}
	state := quicConnection.ConnectionState()
	if state.TLS.Version != tls.VersionTLS13 || state.TLS.NegotiatedProtocol != http3.NextProtoH3 {
		return operationResult{}, fmt.Errorf("protocol substitution: tls=%x alpn=%q", state.TLS.Version, state.TLS.NegotiatedProtocol)
	}
	if !state.SupportsDatagrams.Local || !state.SupportsDatagrams.Remote {
		return operationResult{}, errors.New("HTTP/3 datagram support not negotiated")
	}
	if len(state.TLS.PeerCertificates) != 1 {
		return operationResult{}, fmt.Errorf("expected one peer certificate, observed %d", len(state.TLS.PeerCertificates))
	}
	leaf := state.TLS.PeerCertificates[0]
	if hash(leaf.Raw) != certificateDERSHA256 || hash(leaf.RawSubjectPublicKeyInfo) != certificateSPKISHA256 {
		return operationResult{}, errors.New("certificate identity mismatch")
	}
	proxyRole := response.Header.Get("X-ProtocolLab-Proxy-Role")
	targetRole := response.Header.Get("X-ProtocolLab-Target-Role")
	if proxyRole == "" || targetRole != targetAuthority {
		return operationResult{}, fmt.Errorf("role proof mismatch: proxy=%q target=%q", proxyRole, targetRole)
	}

	latencies := make([]float64, 0, len(payloads))
	receivedPayloads := make([][]byte, 0, len(payloads))
	buffer := make([]byte, 1500)
	for index, payload := range payloads {
		if err = proxiedConn.SetDeadline(time.Now().Add(operationTimeout)); err != nil {
			return operationResult{}, err
		}
		started := time.Now()
		if _, err = proxiedConn.WriteTo(payload, nil); err != nil {
			return operationResult{}, fmt.Errorf("datagram %d write failed: %w", index, err)
		}
		n, _, readErr := proxiedConn.ReadFrom(buffer)
		latencies = append(latencies, durationMS(time.Since(started)))
		if readErr != nil {
			return operationResult{}, fmt.Errorf("datagram %d read failed: %w", index, readErr)
		}
		if n != len(payload) || !bytes.Equal(payload, buffer[:n]) {
			return operationResult{}, fmt.Errorf("datagram %d exact echo mismatch", index)
		}
		receivedPayloads = append(receivedPayloads, append([]byte(nil), buffer[:n]...))
	}
	observedHash := hashPayloadSet(receivedPayloads)
	if observedHash != payloadSetSHA256 {
		return operationResult{}, errors.New("echoed datagram payload-set hash mismatch")
	}
	proof := protocolProof{
		Protocol: "masque", ProtocolVersion: "RFC 9298", ProtocolVariant: protocolVariant,
		TLSVersion: "TLS 1.3", ALPN: http3.NextProtoH3, DidResume: state.TLS.DidResume,
		CertificateDERSHA256: certificateDERSHA256, CertificateSPKISHA256: certificateSPKISHA256,
		RequestMethod: "CONNECT", RequestProtocol: "connect-udp", RequestScheme: "https", RequestAuthority: proxyAuthority,
		RequestPathTemplate: pathTemplate, RequestExpandedPath: expandedPath, ResponseStatus: response.StatusCode,
		CapsuleProtocol: response.Header.Get(http3.CapsuleProtocolHeader), DatagramsEnabled: true, ContextID: 0,
		TunnelEstablished: true, ProxyRole: proxyRole, TargetRole: targetRole, RequestedTargetAuthority: targetAuthority,
		ObservedRemoteAddress: proxiedConn.RemoteAddr().String(), TunnelCount: 1, DatagramCount: len(payloads),
		PayloadBytesPerDatagram: payloadBytesPerDatagram, PayloadGenerator: "datagram-index-plus-octet-mod-251",
		PayloadSetSHA256: payloadSetSHA256, EchoedDatagramCount: len(receivedPayloads), EchoedPayloadSetSHA256: observedHash,
		OrderedEcho: true, LostDatagrams: 0, CleanCompletion: true,
		ConfiguredTunnels: configuredTunnels, ConfiguredConcurrency: configuredConcurrency, ConfiguredDatagrams: datagramCount,
		ObservedActiveTunnels: 1, EffectiveConcurrency: 1,
	}
	return operationResult{Proof: proof, TunnelLatencyMS: durationMS(tunnelLatency), DatagramLatenciesMS: latencies, TransferredBytes: len(payloads) * payloadBytesPerDatagram * 2}, nil
}

const httpStatusOK = 200

func runFor(actualAddress, logicalTemplate string, roots *x509.CertPool, payloads [][]byte, duration time.Duration) summary {
	result := newSummary()
	started := time.Now()
	for time.Since(started) < duration {
		op, err := runOperation(actualAddress, logicalTemplate, roots, payloads)
		if err != nil {
			var timeout net.Error
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
				result.TimedOutOperations++
			} else {
				result.FailedOperations++
			}
			result.Errors[err.Error()]++
			continue
		}
		result.CompletedOperations++
		result.SentDatagrams += datagramCount
		result.ReceivedDatagrams += op.Proof.EchoedDatagramCount
		result.TotalTransferredBytes += int64(op.TransferredBytes)
		result.TunnelLatencies = append(result.TunnelLatencies, op.TunnelLatencyMS)
		result.DatagramLatencies = append(result.DatagramLatencies, op.DatagramLatenciesMS...)
		result.Last = &op
	}
	result.DurationSeconds = time.Since(started).Seconds()
	return result
}

func newSummary() summary { return summary{Errors: map[string]int{}} }

func summaryFromOperation(op operationResult) summary {
	duration := (op.TunnelLatencyMS + sum(op.DatagramLatenciesMS)) / 1000
	if duration <= 0 {
		duration = .001
	}
	return summary{DurationSeconds: duration, CompletedOperations: 1, SentDatagrams: datagramCount, ReceivedDatagrams: datagramCount, TotalTransferredBytes: int64(op.TransferredBytes), TunnelLatencies: []float64{op.TunnelLatencyMS}, DatagramLatencies: op.DatagramLatenciesMS, Last: &op, Errors: map[string]int{}}
}

func mergeSummary(target *summary, source summary) {
	target.DurationSeconds += source.DurationSeconds
	target.CompletedOperations += source.CompletedOperations
	target.FailedOperations += source.FailedOperations
	target.TimedOutOperations += source.TimedOutOperations
	target.SentDatagrams += source.SentDatagrams
	target.ReceivedDatagrams += source.ReceivedDatagrams
	target.TotalTransferredBytes += source.TotalTransferredBytes
	target.TunnelLatencies = append(target.TunnelLatencies, source.TunnelLatencies...)
	target.DatagramLatencies = append(target.DatagramLatencies, source.DatagramLatencies...)
	if source.Last != nil {
		target.Last = source.Last
	}
	for message, count := range source.Errors {
		target.Errors[message] += count
	}
}

func writeProofArtifacts(dir string, op operationResult) {
	checks := []string{"protocol:masque-connect-udp-over-h3", "tls:1.3", "alpn:h3", "no-fallback", "extended-connect", "pseudo-protocol:connect-udp", "response-status:200", "http-datagrams-enabled", "context-id:0", "explicit-proxy-role", "explicit-udp-target-role", "tunnel-established", "datagram-count:32", "payload-bytes-per-datagram:256", "payload-set-sha256", "ordered-echo", "zero-datagram-loss", "zero-unexpected-failures", "zero-timeouts"}
	writeJSON(dir, "validation.json", map[string]any{"schemaVersion": "protocol-lab.validation.v1", "scenarioId": scenarioID, "status": "passed", "checks": checks})
	writeJSON(dir, "protocol-proof.json", op.Proof)
	writeJSON(dir, "masque-summary.json", op)
	writeJSON(dir, "datagram-hash.json", map[string]any{"algorithm": "sha256", "datagramCount": datagramCount, "payloadBytesPerDatagram": payloadBytesPerDatagram, "expected": payloadSetSHA256, "observed": op.Proof.EchoedPayloadSetSHA256})
}

func writeResult(dir string, repetitions []summary, combined summary) {
	metrics := metricsForSummary(combined)
	repetitionResults := make([]map[string]any, 0, len(repetitions))
	for index, repetition := range repetitions {
		repetitionResults = append(repetitionResults, map[string]any{"index": index + 1, "status": "passed", "summary": repetition, "metrics": metricsForSummary(repetition)})
	}
	result := map[string]any{
		"schemaVersion": "protocol-lab.masque-executor-result.v1", "scenarioId": scenarioID, "authorityCommit": authorityCommit,
		"executor":      map[string]string{"id": executorID, "version": executorVersion},
		"loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": engineModule, "engineModuleVersion": engineModuleVersion},
		"validation":    map[string]string{"status": "passed"}, "protocolProof": combined.Last.Proof,
		"requestedLoad": map[string]any{"profileId": profileID, "tunnels": configuredTunnels, "concurrency": configuredConcurrency, "datagramsPerTunnel": datagramCount, "durationSeconds": 15, "warmupSeconds": 3, "cooldownSeconds": 3, "repetitions": repetitionCount, "operationTimeoutMilliseconds": 15000},
		"effectiveLoad": map[string]any{"tunnels": 1, "concurrency": 1, "datagramsPerTunnel": datagramCount, "observed": map[string]int{"activeTunnels": 1, "effectiveConcurrency": 1}},
		"repetitions":   repetitionResults, "metrics": metrics,
		"warnings": []string{"Proxy and UDP target roles are separately proven but co-located in one target container; measurements are real comparable observations and non-rankable."},
	}
	writeJSON(dir, "masque-load-summary.json", combined)
	writeJSON(dir, "masque-executor-result.json", result)
	writeJSON(dir, "result.json", result)
}

func metricsForSummary(value summary) map[string]any {
	duration := value.DurationSeconds
	if duration <= 0 {
		duration = 1
	}
	return map[string]any{
		"tunnelsPerSecond":   float64(value.CompletedOperations) / duration,
		"datagramsPerSecond": float64(value.ReceivedDatagrams) / duration,
		"bytesPerSecond":     float64(value.TotalTransferredBytes) / duration,
		"tunnelLatencyMean":  mean(value.TunnelLatencies), "tunnelLatencyP50": percentile(value.TunnelLatencies, .5), "tunnelLatencyP95": percentile(value.TunnelLatencies, .95), "tunnelLatencyP99": percentile(value.TunnelLatencies, .99),
		"datagramLatencyMean": mean(value.DatagramLatencies), "datagramLatencyP50": percentile(value.DatagramLatencies, .5), "datagramLatencyP75": percentile(value.DatagramLatencies, .75), "datagramLatencyP90": percentile(value.DatagramLatencies, .9), "datagramLatencyP95": percentile(value.DatagramLatencies, .95), "datagramLatencyP99": percentile(value.DatagramLatencies, .99),
		"completedOperations": value.CompletedOperations, "failedOperations": value.FailedOperations, "timedOutOperations": value.TimedOutOperations,
		"sentDatagrams": value.SentDatagrams, "receivedDatagrams": value.ReceivedDatagrams, "lostDatagrams": value.SentDatagrams - value.ReceivedDatagrams,
		"totalTransferredBytes": value.TotalTransferredBytes, "configuredTunnels": 1, "configuredConcurrency": 1, "configuredDatagramsPerTunnel": datagramCount,
		"observedActiveTunnels": 1, "effectiveConcurrency": 1, "effectiveTunnels": 1,
	}
}

func makePayloadSet() [][]byte {
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
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", "", errors.New("target must be https://host:port")
	}
	if parsed.Scheme != "https" {
		return "", "", errors.New("target must use https")
	}
	port := parsed.Port()
	if port == "" {
		port = "443"
	}
	actual := net.JoinHostPort(parsed.Hostname(), port)
	logical := "https://" + net.JoinHostPort(proxyAuthority, port) + pathTemplate
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
	if executable, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(executable), "..", "..", relative)
		if _, err = os.Stat(candidate); err == nil {
			return candidate
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
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func verifyOptional(name, expected string) {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" && value != expected {
		fatal(2, fmt.Errorf("%s substitution: expected %q observed %q", name, expected, value))
	}
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func durationMS(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }

func sum(values []float64) float64 {
	var result float64
	for _, value := range values {
		result += value
	}
	return result
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return sum(values) / float64(len(values))
}

func percentile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	index := int(math.Ceil(quantile*float64(len(ordered)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(ordered) {
		index = len(ordered) - 1
	}
	return ordered[index]
}

func writeIdentity(dir string) {
	writeJSON(dir, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedProtocols": []string{"masque", "h3", "udp"}, "supportedScenarios": []string{scenarioID}, "supportedLoadProfiles": []string{profileID}})
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

func printResult(dir string) {
	data, _ := os.ReadFile(filepath.Join(dir, "result.json"))
	fmt.Print(string(data))
}

func fatal(code int, err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(code)
}
