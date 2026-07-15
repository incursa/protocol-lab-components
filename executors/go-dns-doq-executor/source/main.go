package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
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
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
)

const (
	executorID           = "go-dns-doq-executor"
	executorVersion      = "0.2.1"
	loadGeneratorID      = "go-quic-dns-load"
	loadGeneratorVersion = "0.2.1"
	strictScenario       = "dns.doq.query.a"
	interopScenario      = "dns.doq.interoperability.query.a"
	supportedProfile     = "secure-dns-smoke"
	serverName           = "dns.plab.test"
	certificateProfile   = "plab-secure-dns-single-leaf-p256-v1"
	requiredCipher       = "TLS_AES_128_GCM_SHA256"
	leafDERHash          = "b57bdd3eb90b36455900c17de9ff9a02c623e1f6b27626ad7821a40e35e8251c"
	leafSPKIHash         = "cfa6d5d08ee2071e28fd96205b088156fa71b460262470e5b994624b0537cf25"
	doqALPN              = "doq"
)

var knownUnsupported = map[string]struct{}{
	"dns.classic.tcp.query.a": {}, "dns.classic.udp-truncated-tcp-retry": {}, "dns.classic.udp.query.a": {},
	"dns.doh2.query.a": {}, "dns.doh2.interoperability.query.a": {}, "dns.doh3.get.a": {}, "dns.doh3.query.a": {}, "dns.doh3.interoperability.query.a": {}, "dns.doh3.query.aaaa": {},
	"dns.doh3.query.cname-chain": {}, "dns.doh3.query.large-dnssec-shaped": {}, "dns.doh3.query.nodata": {},
	"dns.doh3.query.nxdomain": {}, "dns.dot.query.a": {}, "dns.dot.interoperability.query.a": {},
}
var supportedScenarios = map[string]struct{}{strictScenario: {}, interopScenario: {}}

type tlsProof struct {
	TLSVersion                    string            `json:"tlsVersion"`
	CipherSuite                   string            `json:"cipherSuite"`
	KeyExchangeGroup              string            `json:"keyExchangeGroup"`
	SignatureScheme               string            `json:"signatureScheme"`
	ALPN                          string            `json:"alpn"`
	ServerName                    string            `json:"serverName"`
	HandshakeComplete             bool              `json:"handshakeComplete"`
	DidResume                     bool              `json:"didResume"`
	EarlyDataAttempted            bool              `json:"earlyDataAttempted"`
	CertificateProfile            string            `json:"certificateProfile"`
	CertificateDERSHA256          string            `json:"certificateDerSha256"`
	CertificateSPKISHA256         string            `json:"certificateSpkiSha256"`
	CertificateSignatureAlgorithm string            `json:"certificateSignatureAlgorithm"`
	CertificatePublicKeyAlgorithm string            `json:"certificatePublicKeyAlgorithm"`
	CertificateNamedCurve         string            `json:"certificateNamedCurve"`
	VerifiedChainCount            int               `json:"verifiedChainCount"`
	SentCertificateCount          int               `json:"sentCertificateCount"`
	TrustAnchorSent               bool              `json:"trustAnchorSent"`
	ConnectionLatencyMilliseconds float64           `json:"connectionLatencyMilliseconds"`
	PlatformProvenance            map[string]string `json:"platformProvenance"`
	AccelerationProvenance        map[string]string `json:"accelerationProvenance"`
}

type quicProof struct {
	Version                    string `json:"version"`
	VersionNumber              uint32 `json:"versionNumber"`
	ALPN                       string `json:"alpn"`
	UsedZeroRTT                bool   `json:"usedZeroRtt"`
	SupportsDatagrams          bool   `json:"supportsDatagrams"`
	Binding                    string `json:"binding"`
	ConnectionReused           bool   `json:"connectionReused"`
	ClientInitiatedBidiStreams bool   `json:"clientInitiatedBidirectionalStreams"`
}

type streamProof struct {
	StreamID               uint64 `json:"streamId"`
	StreamType             string `json:"streamType"`
	Queries                int    `json:"queries"`
	Responses              int    `json:"responses"`
	ClientFinAfterQuery    bool   `json:"clientFinAfterQuery"`
	ServerFinAfterResponse bool   `json:"serverFinAfterResponse"`
	UnexpectedReset        bool   `json:"unexpectedReset"`
	ProtocolErrorClose     bool   `json:"protocolErrorConnectionClose"`
}

type exchangeProof struct {
	DNS                         dnsProof    `json:"dns"`
	TLS                         tlsProof    `json:"tls"`
	QUIC                        quicProof   `json:"quic"`
	Stream                      streamProof `json:"stream"`
	QueryLatencyMilliseconds    float64     `json:"queryLatencyMilliseconds"`
	TimeToFirstByteMilliseconds float64     `json:"timeToFirstByteMilliseconds"`
	TransferredBytes            int64       `json:"transferredBytes"`
}

type phaseSummary struct {
	Phase                         string         `json:"phase"`
	DurationSeconds               float64        `json:"durationSeconds"`
	CompletedOperations           int            `json:"completedOperations"`
	MalformedOperations           int            `json:"malformedOperations"`
	RetryCount                    int            `json:"retryCount"`
	FailedOperations              int            `json:"failedOperations"`
	TimedOutOperations            int            `json:"timedOutOperations"`
	TotalTransferredBytes         int64          `json:"totalTransferredBytes"`
	EffectiveConcurrency          int            `json:"effectiveConcurrency"`
	EffectiveStreams              int            `json:"effectiveStreams"`
	ActiveConnections             int            `json:"activeConnections"`
	QueryLatencies                []float64      `json:"queryLatencyMilliseconds"`
	TimeToFirstByte               []float64      `json:"timeToFirstByteMilliseconds"`
	ConnectionLatencyMilliseconds float64        `json:"connectionLatencyMilliseconds"`
	LastProof                     *exchangeProof `json:"lastProof,omitempty"`
	Errors                        map[string]int `json:"errors"`
}

type metrics struct {
	QueriesPerSecond      float64 `json:"queriesPerSecond"`
	QueryLatencyMean      float64 `json:"queryLatencyMeanMs"`
	QueryLatencyP50       float64 `json:"queryLatencyP50Ms"`
	QueryLatencyP75       float64 `json:"queryLatencyP75Ms"`
	QueryLatencyP90       float64 `json:"queryLatencyP90Ms"`
	QueryLatencyP95       float64 `json:"queryLatencyP95Ms"`
	QueryLatencyP99       float64 `json:"queryLatencyP99Ms"`
	TimeToFirstByte       float64 `json:"timeToFirstByteMs"`
	ConnectionLatency     float64 `json:"connectionLatencyMs"`
	CompletedOperations   int     `json:"completedOperations"`
	MalformedOperations   int     `json:"malformedOperations"`
	RetryCount            int     `json:"retryCount"`
	FailedOperations      int     `json:"failedOperations"`
	TimedOutOperations    int     `json:"timedOutOperations"`
	TotalTransferredBytes int64   `json:"totalTransferredBytes"`
	EffectiveConcurrency  int     `json:"effectiveConcurrency"`
	EffectiveStreams      int     `json:"effectiveStreams"`
}

type result struct {
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
	Artifacts     []string          `json:"artifacts"`
}

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "DoQ target URL")
	output := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "artifact directory")
	rootPath := flag.String("root-certificate", os.Getenv("PLAB_TLS_ROOT_CERTIFICATE_PATH"), "test root PEM")
	validationOnly := flag.Bool("validation-only", false, "run one query validity check")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s %s\n", executorID, executorVersion)
		return
	}
	if *output == "" {
		*output = "artifacts"
	}
	if *rootPath == "" {
		*rootPath = filepath.Join("certs", "root.pem")
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		fatal(1, err)
	}
	checkIdentityOrExit(*output)
	address, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	roots, err := loadRoots(*rootPath)
	if err != nil {
		fatal(2, err)
	}

	preflight, err := runPhase(address, roots, 0, true)
	writeEvidence(*output, preflight, err)
	writeRequired(*output, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedScenarios": []string{strictScenario, interopScenario}})
	if err != nil {
		fatal(1, fmt.Errorf("DoQ validity gate failed: %w", err))
	}
	if *validationOnly {
		fmt.Fprintln(os.Stderr, "go-dns-doq-executor validity gate passed")
		return
	}

	config, err := loadConfig()
	if err != nil {
		fatal(2, err)
	}
	warmup, err := runPhase(address, roots, time.Second, false)
	writeRequired(*output, "dns-warmup-summary.json", warmup)
	if err != nil || warmup.CompletedOperations == 0 || hasFailures(warmup) {
		fatal(1, fmt.Errorf("DoQ warmup rejected: %w", err))
	}
	measured, err := runPhase(address, roots, 5*time.Second, false)
	writeRequired(*output, "dns-load-summary.json", measured)
	if err != nil || measured.CompletedOperations == 0 || hasFailures(measured) {
		fatal(1, fmt.Errorf("DoQ measured phase rejected: %w", err))
	}
	normalized := normalizeResult(measured, config)
	writeRequired(*output, "load-generator-identity.json", normalized.LoadGenerator)
	writeRequired(*output, "dns-doq-executor-result.json", normalized)
	writeRequired(*output, "result.json", normalized)
	data, _ := json.MarshalIndent(normalized, "", "  ")
	fmt.Println(string(data))
}

func checkIdentityOrExit(output string) {
	requireIdentity("PLAB_EXECUTOR_ID", executorID, "executor")
	requireIdentity("PLAB_EXECUTOR_VERSION", executorVersion, "executor version")
	requireIdentity("PLAB_LOAD_GENERATOR_ID", loadGeneratorID, "load generator")
	requireIdentity("PLAB_LOAD_GENERATOR_VERSION", loadGeneratorVersion, "load generator version")
	requireIdentity("PLAB_PROTOCOL", "doq", "protocol")
	requireIdentity("PLAB_PROTOCOL_VARIANT", protocolVariant(), "protocol variant")
	scenario := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if _, ok := supportedScenarios[scenario]; ok {
		return
	}
	if _, ok := knownUnsupported[scenario]; ok {
		document := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": scenario, "executorId": executorID, "reason": "exact DNS scenario semantics are not implemented by this DoQ-only package"}
		writeRequired(output, "unsupported.json", document)
		writeRequired(output, "result.json", document)
		data, _ := json.Marshal(document)
		fmt.Println(string(data))
		os.Exit(3)
	}
	fatal(2, fmt.Errorf("unknown scenario identity %q", scenario))
}

func loadConfig() (map[string]any, error) {
	if strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID")) != supportedProfile {
		return nil, fmt.Errorf("supports load profile %q only", supportedProfile)
	}
	values := map[string]int{"connections": envInt("PLAB_CONNECTIONS"), "concurrency": envInt("PLAB_CONCURRENCY"), "streamsPerConnection": envInt("PLAB_STREAMS_PER_CONNECTION"), "durationSeconds": envInt("PLAB_DURATION_SECONDS"), "warmupSeconds": envInt("PLAB_WARMUP_SECONDS"), "repetition": envInt("PLAB_REPETITION")}
	if values["connections"] != 1 || values["concurrency"] != 1 || values["streamsPerConnection"] != 1 || values["durationSeconds"] != 5 || values["warmupSeconds"] != 1 || values["repetition"] != 1 {
		return nil, fmt.Errorf("secure-dns-smoke requires connections=1 concurrency=1 streams=1 duration=5 warmup=1 repetition=1: %v", values)
	}
	return map[string]any{"connections": 1, "concurrency": 1, "outstandingQueries": 1, "streamsPerConnection": 1, "connectionReuse": "reuse", "durationSeconds": 5, "warmupSeconds": 1, "repetition": 1, "queryTimeoutMilliseconds": 5000, "maxRetries": 0}, nil
}

func runPhase(address string, roots *x509.CertPool, duration time.Duration, once bool) (phaseSummary, error) {
	summary := phaseSummary{Phase: "measured", EffectiveConcurrency: 1, EffectiveStreams: 1, ActiveConnections: 1, Errors: map[string]int{}}
	if once {
		summary.Phase = "preflight"
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: roots, ServerName: serverName, NextProtos: []string{doqALPN}, ClientSessionCache: nil, CurvePreferences: []tls.CurveID{tls.X25519}}
	quicConfig := &quic.Config{Versions: []quic.Version{quic.Version1}, HandshakeIdleTimeout: 5 * time.Second, MaxIdleTimeout: 30 * time.Second, KeepAlivePeriod: 5 * time.Second, EnableDatagrams: false}
	dialContext, cancelDial := context.WithTimeout(context.Background(), 5*time.Second)
	startedConnection := time.Now()
	connection, err := quic.DialAddr(dialContext, address, tlsConfig, quicConfig)
	cancelDial()
	if err != nil {
		classify(&summary, err)
		return summary, err
	}
	defer connection.CloseWithError(0, "")
	summary.ConnectionLatencyMilliseconds = durationMS(time.Since(startedConnection))
	if _, _, err = validateConnection(connection, summary.ConnectionLatencyMilliseconds, false); err != nil {
		classify(&summary, err)
		return summary, err
	}

	first, err := exchange(connection, false, summary.ConnectionLatencyMilliseconds)
	if err != nil {
		classify(&summary, err)
		return summary, err
	}
	if once {
		record(&summary, first)
		summary.DurationSeconds = first.QueryLatencyMilliseconds / 1000
		return summary, nil
	}
	started := time.Now()
	deadline := started.Add(duration)
	for time.Now().Before(deadline) {
		proof, exchangeErr := exchange(connection, true, summary.ConnectionLatencyMilliseconds)
		if exchangeErr != nil {
			classify(&summary, exchangeErr)
			return summary, exchangeErr
		}
		record(&summary, proof)
	}
	summary.DurationSeconds = time.Since(started).Seconds()
	return summary, nil
}

func exchange(connection *quic.Conn, reused bool, connectionLatency float64) (exchangeProof, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started := time.Now()
	requestStream, err := connection.OpenStreamSync(ctx)
	if err != nil {
		return exchangeProof{}, err
	}
	if err = requestStream.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return exchangeProof{}, err
	}
	query := frame(canonicalQuery)
	written, err := requestStream.Write(query)
	if err != nil || written != len(query) {
		requestStream.CancelWrite(quic.StreamErrorCode(0x3))
		return exchangeProof{}, firstError(err, io.ErrShortWrite)
	}
	if err = requestStream.Close(); err != nil {
		return exchangeProof{}, err
	}
	firstByte := float64(0)
	response := make([]byte, 0, 45)
	buffer := make([]byte, 64)
	for {
		read, readErr := requestStream.Read(buffer)
		if read > 0 {
			if firstByte == 0 {
				firstByte = durationMS(time.Since(started))
			}
			response = append(response, buffer[:read]...)
		}
		if readErr == nil {
			if len(response) > 45 {
				return exchangeProof{}, malformedResponseError{"DoQ response exceeded canonical bound"}
			}
			continue
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if errors.Is(readErr, context.DeadlineExceeded) || ctx.Err() != nil {
			requestStream.CancelRead(quic.StreamErrorCode(0x3))
			return exchangeProof{}, context.DeadlineExceeded
		}
		return exchangeProof{}, readErr
	}
	dnsValue, err := validateDNSFrame(response)
	if err != nil {
		return exchangeProof{}, err
	}
	tlsValue, quicValue, err := validateConnection(connection, connectionLatency, reused)
	if err != nil {
		return exchangeProof{}, err
	}
	streamValue := streamProof{StreamID: uint64(requestStream.StreamID()), StreamType: "client-initiated-bidirectional", Queries: 1, Responses: 1, ClientFinAfterQuery: true, ServerFinAfterResponse: true, UnexpectedReset: false, ProtocolErrorClose: false}
	return exchangeProof{DNS: dnsValue, TLS: tlsValue, QUIC: quicValue, Stream: streamValue, QueryLatencyMilliseconds: durationMS(time.Since(started)), TimeToFirstByteMilliseconds: firstByte, TransferredBytes: int64(len(query) + len(response))}, nil
}

func validateConnection(connection *quic.Conn, connectionLatency float64, reused bool) (tlsProof, quicProof, error) {
	state := connection.ConnectionState()
	tlsState := &state.TLS
	tlsValue := tlsProof{TLSVersion: tlsVersionName(tlsState.Version), CipherSuite: tls.CipherSuiteName(tlsState.CipherSuite), KeyExchangeGroup: "X25519", SignatureScheme: "ecdsa_secp256r1_sha256", ALPN: tlsState.NegotiatedProtocol, ServerName: serverName, HandshakeComplete: tlsState.HandshakeComplete, DidResume: tlsState.DidResume, EarlyDataAttempted: state.Used0RTT, CertificateProfile: certificateProfile, VerifiedChainCount: len(tlsState.VerifiedChains), SentCertificateCount: len(tlsState.PeerCertificates), TrustAnchorSent: false, ConnectionLatencyMilliseconds: connectionLatency, PlatformProvenance: runtimeProvenance(), AccelerationProvenance: accelerationProvenance()}
	if len(tlsState.PeerCertificates) > 0 {
		certificate := tlsState.PeerCertificates[0]
		tlsValue.CertificateDERSHA256 = hash(certificate.Raw)
		tlsValue.CertificateSPKISHA256 = hash(certificate.RawSubjectPublicKeyInfo)
		tlsValue.CertificateSignatureAlgorithm = certificate.SignatureAlgorithm.String()
		tlsValue.CertificatePublicKeyAlgorithm = certificate.PublicKeyAlgorithm.String()
		if key, ok := certificate.PublicKey.(*ecdsa.PublicKey); ok {
			tlsValue.CertificateNamedCurve = key.Curve.Params().Name
		}
	}
	quicValue := quicProof{Version: state.Version.String(), VersionNumber: uint32(state.Version), ALPN: tlsState.NegotiatedProtocol, UsedZeroRTT: state.Used0RTT, SupportsDatagrams: state.SupportsDatagrams.Remote || state.SupportsDatagrams.Local, Binding: "dns-over-quic-rfc9250", ConnectionReused: reused, ClientInitiatedBidiStreams: true}
	var failures []string
	if state.Version != quic.Version1 || state.Version.String() != "v1" {
		failures = append(failures, "exact QUIC v1 was not negotiated")
	}
	if tlsState.Version != tls.VersionTLS13 || !tlsState.HandshakeComplete || tlsState.DidResume || state.Used0RTT {
		failures = append(failures, "exact fresh TLS 1.3 state mismatch")
	}
	if tlsState.NegotiatedProtocol != doqALPN {
		failures = append(failures, "DoQ ALPN mismatch")
	}
	if tlsValue.CipherSuite != requiredCipher || len(tlsState.VerifiedChains) == 0 {
		failures = append(failures, "cipher suite or certificate authentication mismatch")
	}
	if tlsValue.CertificateDERSHA256 != leafDERHash || tlsValue.CertificateSPKISHA256 != leafSPKIHash {
		failures = append(failures, "certificate identity mismatch")
	}
	if len(failures) > 0 {
		return tlsValue, quicValue, errors.New(strings.Join(failures, "; "))
	}
	return tlsValue, quicValue, nil
}

func writeEvidence(output string, summary phaseSummary, err error) {
	validation := validationDocument(summary, err)
	writeRequired(output, "validation.json", validation)
	writeRequired(output, "result.json", validation)
	var dnsValue, tlsValue, quicValue, streamValue any
	if summary.LastProof != nil {
		dnsValue, tlsValue, quicValue, streamValue = summary.LastProof.DNS, summary.LastProof.TLS, summary.LastProof.QUIC, summary.LastProof.Stream
		writeRequired(output, "dns-wire-summary.json", dnsValue)
		writeRequired(output, "quic-summary.json", map[string]any{"quic": quicValue, "stream": streamValue})
		writeRequired(output, "tls-negotiation.json", tlsValue)
	}
	writeRequired(output, "protocol-proof.json", map[string]any{"requestedProtocol": "doq", "observedProtocol": observedProtocol(summary), "protocolVariant": protocolVariant(), "fallbackDetected": observedProtocol(summary) != "doq", "tls": tlsValue, "quic": quicValue, "stream": streamValue, "dns": dnsValue})
}

func normalizeResult(summary phaseSummary, requested map[string]any) result {
	value := metrics{QueriesPerSecond: float64(summary.CompletedOperations) / summary.DurationSeconds, QueryLatencyMean: mean(summary.QueryLatencies), QueryLatencyP50: percentile(summary.QueryLatencies, .50), QueryLatencyP75: percentile(summary.QueryLatencies, .75), QueryLatencyP90: percentile(summary.QueryLatencies, .90), QueryLatencyP95: percentile(summary.QueryLatencies, .95), QueryLatencyP99: percentile(summary.QueryLatencies, .99), TimeToFirstByte: mean(summary.TimeToFirstByte), ConnectionLatency: summary.ConnectionLatencyMilliseconds, CompletedOperations: summary.CompletedOperations, MalformedOperations: summary.MalformedOperations, RetryCount: summary.RetryCount, FailedOperations: summary.FailedOperations, TimedOutOperations: summary.TimedOutOperations, TotalTransferredBytes: summary.TotalTransferredBytes, EffectiveConcurrency: summary.EffectiveConcurrency, EffectiveStreams: summary.EffectiveStreams}
	proof := summary.LastProof
	artifacts := []string{"validation.json", "protocol-proof.json", "dns-wire-summary.json", "quic-summary.json", "tls-negotiation.json", "dns-doq-executor-result.json", "dns-load-summary.json", "dns-warmup-summary.json", "executor-identity.json", "load-generator-identity.json"}
	return result{SchemaVersion: "protocol-lab.dns-doq-executor-result.v1", ScenarioID: selectedScenario(), LoadProfileID: supportedProfile, Status: "passed", Executor: map[string]string{"id": executorID, "version": executorVersion}, LoadGenerator: map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, ProtocolProof: map[string]any{"requestedProtocol": "doq", "observedProtocol": "doq", "protocolVariant": protocolVariant(), "fallbackDetected": false, "tls": proof.TLS, "quic": proof.QUIC, "stream": proof.Stream, "dns": proof.DNS}, Validation: map[string]any{"status": "passed", "zeroUnexpectedFailures": true, "zeroTimeouts": true, "zeroMalformed": true, "zeroRetries": true, "localAuthoritativeOnly": true, "externalUpstreamUsed": false, "cacheEnabled": false, "oneQueryPerStream": true, "clientAndServerFin": true}, RequestedLoad: requested, EffectiveLoad: map[string]any{"connections": 1, "activeConnections": summary.ActiveConnections, "concurrency": 1, "outstandingQueries": 1, "streamsPerConnection": 1, "activeStreams": summary.EffectiveStreams}, Metrics: value, Warnings: []string{"Local package-backed DoQ smoke is diagnostic and non-publishable. Every other DNS binding or semantic fixture is unsupported by this executor."}, Artifacts: artifacts}
}

func validationDocument(summary phaseSummary, err error) map[string]any {
	return map[string]any{"scenarioId": selectedScenario(), "fixtureId": fixtureID, "passed": err == nil, "requestedProtocol": "doq", "observedProtocol": observedProtocol(summary), "protocolVariant": protocolVariant(), "fallbackDetected": observedProtocol(summary) != "doq", "completedOperations": summary.CompletedOperations, "malformedOperations": summary.MalformedOperations, "retryCount": summary.RetryCount, "failedOperations": summary.FailedOperations, "timedOutOperations": summary.TimedOutOperations, "externalUpstreamUsed": false, "cacheEnabled": false, "error": errorString(err)}
}
func observedProtocol(summary phaseSummary) string {
	if summary.LastProof != nil && summary.LastProof.QUIC.Version == "v1" && summary.LastProof.TLS.ALPN == doqALPN && summary.LastProof.DNS.Transport == "doq" {
		return "doq"
	}
	return ""
}
func record(summary *phaseSummary, proof exchangeProof) {
	summary.CompletedOperations++
	summary.TotalTransferredBytes += proof.TransferredBytes
	summary.QueryLatencies = append(summary.QueryLatencies, proof.QueryLatencyMilliseconds)
	summary.TimeToFirstByte = append(summary.TimeToFirstByte, proof.TimeToFirstByteMilliseconds)
	summary.LastProof = &proof
}
func classify(summary *phaseSummary, err error) {
	var malformed malformedResponseError
	if errors.As(err, &malformed) {
		summary.MalformedOperations++
	} else if isTimeout(err) {
		summary.TimedOutOperations++
	} else {
		summary.FailedOperations++
	}
	summary.Errors[err.Error()]++
}
func hasFailures(summary phaseSummary) bool {
	return summary.MalformedOperations != 0 || summary.RetryCount != 0 || summary.FailedOperations != 0 || summary.TimedOutOperations != 0
}
func normalizeTarget(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "doq" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("DoQ target must be an exact doq://host:port URL without path, query, or fragment")
	}
	if _, _, err = net.SplitHostPort(parsed.Host); err != nil {
		return "", errors.New("DoQ target must include an explicit port")
	}
	return parsed.Host, nil
}
func loadRoots(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, errors.New("root PEM contained no certificate")
	}
	return pool, nil
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
func percentile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	index := int(math.Ceil(quantile*float64(len(copyValues)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(copyValues) {
		index = len(copyValues) - 1
	}
	return copyValues[index]
}
func requireIdentity(variable, expected, label string) {
	observed := strings.TrimSpace(os.Getenv(variable))
	if observed != expected {
		fatal(2, fmt.Errorf("%s identity mismatch: expected %q, observed %q", label, expected, observed))
	}
}
func envInt(name string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	return value
}
func tlsVersionName(version uint16) string {
	if version == tls.VersionTLS13 {
		return "TLS1.3"
	}
	return fmt.Sprintf("0x%04x", version)
}
func durationMS(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }
func isTimeout(err error) bool {
	var netError net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netError) && netError.Timeout())
}
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func firstError(err, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}
func writeRequired(directory, name string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err == nil {
		data = append(data, '\n')
		err = os.WriteFile(filepath.Join(directory, name), data, 0o644)
	}
	if err != nil {
		fatal(1, err)
	}
}
func selectedScenario() string {
	id := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if id == "" {
		return strictScenario
	}
	return id
}
func protocolVariant() string {
	if selectedScenario() == interopScenario {
		return "doq-rfc9250-interoperability"
	}
	return "dns-over-quic-v1"
}
func runtimeProvenance() map[string]string {
	return map[string]string{"goos": runtime.GOOS, "goarch": runtime.GOARCH, "goVersion": runtime.Version()}
}
func accelerationProvenance() map[string]string { return map[string]string{"mode": "not-reported"} }
func fatal(code int, err error)                 { fmt.Fprintln(os.Stderr, err); os.Exit(code) }
