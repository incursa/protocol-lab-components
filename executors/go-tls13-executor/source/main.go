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
	"io"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	executorID           = "go-tls13-executor"
	executorVersion      = "0.2.0"
	loadGeneratorID      = "go-crypto-tls13-handshake-load"
	loadGeneratorVersion = "0.2.0"
	fullScenarioID       = "tls.handshake.full"
	resumedScenarioID    = "tls.handshake.resumed"
	loadProfileID        = "tls-smoke"
	serverName           = "tls.plab.test"
	alpn                 = "protocol-lab-tls"
	requiredCipherSuite  = "TLS_AES_128_GCM_SHA256"
	requiredKeyExchange  = "X25519"
	certificateProfile   = "plab-single-leaf-p256-v1"
	leafDERHash          = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
	leafSPKIHash         = "407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0"
)

type loadConfig struct {
	ScenarioID              string
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
	LastResumptionProof                *resumptionProof      `json:"lastResumptionProof,omitempty"`
	Errors                             map[string]int        `json:"errors"`
}

type resumptionProof struct {
	ScenarioID                        string               `json:"scenarioId"`
	ResumptionPolicy                  string               `json:"resumptionPolicy"`
	PrerequisitePolicy                string               `json:"prerequisitePolicy"`
	WarmupIsolation                   string               `json:"warmupIsolation"`
	MeasuredWindow                    string               `json:"measuredWindow"`
	SourceSession                     handshakeObservation `json:"sourceSession"`
	MeasuredSession                   handshakeObservation `json:"measuredSession"`
	SourceHandshakeOutsideMeasured    bool                 `json:"sourceHandshakeOutsideMeasuredWindow"`
	SessionTicketAvailableAfterSource bool                 `json:"sessionTicketAvailableAfterSource"`
	SessionTicketConsumedExactlyOnce  bool                 `json:"sessionTicketConsumedExactlyOnce"`
	WarmupSessionStateReused          bool                 `json:"warmupSessionStateReusedByMeasurement"`
	EarlyDataAttempted                bool                 `json:"earlyDataAttempted"`
	ApplicationDataBytes              int                  `json:"applicationDataBytes"`
	CachePutCountAfterSource          int                  `json:"cachePutCountAfterSource"`
	CacheGetCountForMeasuredHandshake int                  `json:"cacheGetCountForMeasuredHandshake"`
	CacheHitCountForMeasuredHandshake int                  `json:"cacheHitCountForMeasuredHandshake"`
}

type singleUseSessionCache struct {
	mu      sync.Mutex
	key     string
	session *tls.ClientSessionState
	puts    int
	gets    int
	hits    int
}

func (cache *singleUseSessionCache) Put(key string, session *tls.ClientSessionState) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.puts++
	cache.key = key
	cache.session = session
}

func (cache *singleUseSessionCache) Get(key string) (*tls.ClientSessionState, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.gets++
	if cache.session == nil || cache.key != key {
		return nil, false
	}
	session := cache.session
	cache.session = nil
	cache.hits++
	return session, true
}

func (cache *singleUseSessionCache) counts() (puts, gets, hits int, available bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.puts, cache.gets, cache.hits, cache.session != nil
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
	verifySubstitution("PLAB_LOAD_GENERATOR_ID", loadGeneratorID, "load generator")
	verifySubstitution("PLAB_LOAD_GENERATOR_VERSION", loadGeneratorVersion, "load generator version")
	verifySubstitution("PLAB_PROTOCOL", "tls", "protocol")
	if strings.TrimSpace(*outputDir) == "" {
		*outputDir = "artifacts"
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fatal(1, err)
	}
	requestedID := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if isKnownUnsupportedScenario(requestedID) {
		emitUnsupported(*outputDir, requestedID)
		os.Exit(3)
	}
	scenario, protocolVariant, err := requestedScenario()
	if err != nil {
		fatal(2, err)
	}
	verifySubstitution("PLAB_PROTOCOL_VARIANT", protocolVariant, "protocol variant")

	address, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	if strings.TrimSpace(*rootCertificate) == "" {
		*rootCertificate = filepath.Join("certs", "root.pem")
	}
	roots, err := loadRoots(*rootCertificate)
	if err != nil {
		fatal(2, err)
	}

	preflight, preflightResumption, err := runOperation(context.Background(), scenario, address, roots, 5*time.Second)
	validation := map[string]any{
		"scenarioId":             scenario,
		"passed":                 err == nil,
		"requestedProtocol":      protocolVariant,
		"observedProtocol":       preflight.TLSVersion,
		"fallbackDetected":       preflight.TLSVersion != "TLS1.3",
		"didResume":              preflight.DidResume,
		"unexpectedFailureCount": boolInt(err != nil),
		"timeoutCount":           boolInt(isTimeout(err)),
		"error":                  errorString(err),
	}
	writeRequired(*outputDir, "validation.json", validation)
	writeRequired(*outputDir, "result.json", validation)
	writeRequired(*outputDir, "protocol-proof.json", preflight)
	writeRequired(*outputDir, "tls-negotiation.json", preflight)
	if scenario == resumedScenarioID {
		writeRequired(*outputDir, "resumption-proof.json", preflightResumption)
	}
	writeRequired(*outputDir, "executor-identity.json", map[string]any{
		"id": executorID, "version": executorVersion, "role": "client-test-executor",
		"supportedProtocols": []string{"tls"}, "supportedScenarios": []string{fullScenarioID, resumedScenarioID},
		"supportedLoadProfiles": []string{loadProfileID},
	})
	if err != nil {
		fatal(1, fmt.Errorf("TLS validity handshake failed: %w", err))
	}
	if *validationOnly {
		fmt.Fprintf(os.Stderr, "go-tls13-executor validation passed with exact %s proof\n", protocolVariant)
		return
	}

	config, err := loadConfigFromEnvironment()
	if err != nil {
		fatal(2, err)
	}
	if config.Warmup > 0 {
		warmup := runPhase(config.ScenarioID, address, roots, config.Warmup, config.ConnectionTimeout, "warmup")
		writeRequired(*outputDir, "tls-warmup-summary.json", warmup)
		if warmup.FailedOperations != 0 || warmup.TimedOutOperations != 0 || warmup.CompletedOperations == 0 {
			fatal(1, errors.New("TLS warmup did not satisfy the minimal validity gate"))
		}
	}
	measured := runPhase(config.ScenarioID, address, roots, config.Duration, config.ConnectionTimeout, "measured")
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
		Warnings: []string{"TLS handshake smoke evidence is local and non-publishable; 0-RTT, alternate cipher/version/authentication profiles, record workloads, comparison, and ranking are unsupported."},
	}
	writeRequired(*outputDir, "tls-topology.json", map[string]any{"schemaVersion": "protocol-lab.tls-topology.v1", "requested": shape, "effective": effective})
	writeRequired(*outputDir, "connection-and-handshake-latency.json", map[string]any{"samplesMilliseconds": measured.ConnectionAndHandshakeMilliseconds})
	writeRequired(*outputDir, "load-generator-identity.json", result.LoadGenerator)
	if config.ScenarioID == resumedScenarioID {
		writeRequired(*outputDir, "resumption-proof.json", measured.LastResumptionProof)
	}
	writeRequired(*outputDir, "tls-executor-result.json", result)
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func loadConfigFromEnvironment() (loadConfig, error) {
	scenario, _, err := requestedScenario()
	if err != nil {
		return loadConfig{}, err
	}
	if strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID")) != loadProfileID {
		return loadConfig{}, fmt.Errorf("go-tls13-executor supports load profile %q only", loadProfileID)
	}
	config := loadConfig{
		ScenarioID:  scenario,
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
		return config, fmt.Errorf("tls-smoke with %s requires connections=1 concurrency=1 one handshake per connection, zero application bytes, duration=5s warmup=1s repetition=1 operationTimeout=5s; observed %+v", scenario, config)
	}
	return config, nil
}

func requestedScenario() (scenario, variant string, err error) {
	scenario = strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	switch scenario {
	case fullScenarioID:
		return scenario, "tls1.3-full", nil
	case resumedScenarioID:
		return scenario, "tls1.3-psk-resumed", nil
	default:
		return "", "", fmt.Errorf("unknown or invalid TLS scenario %q; supported scenarios are %q and %q", scenario, fullScenarioID, resumedScenarioID)
	}
}

func isKnownUnsupportedScenario(scenario string) bool {
	switch scenario {
	case "tls.handshake.full.tls12",
		"tls.handshake.full.chacha20",
		"tls.handshake.mutual-auth",
		"tls.early-data.accepted",
		"tls.early-data.rejected",
		"tls.key-update.diagnostic",
		"tls.record.coverage",
		"tls.record.throughput":
		return true
	default:
		return false
	}
}

func emitUnsupported(outputDir, scenario string) {
	unsupported := map[string]any{
		"schemaVersion":      "protocol-lab.unsupported.v1",
		"status":             "unsupported",
		"scenarioId":         scenario,
		"reasonCode":         "scenario-not-implemented",
		"message":            fmt.Sprintf("go-tls13-executor@%s recognizes %s but does not implement its exact semantics", executorVersion, scenario),
		"authorityCommit":    "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574",
		"executor":           map[string]string{"id": executorID, "version": executorVersion},
		"loadGenerator":      map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion},
		"supportedScenarios": []string{fullScenarioID, resumedScenarioID},
	}
	writeRequired(outputDir, "unsupported.json", unsupported)
	writeRequired(outputDir, "result.json", unsupported)
	writeRequired(outputDir, "executor-identity.json", map[string]any{
		"id": executorID, "version": executorVersion, "role": "client-test-executor",
		"supportedProtocols": []string{"tls"}, "supportedScenarios": []string{fullScenarioID, resumedScenarioID},
		"supportedLoadProfiles": []string{loadProfileID},
	})
	data, err := json.MarshalIndent(unsupported, "", "  ")
	if err == nil {
		fmt.Println(string(data))
	}
}

func runPhase(scenario, address string, roots *x509.CertPool, duration, timeout time.Duration, phase string) phaseSummary {
	started := time.Now()
	summary := phaseSummary{Phase: phase, MaximumEffectiveConcurrency: 1, Errors: map[string]int{}}
	measuredDuration := time.Duration(0)
	for measuredDuration < duration {
		observation, resumption, err := runOperation(context.Background(), scenario, address, roots, timeout)
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
		if resumption != nil {
			summary.LastResumptionProof = resumption
		}
		measuredDuration += time.Duration(observation.TLSHandshakeLatencyMS * float64(time.Millisecond))
	}
	summary.DurationSeconds = measuredDuration.Seconds()
	if scenario == fullScenarioID {
		summary.DurationSeconds = time.Since(started).Seconds()
	}
	return summary
}

func runOperation(ctx context.Context, scenario, address string, roots *x509.CertPool, timeout time.Duration) (handshakeObservation, *resumptionProof, error) {
	if scenario == fullScenarioID {
		observation, err := runHandshake(ctx, address, roots, timeout, nil, false, false)
		return observation, nil, err
	}
	cache := &singleUseSessionCache{}
	source, err := runHandshake(ctx, address, roots, timeout, cache, false, true)
	if err != nil {
		return source, nil, fmt.Errorf("unmeasured source handshake failed: %w", err)
	}
	puts, sourceGets, sourceHits, available := cache.counts()
	if !available || puts < 1 {
		return source, nil, errors.New("unmeasured source handshake did not yield a resumable TLS 1.3 session ticket")
	}
	measured, err := runHandshake(ctx, address, roots, timeout, cache, true, false)
	_, totalGets, totalHits, availableAfter := cache.counts()
	measuredGets := totalGets - sourceGets
	measuredHits := totalHits - sourceHits
	proof := &resumptionProof{
		ScenarioID:                        resumedScenarioID,
		ResumptionPolicy:                  "accepted-psk-single-use-ticket",
		PrerequisitePolicy:                "unmeasured-source-session-per-measured-operation",
		WarmupIsolation:                   "warmup-state-not-reused-by-measurement",
		MeasuredWindow:                    "resumed-handshake",
		SourceSession:                     source,
		MeasuredSession:                   measured,
		SourceHandshakeOutsideMeasured:    true,
		SessionTicketAvailableAfterSource: true,
		SessionTicketConsumedExactlyOnce:  measuredGets == 1 && measuredHits == 1 && !availableAfter,
		WarmupSessionStateReused:          false,
		EarlyDataAttempted:                false,
		ApplicationDataBytes:              0,
		CachePutCountAfterSource:          puts,
		CacheGetCountForMeasuredHandshake: measuredGets,
		CacheHitCountForMeasuredHandshake: measuredHits,
	}
	if err != nil {
		return measured, proof, fmt.Errorf("measured resumed handshake failed: %w", err)
	}
	if !proof.SessionTicketConsumedExactlyOnce {
		return measured, proof, fmt.Errorf("session ticket was not consumed exactly once: measuredGets=%d measuredHits=%d availableAfter=%t", measuredGets, measuredHits, availableAfter)
	}
	return measured, proof, nil
}

func runHandshake(ctx context.Context, address string, roots *x509.CertPool, timeout time.Duration, cache tls.ClientSessionCache, expectResume, drainTicket bool) (handshakeObservation, error) {
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
		CurvePreferences:   []tls.CurveID{tls.X25519},
		ClientSessionCache: cache,
	})
	handshakeStart := time.Now()
	if err := client.HandshakeContext(operationContext); err != nil {
		return handshakeObservation{}, err
	}
	handshakeFinished := time.Now()
	state := client.ConnectionState()
	observation, err := validateState(state, expectResume)
	observation.TLSHandshakeLatencyMS = durationMS(handshakeFinished.Sub(handshakeStart))
	observation.ConnectionAndHandshakeLatencyMS = durationMS(handshakeFinished.Sub(connectionStart))
	if err == nil && drainTicket {
		err = drainSessionTicket(client, timeout)
	}
	_ = client.Close()
	return observation, err
}

func drainSessionTicket(client *tls.Conn, timeout time.Duration) error {
	if err := client.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	buffer := make([]byte, 1)
	for {
		n, err := client.Read(buffer)
		if n != 0 {
			return errors.New("TLS handshake workload received unexpected application data while awaiting the session ticket")
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("session ticket was not received before the source connection ended: %w", err)
		}
	}
}

func validateState(state tls.ConnectionState, expectResume bool) (handshakeObservation, error) {
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
	if state.DidResume != expectResume {
		failures = append(failures, fmt.Sprintf("session resumption mismatch: expected didResume=%t, observed %t", expectResume, state.DidResume))
	}
	if state.NegotiatedProtocol != alpn {
		failures = append(failures, "ALPN mismatch")
	}
	if observation.CipherSuite != requiredCipherSuite {
		failures = append(failures, fmt.Sprintf("cipher suite mismatch: expected %s, observed %s", requiredCipherSuite, observation.CipherSuite))
	}
	if observation.KeyExchangeGroup != requiredKeyExchange {
		failures = append(failures, fmt.Sprintf("key-exchange group mismatch: expected %s, observed %s", requiredKeyExchange, observation.KeyExchangeGroup))
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
