package main

import (
	"context"
	"crypto/ecdsa"
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
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	executorID                 = "go-tls13-executor"
	executorVersion            = "0.3.2"
	loadGeneratorID            = "go-crypto-tls13-load"
	loadGeneratorVersion       = "0.3.2"
	fullScenarioID             = "tls.handshake.full"
	resumedScenarioID          = "tls.handshake.resumed"
	recordThroughputScenarioID = "tls.record.throughput"
	recordCoverageScenarioID   = "tls.record.coverage"
	tlsSmokeProfileID          = "tls-smoke"
	tlsDiagnosticProfileID     = "tls-diagnostic"
	serverName                 = "tls.plab.test"
	alpn                       = "protocol-lab-tls"
	requiredCipherSuite        = "TLS_AES_128_GCM_SHA256"
	requiredKeyExchange        = "X25519"
	certificateProfile         = "plab-single-leaf-p256-v1"
	leafDERHash                = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
	leafSPKIHash               = "407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0"
	commandMagic               = "PLABTLS1"
	payloadOctet               = byte(0x5a)
)

var supportedScenarioIDs = []string{fullScenarioID, resumedScenarioID, recordThroughputScenarioID, recordCoverageScenarioID}

type loadConfig struct {
	ScenarioID              string
	LoadProfileID           string
	Connections             int
	Concurrency             int
	HandshakesPerConnection int
	ApplicationDataBytes    int
	TotalOperations         int
	Duration                time.Duration
	Warmup                  time.Duration
	Repetition              int
	ConnectionTimeout       time.Duration
}

type handshakeObservation struct {
	TLSProfileID                    string  `json:"tlsProfileId"`
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

type recordCounter struct {
	Records         int `json:"records"`
	CiphertextBytes int `json:"ciphertextBytes"`
}
type recordSnapshot struct {
	Sent     recordCounter `json:"sent"`
	Received recordCounter `json:"received"`
}
type recordCaseObservation struct {
	CaseID               string               `json:"caseId"`
	Direction            string               `json:"direction"`
	ApplicationDataBytes int                  `json:"applicationDataBytes"`
	PayloadSHA256        string               `json:"payloadSha256"`
	PayloadGenerator     string               `json:"payloadGenerator"`
	PayloadOctet         int                  `json:"payloadOctet"`
	MeasuredWindow       string               `json:"measuredWindow"`
	TransferLatencyMS    float64              `json:"transferLatencyMs"`
	BytesPerSecond       float64              `json:"bytesPerSecond"`
	WireRecordDelta      recordSnapshot       `json:"wireRecordDelta"`
	Negotiation          handshakeObservation `json:"negotiation"`
}

type phaseSummary struct {
	Phase                              string                  `json:"phase"`
	DurationSeconds                    float64                 `json:"durationSeconds"`
	CompletedOperations                int                     `json:"completedOperations"`
	FailedOperations                   int                     `json:"failedOperations"`
	TimedOutOperations                 int                     `json:"timedOutOperations"`
	MaximumEffectiveConcurrency        int                     `json:"maximumEffectiveConcurrency"`
	TLSHandshakeLatencyMilliseconds    []float64               `json:"tlsHandshakeLatencyMilliseconds"`
	ConnectionAndHandshakeMilliseconds []float64               `json:"connectionAndHandshakeLatencyMilliseconds"`
	TransferLatencyMilliseconds        []float64               `json:"transferLatencyMilliseconds"`
	TotalTransferredBytes              int64                   `json:"totalTransferredBytes"`
	LastNegotiation                    *handshakeObservation   `json:"lastNegotiation,omitempty"`
	LastResumptionProof                *resumptionProof        `json:"lastResumptionProof,omitempty"`
	LastRecordCases                    []recordCaseObservation `json:"lastRecordCases,omitempty"`
	Errors                             map[string]int          `json:"errors"`
}

type singleUseSessionCache struct {
	mu               sync.Mutex
	key              string
	session          *tls.ClientSessionState
	puts, gets, hits int
}

func (c *singleUseSessionCache) Put(key string, session *tls.ClientSessionState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.puts++
	c.key = key
	c.session = session
}
func (c *singleUseSessionCache) Get(key string) (*tls.ClientSessionState, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gets++
	if c.session == nil || c.key != key {
		return nil, false
	}
	s := c.session
	c.session = nil
	c.hits++
	return s, true
}
func (c *singleUseSessionCache) counts() (int, int, int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.puts, c.gets, c.hits, c.session != nil
}

type loadShape struct {
	Connections             int     `json:"connections"`
	Concurrency             int     `json:"concurrency"`
	HandshakesPerConnection int     `json:"handshakesPerConnection"`
	ApplicationDataBytes    int     `json:"applicationDataBytes"`
	TotalOperations         *int    `json:"totalOperations,omitempty"`
	DurationSeconds         float64 `json:"durationSeconds"`
	WarmupSeconds           float64 `json:"warmupSeconds"`
	Repetition              int     `json:"repetition"`
}
type normalizedMetrics struct {
	HandshakesPerSecond                 float64 `json:"handshakesPerSecond"`
	TLSHandshakeLatencyMeanMS           float64 `json:"tlsHandshakeLatencyMeanMs"`
	TLSHandshakeLatencyP50MS            float64 `json:"tlsHandshakeLatencyP50Ms"`
	TLSHandshakeLatencyP75MS            float64 `json:"tlsHandshakeLatencyP75Ms"`
	TLSHandshakeLatencyP90MS            float64 `json:"tlsHandshakeLatencyP90Ms"`
	TLSHandshakeLatencyP95MS            float64 `json:"tlsHandshakeLatencyP95Ms"`
	TLSHandshakeLatencyP99MS            float64 `json:"tlsHandshakeLatencyP99Ms"`
	ConnectionAndHandshakeLatencyMeanMS float64 `json:"connectionAndHandshakeLatencyMeanMs"`
	BytesPerSecond                      float64 `json:"bytesPerSecond"`
	TransferLatencyMeanMS               float64 `json:"transferLatencyMeanMs"`
	TransferLatencyP50MS                float64 `json:"transferLatencyP50Ms"`
	TransferLatencyP75MS                float64 `json:"transferLatencyP75Ms"`
	TransferLatencyP90MS                float64 `json:"transferLatencyP90Ms"`
	TransferLatencyP95MS                float64 `json:"transferLatencyP95Ms"`
	TransferLatencyP99MS                float64 `json:"transferLatencyP99Ms"`
	TotalTransferredBytes               int64   `json:"totalTransferredBytes"`
	CompletedOperations                 int     `json:"completedOperations"`
	FailedOperations                    int     `json:"failedOperations"`
	TimedOutOperations                  int     `json:"timedOutOperations"`
}
type executorResult struct {
	SchemaVersion string               `json:"schemaVersion"`
	ScenarioID    string               `json:"scenarioId"`
	Mode          string               `json:"mode"`
	Executor      map[string]string    `json:"executor"`
	LoadGenerator map[string]string    `json:"loadGenerator"`
	Validation    map[string]string    `json:"validation"`
	ProtocolProof handshakeObservation `json:"protocolProof"`
	RequestedLoad loadShape            `json:"requestedLoad"`
	EffectiveLoad loadShape            `json:"effectiveLoad"`
	Metrics       normalizedMetrics    `json:"metrics"`
	Warnings      []string             `json:"warnings"`
}

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "TLS target address or tls:// URL")
	outputDir := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "Artifact output directory")
	rootCertificate := flag.String("root-certificate", os.Getenv("PLAB_TLS_ROOT_CERTIFICATE_PATH"), "Public test root PEM")
	validationOnly := flag.Bool("validation-only", false, "Run one validity operation and stop")
	showVersion := flag.Bool("version", false, "Print executor version and exit")
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
	scenario, variant, err := requestedScenario()
	if err != nil {
		fatal(2, err)
	}
	verifySubstitution("PLAB_PROTOCOL_VARIANT", variant, "protocol variant")
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

	preflight, preflightResume, preflightCases, err := runOperation(context.Background(), scenario, address, roots, 15*time.Second)
	validation := map[string]any{"scenarioId": scenario, "passed": err == nil, "requestedProtocol": variant, "observedProtocol": preflight.TLSVersion, "fallbackDetected": preflight.TLSVersion != "TLS1.3", "didResume": preflight.DidResume, "earlyDataAttempted": false, "unexpectedFailureCount": boolInt(err != nil), "timeoutCount": boolInt(isTimeout(err)), "error": errorString(err)}
	writeRequired(*outputDir, "validation.json", validation)
	writeRequired(*outputDir, "result.json", validation)
	writeRequired(*outputDir, "protocol-proof.json", preflight)
	writeRequired(*outputDir, "tls-negotiation.json", preflight)
	if scenario == resumedScenarioID {
		writeRequired(*outputDir, "resumption-proof.json", preflightResume)
	}
	if isRecordScenario(scenario) {
		writeRecordArtifacts(*outputDir, scenario, preflightCases)
	}
	writeIdentity(*outputDir)
	if err != nil {
		fatal(1, fmt.Errorf("TLS validity operation failed: %w", err))
	}
	if *validationOnly {
		fmt.Fprintf(os.Stderr, "go-tls13-executor validation passed with exact %s proof\n", variant)
		return
	}

	config, err := loadConfigFromEnvironment()
	if err != nil {
		fatal(2, err)
	}
	if config.Warmup > 0 {
		warmup := runPhase(config, address, roots, "warmup")
		writeRequired(*outputDir, "tls-warmup-summary.json", warmup)
		if warmup.FailedOperations != 0 || warmup.TimedOutOperations != 0 || warmup.CompletedOperations == 0 {
			fatal(1, errors.New("TLS warmup did not satisfy the minimal validity gate"))
		}
	}
	measured := runPhase(config, address, roots, "measured")
	writeRequired(*outputDir, "tls-load-summary.json", measured)
	if measured.FailedOperations != 0 || measured.TimedOutOperations != 0 || measured.CompletedOperations == 0 || measured.LastNegotiation == nil {
		fatal(1, fmt.Errorf("TLS measured phase rejected: completed=%d failed=%d timedOut=%d", measured.CompletedOperations, measured.FailedOperations, measured.TimedOutOperations))
	}
	shape := shapeFrom(config)
	effective := shape
	effective.Concurrency = measured.MaximumEffectiveConcurrency
	metrics := normalizeMetrics(measured)
	result := executorResult{SchemaVersion: "protocol-lab.tls-executor-result.v1", ScenarioID: scenario, Mode: scenarioMode(scenario), Executor: map[string]string{"id": executorID, "version": executorVersion}, LoadGenerator: map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, Validation: map[string]string{"status": "passed"}, ProtocolProof: *measured.LastNegotiation, RequestedLoad: shape, EffectiveLoad: effective, Metrics: metrics, Warnings: []string{"Local package smoke evidence is diagnostic and non-publishable; wire record deltas are encrypted-record observations and do not claim plaintext-to-record boundary control."}}
	writeRequired(*outputDir, "tls-topology.json", map[string]any{"schemaVersion": "protocol-lab.tls-topology.v1", "requested": shape, "effective": effective})
	writeRequired(*outputDir, "connection-and-handshake-latency.json", map[string]any{"samplesMilliseconds": measured.ConnectionAndHandshakeMilliseconds})
	writeRequired(*outputDir, "load-generator-identity.json", result.LoadGenerator)
	if scenario == resumedScenarioID {
		writeRequired(*outputDir, "resumption-proof.json", measured.LastResumptionProof)
	}
	if isRecordScenario(scenario) {
		writeRecordArtifacts(*outputDir, scenario, measured.LastRecordCases)
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
	profile := strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID"))
	c := loadConfig{ScenarioID: scenario, LoadProfileID: profile, Connections: envInt("PLAB_CONNECTIONS"), Concurrency: envInt("PLAB_CONCURRENCY"), HandshakesPerConnection: 1, ApplicationDataBytes: 0, TotalOperations: envInt("PLAB_TOTAL_OPERATIONS"), Duration: time.Duration(envInt("PLAB_DURATION_SECONDS")) * time.Second, Warmup: time.Duration(envInt("PLAB_WARMUP_SECONDS")) * time.Second, Repetition: envInt("PLAB_REPETITION"), ConnectionTimeout: operationTimeout()}
	if scenario == recordThroughputScenarioID {
		c.ApplicationDataBytes = 1048576
	}
	if scenario == recordCoverageScenarioID {
		if profile == tlsSmokeProfileID {
			if c.Connections != 1 || c.Concurrency != 1 || c.Duration != 5*time.Second || c.Warmup != time.Second || c.Repetition < 1 || c.ConnectionTimeout != 5*time.Second {
				return c, fmt.Errorf("tls-smoke record coverage requires connections=1 concurrency=1 duration=5s warmup=1s repetition>=1 operationTimeout=5s; observed %+v", c)
			}
			return c, nil
		}
		if profile != tlsDiagnosticProfileID || c.Connections != 1 || c.Concurrency != 1 || c.TotalOperations != 1 || c.Duration != 10*time.Second || c.Warmup != 0 || c.Repetition < 1 || c.ConnectionTimeout != 15*time.Second {
			return c, fmt.Errorf("record coverage requires the exact tls-smoke or tls-diagnostic shape; observed %+v", c)
		}
		return c, nil
	}
	if profile != tlsSmokeProfileID || c.Connections != 1 || c.Concurrency != 1 || c.Duration != 5*time.Second || c.Warmup != time.Second || c.Repetition < 1 || c.ConnectionTimeout != 5*time.Second {
		return c, fmt.Errorf("tls-smoke with %s requires connections=1 concurrency=1 duration=5s warmup=1s repetition>=1 operationTimeout=5s; observed %+v", scenario, c)
	}
	return c, nil
}

func requestedScenario() (string, string, error) {
	s := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	switch s {
	case fullScenarioID:
		return s, "tls1.3-full", nil
	case resumedScenarioID:
		return s, "tls1.3-psk-resumed", nil
	case recordThroughputScenarioID:
		return s, "tls1.3-record", nil
	case recordCoverageScenarioID:
		return s, "tls1.3-record-coverage", nil
	default:
		return "", "", fmt.Errorf("unknown or invalid TLS scenario %q", s)
	}
}
func isKnownUnsupportedScenario(s string) bool {
	switch s {
	case "tls.handshake.full.tls12", "tls.handshake.full.chacha20", "tls.handshake.mutual-auth", "tls.early-data.accepted", "tls.early-data.rejected", "tls.key-update.diagnostic":
		return true
	default:
		return false
	}
}
func isRecordScenario(s string) bool {
	return s == recordThroughputScenarioID || s == recordCoverageScenarioID
}
func scenarioMode(s string) string {
	switch s {
	case fullScenarioID:
		return "full-handshake"
	case resumedScenarioID:
		return "resumed-handshake"
	case recordThroughputScenarioID:
		return "record-throughput"
	case recordCoverageScenarioID:
		return "record-coverage"
	}
	return ""
}

func emitUnsupported(dir, scenario string) {
	u := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": scenario, "reasonCode": "scenario-not-implemented", "message": fmt.Sprintf("go-tls13-executor@%s recognizes %s but does not implement its exact semantics", executorVersion, scenario), "authorityCommit": "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574", "executor": map[string]string{"id": executorID, "version": executorVersion}, "loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, "supportedScenarios": supportedScenarioIDs}
	writeRequired(dir, "unsupported.json", u)
	writeRequired(dir, "result.json", u)
	writeIdentity(dir)
	data, _ := json.MarshalIndent(u, "", "  ")
	fmt.Println(string(data))
}
func writeIdentity(dir string) {
	writeRequired(dir, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedProtocols": []string{"tls"}, "supportedScenarios": supportedScenarioIDs, "supportedLoadProfiles": []string{tlsSmokeProfileID, tlsDiagnosticProfileID}})
}

func runPhase(c loadConfig, address string, roots *x509.CertPool, phase string) phaseSummary {
	s := phaseSummary{Phase: phase, MaximumEffectiveConcurrency: 1, Errors: map[string]int{}}
	started := time.Now()
	limit := c.Duration
	if phase == "warmup" {
		limit = c.Warmup
	}
	for {
		if c.TotalOperations > 0 && s.CompletedOperations+s.FailedOperations+s.TimedOutOperations >= c.TotalOperations {
			break
		}
		if c.TotalOperations == 0 && time.Since(started) >= limit {
			break
		}
		o, r, cases, err := runOperation(context.Background(), c.ScenarioID, address, roots, c.ConnectionTimeout)
		if err != nil {
			if isTimeout(err) {
				s.TimedOutOperations++
			} else {
				s.FailedOperations++
			}
			s.Errors[errorString(err)]++
			continue
		}
		s.CompletedOperations++
		s.TLSHandshakeLatencyMilliseconds = append(s.TLSHandshakeLatencyMilliseconds, o.TLSHandshakeLatencyMS)
		s.ConnectionAndHandshakeMilliseconds = append(s.ConnectionAndHandshakeMilliseconds, o.ConnectionAndHandshakeLatencyMS)
		s.LastNegotiation = &o
		s.LastResumptionProof = r
		s.LastRecordCases = cases
		for _, c := range cases {
			s.TransferLatencyMilliseconds = append(s.TransferLatencyMilliseconds, c.TransferLatencyMS)
			s.TotalTransferredBytes += int64(c.ApplicationDataBytes)
		}
	}
	s.DurationSeconds = time.Since(started).Seconds()
	return s
}

func runOperation(ctx context.Context, scenario, address string, roots *x509.CertPool, timeout time.Duration) (handshakeObservation, *resumptionProof, []recordCaseObservation, error) {
	switch scenario {
	case fullScenarioID:
		o, err := runHandshake(ctx, address, roots, timeout, nil, false, false)
		return o, nil, nil, err
	case resumedScenarioID:
		o, p, err := runResumed(ctx, address, roots, timeout)
		return o, p, nil, err
	case recordThroughputScenarioID:
		c, err := runRecordCase(ctx, address, roots, timeout, "server-to-client-1mib", "server-to-client", 1048576)
		return c.Negotiation, nil, []recordCaseObservation{c}, err
	case recordCoverageScenarioID:
		definitions := []struct {
			id, direction string
			size          int
		}{{"client-to-server-1kib", "client-to-server", 1024}, {"server-to-client-1kib", "server-to-client", 1024}, {"client-to-server-64kib", "client-to-server", 65536}, {"server-to-client-64kib", "server-to-client", 65536}, {"client-to-server-1mib", "client-to-server", 1048576}, {"server-to-client-1mib", "server-to-client", 1048576}}
		cases := make([]recordCaseObservation, 0, 6)
		var last handshakeObservation
		for _, d := range definitions {
			c, err := runRecordCase(ctx, address, roots, timeout, d.id, d.direction, d.size)
			if err != nil {
				return c.Negotiation, nil, cases, fmt.Errorf("record coverage case %s failed: %w", d.id, err)
			}
			cases = append(cases, c)
			last = c.Negotiation
		}
		return last, nil, cases, nil
	default:
		return handshakeObservation{}, nil, nil, errors.New("unsupported operation")
	}
}

func runResumed(ctx context.Context, address string, roots *x509.CertPool, timeout time.Duration) (handshakeObservation, *resumptionProof, error) {
	cache := &singleUseSessionCache{}
	source, err := runHandshake(ctx, address, roots, timeout, cache, false, true)
	if err != nil {
		return source, nil, fmt.Errorf("unmeasured source handshake failed: %w", err)
	}
	puts, sourceGets, sourceHits, available := cache.counts()
	if !available || puts < 1 {
		return source, nil, errors.New("source handshake did not yield a resumable TLS 1.3 ticket")
	}
	measured, err := runHandshake(ctx, address, roots, timeout, cache, true, false)
	_, totalGets, totalHits, availableAfter := cache.counts()
	mg, mh := totalGets-sourceGets, totalHits-sourceHits
	p := &resumptionProof{ScenarioID: resumedScenarioID, ResumptionPolicy: "accepted-psk-single-use-ticket", PrerequisitePolicy: "unmeasured-source-session-per-measured-operation", WarmupIsolation: "warmup-state-not-reused-by-measurement", MeasuredWindow: "resumed-handshake", SourceSession: source, MeasuredSession: measured, SourceHandshakeOutsideMeasured: true, SessionTicketAvailableAfterSource: true, SessionTicketConsumedExactlyOnce: mg == 1 && mh == 1 && !availableAfter, WarmupSessionStateReused: false, EarlyDataAttempted: false, ApplicationDataBytes: 0, CachePutCountAfterSource: puts, CacheGetCountForMeasuredHandshake: mg, CacheHitCountForMeasuredHandshake: mh}
	if err != nil {
		return measured, p, err
	}
	if !p.SessionTicketConsumedExactlyOnce {
		return measured, p, errors.New("session ticket was not consumed exactly once")
	}
	return measured, p, nil
}

func runHandshake(ctx context.Context, address string, roots *x509.CertPool, timeout time.Duration, cache tls.ClientSessionCache, expectResume, drainTicket bool) (handshakeObservation, error) {
	start := time.Now()
	op, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	raw, err := (&net.Dialer{}).DialContext(op, "tcp", address)
	if err != nil {
		return handshakeObservation{}, err
	}
	defer raw.Close()
	client := tls.Client(raw, tlsConfig(roots, cache))
	hs := time.Now()
	if err := client.HandshakeContext(op); err != nil {
		return handshakeObservation{}, err
	}
	finished := time.Now()
	o, err := validateState(client.ConnectionState(), expectResume)
	o.TLSHandshakeLatencyMS = durationMS(finished.Sub(hs))
	o.ConnectionAndHandshakeLatencyMS = durationMS(finished.Sub(start))
	if err == nil && drainTicket {
		err = drainSessionTicket(client, timeout)
	}
	_ = client.Close()
	return o, err
}

func runRecordCase(ctx context.Context, address string, roots *x509.CertPool, timeout time.Duration, caseID, direction string, size int) (recordCaseObservation, error) {
	o := recordCaseObservation{CaseID: caseID, Direction: direction, ApplicationDataBytes: size, PayloadSHA256: payloadHash(size), PayloadGenerator: "repeated-octet", PayloadOctet: int(payloadOctet), MeasuredWindow: "application-data-transfer"}
	op, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	raw, err := (&net.Dialer{}).DialContext(op, "tcp", address)
	if err != nil {
		return o, err
	}
	observed := newRecordObservingConn(raw)
	defer observed.Close()
	client := tls.Client(observed, tlsConfig(roots, nil))
	hs := time.Now()
	if err := client.HandshakeContext(op); err != nil {
		return o, err
	}
	finished := time.Now()
	neg, err := validateState(client.ConnectionState(), false)
	if strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")) == recordCoverageScenarioID {
		neg.TLSProfileID = "plab-tls13-aes128gcm-p256-server-auth-v2"
		neg.CertificateProfile = "plab-single-leaf-p256-server-v2"
	}
	neg.TLSHandshakeLatencyMS = durationMS(finished.Sub(hs))
	neg.ConnectionAndHandshakeLatencyMS = neg.TLSHandshakeLatencyMS
	if err != nil {
		return o, err
	}
	o.Negotiation = neg
	dir := byte('S')
	if direction == "client-to-server" {
		dir = 'C'
	}
	header := make([]byte, 13)
	copy(header, commandMagic)
	header[8] = dir
	binary.BigEndian.PutUint32(header[9:], uint32(size))
	if err := writeAll(client, header); err != nil {
		return o, err
	}
	before := observed.snapshot()
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = payloadOctet
	}
	started := time.Now()
	if dir == 'S' {
		received := make([]byte, size)
		if _, err := io.ReadFull(client, received); err != nil {
			return o, err
		}
		if hash(received) != o.PayloadSHA256 {
			return o, errors.New("server-to-client payload hash mismatch")
		}
		neg.ApplicationDataBytesReceived = size
	} else {
		if err := writeAll(client, payload); err != nil {
			return o, err
		}
		ack := make([]byte, 34)
		if _, err := io.ReadFull(client, ack); err != nil {
			return o, err
		}
		if string(ack[:2]) != "OK" || hex.EncodeToString(ack[2:]) != o.PayloadSHA256 {
			return o, errors.New("client-to-server acknowledgement hash mismatch")
		}
		neg.ApplicationDataBytesSent = size
	}
	elapsed := time.Since(started)
	after := observed.snapshot()
	o.TransferLatencyMS = durationMS(elapsed)
	if elapsed > 0 {
		o.BytesPerSecond = float64(size) / elapsed.Seconds()
	}
	o.WireRecordDelta = subtractSnapshot(after, before)
	if direction == "server-to-client" && o.WireRecordDelta.Received.Records < 1 {
		return o, errors.New("no encrypted server-to-client TLS records were observed in the measured transfer window")
	}
	if direction == "client-to-server" && o.WireRecordDelta.Sent.Records < 1 {
		return o, errors.New("no encrypted client-to-server TLS records were observed in the measured transfer window")
	}
	o.Negotiation = neg
	_ = client.Close()
	return o, nil
}

func tlsConfig(roots *x509.CertPool, cache tls.ClientSessionCache) *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: roots, ServerName: serverName, NextProtos: []string{alpn}, CurvePreferences: []tls.CurveID{tls.X25519}, ClientSessionCache: cache}
}
func drainSessionTicket(client *tls.Conn, timeout time.Duration) error {
	if err := client.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	b := make([]byte, 1)
	for {
		n, err := client.Read(b)
		if n != 0 {
			return errors.New("unexpected application data while awaiting ticket")
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("session ticket was not received: %w", err)
		}
	}
}

func validateState(state tls.ConnectionState, expectResume bool) (handshakeObservation, error) {
	o := handshakeObservation{TLSVersion: tlsVersionName(state.Version), CipherSuite: tls.CipherSuiteName(state.CipherSuite), KeyExchangeGroup: state.CurveID.String(), ALPN: state.NegotiatedProtocol, ServerName: serverName, HandshakeComplete: state.HandshakeComplete, DidResume: state.DidResume, EarlyDataAttempted: false, CertificateProfile: certificateProfile, VerifiedChainCount: len(state.VerifiedChains)}
	o.TLSProfileID = "plab-tls13-p256-v1"
	if len(state.PeerCertificates) > 0 {
		c := state.PeerCertificates[0]
		o.CertificateDERSHA256 = hash(c.Raw)
		o.CertificateSPKISHA256 = hash(c.RawSubjectPublicKeyInfo)
		o.CertificateSignatureAlgorithm = c.SignatureAlgorithm.String()
		o.CertificatePublicKeyAlgorithm = c.PublicKeyAlgorithm.String()
		if k, ok := c.PublicKey.(*ecdsa.PublicKey); ok {
			o.CertificateNamedCurve = k.Curve.Params().Name
		}
	}
	var f []string
	if state.Version != tls.VersionTLS13 {
		f = append(f, "exact TLS 1.3 was not negotiated")
	}
	if !state.HandshakeComplete {
		f = append(f, "handshake incomplete")
	}
	if state.DidResume != expectResume {
		f = append(f, "session state mismatch")
	}
	if state.NegotiatedProtocol != alpn {
		f = append(f, "ALPN mismatch")
	}
	if o.CipherSuite != requiredCipherSuite {
		f = append(f, "cipher mismatch")
	}
	if o.KeyExchangeGroup != requiredKeyExchange {
		f = append(f, "key exchange mismatch")
	}
	if len(state.VerifiedChains) == 0 {
		f = append(f, "certificate not verified")
	}
	if o.CertificateDERSHA256 != leafDERHash || o.CertificateSPKISHA256 != leafSPKIHash {
		f = append(f, "certificate hash mismatch")
	}
	if len(f) > 0 {
		return o, errors.New(strings.Join(f, "; "))
	}
	return o, nil
}

type tlsRecordParser struct {
	pending []byte
	count   int
	bytes   int
}

func (p *tlsRecordParser) feed(value []byte) {
	p.pending = append(p.pending, value...)
	for len(p.pending) >= 5 {
		length := int(binary.BigEndian.Uint16(p.pending[3:5]))
		if length > 18432 {
			p.pending = nil
			return
		}
		if len(p.pending) < 5+length {
			return
		}
		p.count++
		p.bytes += 5 + length
		p.pending = p.pending[5+length:]
	}
}

type recordObservingConn struct {
	net.Conn
	mu             sync.Mutex
	sent, received tlsRecordParser
}

func newRecordObservingConn(c net.Conn) *recordObservingConn { return &recordObservingConn{Conn: c} }
func (c *recordObservingConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	c.mu.Lock()
	c.sent.feed(p[:n])
	c.mu.Unlock()
	return n, err
}
func (c *recordObservingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.mu.Lock()
	c.received.feed(p[:n])
	c.mu.Unlock()
	return n, err
}
func (c *recordObservingConn) snapshot() recordSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return recordSnapshot{Sent: recordCounter{c.sent.count, c.sent.bytes}, Received: recordCounter{c.received.count, c.received.bytes}}
}
func subtractSnapshot(a, b recordSnapshot) recordSnapshot {
	return recordSnapshot{Sent: recordCounter{a.Sent.Records - b.Sent.Records, a.Sent.CiphertextBytes - b.Sent.CiphertextBytes}, Received: recordCounter{a.Received.Records - b.Received.Records, a.Received.CiphertextBytes - b.Received.CiphertextBytes}}
}

func writeRecordArtifacts(dir, scenario string, cases []recordCaseObservation) {
	payloads := make([]map[string]any, 0, len(cases))
	for _, c := range cases {
		payloads = append(payloads, map[string]any{"caseId": c.CaseID, "direction": c.Direction, "applicationDataBytes": c.ApplicationDataBytes, "payloadSha256": c.PayloadSHA256})
	}
	writeRequired(dir, "payload-hash.json", map[string]any{"schemaVersion": "protocol-lab.payload-hash.v1", "scenarioId": scenario, "payloadGenerator": "repeated-octet", "payloadOctet": 90, "cases": payloads})
	if scenario == recordCoverageScenarioID {
		writeRequired(dir, "record-coverage.json", map[string]any{"schemaVersion": "protocol-lab.tls-record-coverage-proof.v1", "profileId": "plab-tls-record-coverage-v1", "allSixCasesComplete": len(cases) == 6, "cases": cases})
	}
}
func shapeFrom(c loadConfig) loadShape {
	s := loadShape{Connections: c.Connections, Concurrency: c.Concurrency, HandshakesPerConnection: c.HandshakesPerConnection, ApplicationDataBytes: c.ApplicationDataBytes, DurationSeconds: c.Duration.Seconds(), WarmupSeconds: c.Warmup.Seconds(), Repetition: c.Repetition}
	if c.TotalOperations > 0 {
		s.TotalOperations = &c.TotalOperations
	}
	return s
}
func normalizeMetrics(s phaseSummary) normalizedMetrics {
	d := s.DurationSeconds
	if d <= 0 {
		d = 1
	}
	return normalizedMetrics{HandshakesPerSecond: float64(s.CompletedOperations) / d, TLSHandshakeLatencyMeanMS: mean(s.TLSHandshakeLatencyMilliseconds), TLSHandshakeLatencyP50MS: percentile(s.TLSHandshakeLatencyMilliseconds, .5), TLSHandshakeLatencyP75MS: percentile(s.TLSHandshakeLatencyMilliseconds, .75), TLSHandshakeLatencyP90MS: percentile(s.TLSHandshakeLatencyMilliseconds, .9), TLSHandshakeLatencyP95MS: percentile(s.TLSHandshakeLatencyMilliseconds, .95), TLSHandshakeLatencyP99MS: percentile(s.TLSHandshakeLatencyMilliseconds, .99), ConnectionAndHandshakeLatencyMeanMS: mean(s.ConnectionAndHandshakeMilliseconds), BytesPerSecond: float64(s.TotalTransferredBytes) / d, TransferLatencyMeanMS: mean(s.TransferLatencyMilliseconds), TransferLatencyP50MS: percentile(s.TransferLatencyMilliseconds, .5), TransferLatencyP75MS: percentile(s.TransferLatencyMilliseconds, .75), TransferLatencyP90MS: percentile(s.TransferLatencyMilliseconds, .9), TransferLatencyP95MS: percentile(s.TransferLatencyMilliseconds, .95), TransferLatencyP99MS: percentile(s.TransferLatencyMilliseconds, .99), TotalTransferredBytes: s.TotalTransferredBytes, CompletedOperations: s.CompletedOperations, FailedOperations: s.FailedOperations, TimedOutOperations: s.TimedOutOperations}
}
func mean(v []float64) float64 {
	var t float64
	for _, x := range v {
		t += x
	}
	if len(v) == 0 {
		return 0
	}
	return t / float64(len(v))
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
func payloadHash(size int) string {
	v := make([]byte, size)
	for i := range v {
		v[i] = payloadOctet
	}
	return hash(v)
}
func writeAll(w io.Writer, v []byte) error {
	for len(v) > 0 {
		n, err := w.Write(v)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		v = v[n:]
	}
	return nil
}
func durationMS(v time.Duration) float64 { return float64(v) / float64(time.Millisecond) }
func hash(v []byte) string               { s := sha256.Sum256(v); return hex.EncodeToString(s[:]) }
func tlsVersionName(v uint16) string {
	if v == tls.VersionTLS13 {
		return "TLS1.3"
	}
	return fmt.Sprintf("0x%04x", v)
}
func loadRoots(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(data) {
		return nil, errors.New("root certificate PEM contained no certificate")
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
	u, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if u.Scheme != "tls" || u.Host == "" {
		return "", errors.New("TLS target must use tls://host:port")
	}
	return u.Host, nil
}
func envInt(name string) int { v, _ := strconv.Atoi(strings.TrimSpace(os.Getenv(name))); return v }
func operationTimeout() time.Duration {
	seconds := envInt("PLAB_REQUEST_TIMEOUT_SECONDS")
	if seconds == 0 {
		seconds = 5
	}
	return time.Duration(seconds) * time.Second
}
func verifySubstitution(variable, expected, label string) {
	if observed := strings.TrimSpace(os.Getenv(variable)); observed != "" && observed != expected {
		fatal(2, fmt.Errorf("%s substitution detected: expected %q observed %q", label, expected, observed))
	}
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	var n net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &n) && n.Timeout())
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
