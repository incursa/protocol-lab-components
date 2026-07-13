package main

import (
	"context"
	"crypto/ecdsa"
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
	"strconv"
	"strings"
	"time"
)

const (
	executorID           = "go-tls13-executor"
	executorVersion      = "0.1.0"
	loadGeneratorID      = "go-crypto-tls13-handshake-load"
	loadGeneratorVersion = "0.1.0"
	scenarioID           = "tls.handshake.full"
	loadProfileID        = "tls-smoke"
	serverName           = "tls.plab.test"
	alpn                 = "protocol-lab-tls"
	requiredCipherSuite  = "TLS_AES_128_GCM_SHA256"
	certificateProfile   = "plab-single-leaf-p256-v1"
	leafDERHash          = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
	leafSPKIHash         = "407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0"
)

type loadConfig struct {
	Connections             int
	Concurrency             int
	HandshakesPerConnection int
	ApplicationDataBytes    int
	Duration                time.Duration
	Warmup                  time.Duration
	Repetition              int
	ConnectionTimeout       time.Duration
}

type handshakeObservation struct {
	TLSVersion                      string  `json:"tlsVersion"`
	CipherSuite                     string  `json:"cipherSuite"`
	KeyExchangeGroup                string  `json:"keyExchangeGroup"`
	ALPN                            string  `json:"alpn"`
	ServerName                      string  `json:"serverName"`
	HandshakeComplete               bool    `json:"handshakeComplete"`
	DidResume                       bool    `json:"didResume"`
	EarlyDataAttempted              bool    `json:"earlyDataAttempted"`
	ApplicationDataBytesSent        int     `json:"applicationDataBytesSent"`
	ApplicationDataBytesReceived    int     `json:"applicationDataBytesReceived"`
	CertificateProfile              string  `json:"certificateProfile"`
	CertificateDERSHA256            string  `json:"certificateDerSha256"`
	CertificateSPKISHA256           string  `json:"certificateSpkiSha256"`
	CertificateSignatureAlgorithm   string  `json:"certificateSignatureAlgorithm"`
	CertificatePublicKeyAlgorithm   string  `json:"certificatePublicKeyAlgorithm"`
	CertificateNamedCurve           string  `json:"certificateNamedCurve"`
	VerifiedChainCount              int     `json:"verifiedChainCount"`
	TLSHandshakeLatencyMS           float64 `json:"tlsHandshakeLatencyMs"`
	ConnectionAndHandshakeLatencyMS float64 `json:"connectionAndHandshakeLatencyMs"`
}

type phaseSummary struct {
	Phase                              string                `json:"phase"`
	DurationSeconds                    float64               `json:"durationSeconds"`
	CompletedOperations                int                   `json:"completedOperations"`
	FailedOperations                   int                   `json:"failedOperations"`
	TimedOutOperations                 int                   `json:"timedOutOperations"`
	MaximumEffectiveConcurrency        int                   `json:"maximumEffectiveConcurrency"`
	TLSHandshakeLatencyMilliseconds    []float64             `json:"tlsHandshakeLatencyMilliseconds"`
	ConnectionAndHandshakeMilliseconds []float64             `json:"connectionAndHandshakeLatencyMilliseconds"`
	LastNegotiation                    *handshakeObservation `json:"lastNegotiation,omitempty"`
	Errors                             map[string]int        `json:"errors"`
}

type loadShape struct {
	Connections             int     `json:"connections"`
	Concurrency             int     `json:"concurrency"`
	HandshakesPerConnection int     `json:"handshakesPerConnection"`
	ApplicationDataBytes    int     `json:"applicationDataBytes"`
	DurationSeconds         float64 `json:"durationSeconds"`
	WarmupSeconds           float64 `json:"warmupSeconds"`
	Repetition              int     `json:"repetition"`
}

type latencyMetrics struct {
	HandshakesPerSecond                 float64 `json:"handshakesPerSecond"`
	TLSHandshakeLatencyMeanMS           float64 `json:"tlsHandshakeLatencyMeanMs"`
	TLSHandshakeLatencyP50MS            float64 `json:"tlsHandshakeLatencyP50Ms"`
	TLSHandshakeLatencyP75MS            float64 `json:"tlsHandshakeLatencyP75Ms"`
	TLSHandshakeLatencyP90MS            float64 `json:"tlsHandshakeLatencyP90Ms"`
	TLSHandshakeLatencyP95MS            float64 `json:"tlsHandshakeLatencyP95Ms"`
	TLSHandshakeLatencyP99MS            float64 `json:"tlsHandshakeLatencyP99Ms"`
	ConnectionAndHandshakeLatencyMeanMS float64 `json:"connectionAndHandshakeLatencyMeanMs"`
	CompletedOperations                 int     `json:"completedOperations"`
	FailedOperations                    int     `json:"failedOperations"`
	TimedOutOperations                  int     `json:"timedOutOperations"`
}

type executorResult struct {
	SchemaVersion string               `json:"schemaVersion"`
	Executor      map[string]string    `json:"executor"`
	LoadGenerator map[string]string    `json:"loadGenerator"`
	Validation    map[string]string    `json:"validation"`
	ProtocolProof handshakeObservation `json:"protocolProof"`
	RequestedLoad loadShape            `json:"requestedLoad"`
	EffectiveLoad loadShape            `json:"effectiveLoad"`
	Metrics       latencyMetrics       `json:"metrics"`
	Warnings      []string             `json:"warnings"`
}

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "TLS target address or tls:// URL.")
	outputDir := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "Artifact output directory.")
	rootCertificate := flag.String("root-certificate", os.Getenv("PLAB_TLS_ROOT_CERTIFICATE_PATH"), "Public test root PEM.")
	validationOnly := flag.Bool("validation-only", false, "Run one validity handshake and stop.")
	showVersion := flag.Bool("version", false, "Print executor version and exit.")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s %s\n", executorID, executorVersion)
		return
	}

	verifySubstitution("PLAB_EXECUTOR_ID", executorID, "executor")
	verifySubstitution("PLAB_EXECUTOR_VERSION", executorVersion, "executor version")
	verifySubstitution("PLAB_PROTOCOL", "tls", "protocol")
	verifySubstitution("PLAB_PROTOCOL_VARIANT", "tls1.3-full", "protocol variant")

	address, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	if strings.TrimSpace(*outputDir) == "" {
		*outputDir = "artifacts"
	}
	if strings.TrimSpace(*rootCertificate) == "" {
		*rootCertificate = filepath.Join("certs", "root.pem")
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fatal(1, err)
	}
	roots, err := loadRoots(*rootCertificate)
	if err != nil {
		fatal(2, err)
	}

	preflight, err := runHandshake(context.Background(), address, roots, 5*time.Second)
	validation := map[string]any{
		"scenarioId":             scenarioID,
		"passed":                 err == nil,
		"requestedProtocol":      "tls1.3",
		"observedProtocol":       preflight.TLSVersion,
		"fallbackDetected":       preflight.TLSVersion != "TLS1.3",
		"unexpectedFailureCount": boolInt(err != nil),
		"timeoutCount":           boolInt(isTimeout(err)),
		"error":                  errorString(err),
	}
	writeRequired(*outputDir, "validation.json", validation)
	writeRequired(*outputDir, "result.json", validation)
	writeRequired(*outputDir, "protocol-proof.json", preflight)
	writeRequired(*outputDir, "tls-negotiation.json", preflight)
	writeRequired(*outputDir, "executor-identity.json", map[string]any{
		"id": executorID, "version": executorVersion, "role": "client-test-executor",
		"supportedProtocols": []string{"tls"}, "supportedScenarios": []string{scenarioID},
		"supportedLoadProfiles": []string{loadProfileID},
	})
	if err != nil {
		fatal(1, fmt.Errorf("TLS validity handshake failed: %w", err))
	}
	if *validationOnly {
		fmt.Fprintln(os.Stderr, "go-tls13-executor validation passed with exact TLS 1.3 full-handshake proof")
		return
	}

	config, err := loadConfigFromEnvironment()
	if err != nil {
		fatal(2, err)
	}
	if config.Warmup > 0 {
		warmup := runPhase(address, roots, config.Warmup, config.ConnectionTimeout, "warmup")
		writeRequired(*outputDir, "tls-warmup-summary.json", warmup)
		if warmup.FailedOperations != 0 || warmup.TimedOutOperations != 0 || warmup.CompletedOperations == 0 {
			fatal(1, errors.New("TLS warmup did not satisfy the minimal validity gate"))
		}
	}
	measured := runPhase(address, roots, config.Duration, config.ConnectionTimeout, "measured")
	writeRequired(*outputDir, "tls-load-summary.json", measured)
	if measured.FailedOperations != 0 || measured.TimedOutOperations != 0 || measured.CompletedOperations == 0 || measured.LastNegotiation == nil {
		fatal(1, fmt.Errorf("TLS measured phase rejected: completed=%d failed=%d timedOut=%d", measured.CompletedOperations, measured.FailedOperations, measured.TimedOutOperations))
	}
	shape := loadShape{
		Connections: config.Connections, Concurrency: config.Concurrency,
		HandshakesPerConnection: config.HandshakesPerConnection, ApplicationDataBytes: config.ApplicationDataBytes,
		DurationSeconds: config.Duration.Seconds(), WarmupSeconds: config.Warmup.Seconds(), Repetition: config.Repetition,
	}
	effective := shape
	effective.Concurrency = measured.MaximumEffectiveConcurrency
	metrics := normalizeMetrics(measured)
	result := executorResult{
		SchemaVersion: "protocol-lab.tls-executor-result.v1",
		Executor:      map[string]string{"id": executorID, "version": executorVersion},
		LoadGenerator: map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion},
		Validation:    map[string]string{"status": "passed"},
		ProtocolProof: *measured.LastNegotiation,
		RequestedLoad: shape, EffectiveLoad: effective, Metrics: metrics,
		Warnings: []string{"TLS handshake smoke evidence is local and non-publishable; resumed handshakes, 0-RTT, record throughput, comparison, and ranking are unsupported."},
	}
	writeRequired(*outputDir, "tls-topology.json", map[string]any{"schemaVersion": "protocol-lab.tls-topology.v1", "requested": shape, "effective": effective})
	writeRequired(*outputDir, "connection-and-handshake-latency.json", map[string]any{"samplesMilliseconds": measured.ConnectionAndHandshakeMilliseconds})
	writeRequired(*outputDir, "load-generator-identity.json", result.LoadGenerator)
	writeRequired(*outputDir, "tls-executor-result.json", result)
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func loadConfigFromEnvironment() (loadConfig, error) {
	if strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")) != scenarioID {
		return loadConfig{}, fmt.Errorf("go-tls13-executor supports scenario %q only", scenarioID)
	}
	if strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID")) != loadProfileID {
		return loadConfig{}, fmt.Errorf("go-tls13-executor supports load profile %q only", loadProfileID)
	}
	config := loadConfig{
		Connections: envInt("PLAB_CONNECTIONS"), Concurrency: envInt("PLAB_CONCURRENCY"),
		HandshakesPerConnection: 1,
		ApplicationDataBytes:    0,
		Duration:                time.Duration(envInt("PLAB_DURATION_SECONDS")) * time.Second,
		Warmup:                  time.Duration(envInt("PLAB_WARMUP_SECONDS")) * time.Second,
		Repetition:              envInt("PLAB_REPETITION"),
		ConnectionTimeout:       operationTimeout(),
	}
	if config.Connections != 1 || config.Concurrency != 1 || config.HandshakesPerConnection != 1 || config.ApplicationDataBytes != 0 ||
		config.Duration != 5*time.Second || config.Warmup != time.Second || config.Repetition != 1 || config.ConnectionTimeout != 5*time.Second {
		return config, fmt.Errorf("tls-smoke with tls.handshake.full requires connections=1 concurrency=1 fresh handshakes, zero application bytes, duration=5s warmup=1s repetition=1 operationTimeout=5s; observed %+v", config)
	}
	return config, nil
}

func runPhase(address string, roots *x509.CertPool, duration, timeout time.Duration, phase string) phaseSummary {
	started := time.Now()
	summary := phaseSummary{Phase: phase, MaximumEffectiveConcurrency: 1, Errors: map[string]int{}}
	deadline := started.Add(duration)
	for time.Now().Before(deadline) {
		observation, err := runHandshake(context.Background(), address, roots, timeout)
		if err != nil {
			if isTimeout(err) {
				summary.TimedOutOperations++
			} else {
				summary.FailedOperations++
			}
			summary.Errors[errorString(err)]++
			continue
		}
		summary.CompletedOperations++
		summary.TLSHandshakeLatencyMilliseconds = append(summary.TLSHandshakeLatencyMilliseconds, observation.TLSHandshakeLatencyMS)
		summary.ConnectionAndHandshakeMilliseconds = append(summary.ConnectionAndHandshakeMilliseconds, observation.ConnectionAndHandshakeLatencyMS)
		summary.LastNegotiation = &observation
	}
	summary.DurationSeconds = time.Since(started).Seconds()
	return summary
}

func runHandshake(ctx context.Context, address string, roots *x509.CertPool, timeout time.Duration) (handshakeObservation, error) {
	connectionStart := time.Now()
	operationContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	raw, err := (&net.Dialer{}).DialContext(operationContext, "tcp", address)
	if err != nil {
		return handshakeObservation{}, err
	}
	defer raw.Close()
	client := tls.Client(raw, &tls.Config{
		MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13,
		RootCAs: roots, ServerName: serverName, NextProtos: []string{alpn},
		ClientSessionCache: nil,
	})
	handshakeStart := time.Now()
	if err := client.HandshakeContext(operationContext); err != nil {
		return handshakeObservation{}, err
	}
	handshakeFinished := time.Now()
	state := client.ConnectionState()
	observation, err := validateState(state)
	observation.TLSHandshakeLatencyMS = durationMS(handshakeFinished.Sub(handshakeStart))
	observation.ConnectionAndHandshakeLatencyMS = durationMS(handshakeFinished.Sub(connectionStart))
	_ = client.Close()
	return observation, err
}

func validateState(state tls.ConnectionState) (handshakeObservation, error) {
	observation := handshakeObservation{
		TLSVersion: tlsVersionName(state.Version), CipherSuite: tls.CipherSuiteName(state.CipherSuite),
		KeyExchangeGroup: state.CurveID.String(), ALPN: state.NegotiatedProtocol, ServerName: serverName,
		HandshakeComplete: state.HandshakeComplete, DidResume: state.DidResume,
		EarlyDataAttempted: false, ApplicationDataBytesSent: 0, ApplicationDataBytesReceived: 0,
		CertificateProfile: certificateProfile, VerifiedChainCount: len(state.VerifiedChains),
	}
	if len(state.PeerCertificates) > 0 {
		certificate := state.PeerCertificates[0]
		observation.CertificateDERSHA256 = hash(certificate.Raw)
		observation.CertificateSPKISHA256 = hash(certificate.RawSubjectPublicKeyInfo)
		observation.CertificateSignatureAlgorithm = certificate.SignatureAlgorithm.String()
		observation.CertificatePublicKeyAlgorithm = certificate.PublicKeyAlgorithm.String()
		if key, ok := certificate.PublicKey.(*ecdsa.PublicKey); ok {
			observation.CertificateNamedCurve = key.Curve.Params().Name
		}
	}
	var failures []string
	if state.Version != tls.VersionTLS13 {
		failures = append(failures, "exact TLS 1.3 was not negotiated")
	}
	if !state.HandshakeComplete {
		failures = append(failures, "handshake did not complete")
	}
	if state.DidResume {
		failures = append(failures, "session resumption was detected")
	}
	if state.NegotiatedProtocol != alpn {
		failures = append(failures, "ALPN mismatch")
	}
	if observation.CipherSuite != requiredCipherSuite {
		failures = append(failures, fmt.Sprintf("cipher suite mismatch: expected %s, observed %s", requiredCipherSuite, observation.CipherSuite))
	}
	if len(state.VerifiedChains) == 0 {
		failures = append(failures, "authenticated certificate chain was not verified")
	}
	if observation.CertificateDERSHA256 != leafDERHash {
		failures = append(failures, "certificate DER SHA-256 mismatch")
	}
	if observation.CertificateSPKISHA256 != leafSPKIHash {
		failures = append(failures, "certificate SPKI SHA-256 mismatch")
	}
	if len(failures) > 0 {
		return observation, errors.New(strings.Join(failures, "; "))
	}
	return observation, nil
}

func normalizeMetrics(summary phaseSummary) latencyMetrics {
	duration := summary.DurationSeconds
	if duration <= 0 {
		duration = 1
	}
	return latencyMetrics{
		HandshakesPerSecond:                 float64(summary.CompletedOperations) / duration,
		TLSHandshakeLatencyMeanMS:           mean(summary.TLSHandshakeLatencyMilliseconds),
		TLSHandshakeLatencyP50MS:            percentile(summary.TLSHandshakeLatencyMilliseconds, .50),
		TLSHandshakeLatencyP75MS:            percentile(summary.TLSHandshakeLatencyMilliseconds, .75),
		TLSHandshakeLatencyP90MS:            percentile(summary.TLSHandshakeLatencyMilliseconds, .90),
		TLSHandshakeLatencyP95MS:            percentile(summary.TLSHandshakeLatencyMilliseconds, .95),
		TLSHandshakeLatencyP99MS:            percentile(summary.TLSHandshakeLatencyMilliseconds, .99),
		ConnectionAndHandshakeLatencyMeanMS: mean(summary.ConnectionAndHandshakeMilliseconds),
		CompletedOperations:                 summary.CompletedOperations, FailedOperations: summary.FailedOperations, TimedOutOperations: summary.TimedOutOperations,
	}
}

func mean(values []float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}
	if len(values) == 0 {
		return 0
	}
	return total / float64(len(values))
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
func durationMS(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }
func hash(value []byte) string               { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func tlsVersionName(value uint16) string {
	if value == tls.VersionTLS13 {
		return "TLS1.3"
	}
	return fmt.Sprintf("0x%04x", value)
}
func loadRoots(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(data) {
		return nil, errors.New("root certificate PEM did not contain a certificate")
	}
	return roots, nil
}
func normalizeTarget(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("target-address or PLAB_TARGET_BASE_URL is required")
	}
	if !strings.Contains(value, "://") {
		if _, _, err := net.SplitHostPort(value); err != nil {
			return "", err
		}
		return value, nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "tls" || parsed.Host == "" {
		return "", errors.New("TLS target must use tls://host:port")
	}
	return parsed.Host, nil
}
func envInt(name string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	return value
}
func operationTimeout() time.Duration {
	seconds := envInt("PLAB_REQUEST_TIMEOUT_SECONDS")
	if seconds == 0 {
		seconds = 5
	}
	return time.Duration(seconds) * time.Second
}
func verifySubstitution(variable, expected, label string) {
	if observed := strings.TrimSpace(os.Getenv(variable)); observed != "" && observed != expected {
		fatal(2, fmt.Errorf("%s substitution detected: expected %q, observed %q", label, expected, observed))
	}
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	var networkError net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout())
}
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func writeRequired(dir, name string, value any) {
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
