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
	"time"
)

const (
	executorID           = "go-dns-dot-executor"
	executorVersion      = "0.1.0"
	loadGeneratorID      = "go-dns-dot-load"
	loadGeneratorVersion = "0.1.0"
	supportedScenario    = "dns.dot.query.a"
	supportedProfile     = "secure-dns-smoke"
	serverName           = "dns.plab.test"
	alpn                 = "dot"
	fixtureID            = "dns.plab-test-a.canonical"
	certificateProfile   = "plab-secure-dns-single-leaf-p256-v1"
	requiredCipher       = "TLS_AES_128_GCM_SHA256"
	queryHash            = "c46b9fb76019b5a644d0884b17e816cb7c3076275d9468c27d180f70488eb8ec"
	responseHash         = "9d488461675ad5ab9f74c7b203861e1ad17521e413a407a25e6611012a595620"
	leafDERHash          = "b57bdd3eb90b36455900c17de9ff9a02c623e1f6b27626ad7821a40e35e8251c"
	leafSPKIHash         = "cfa6d5d08ee2071e28fd96205b088156fa71b460262470e5b994624b0537cf25"
)

var (
	canonicalQuery    = mustHex("00000000000100000000000004706c616204746573740000010001")
	canonicalResponse = mustHex("00008400000100010000000004706c616204746573740000010001c00c00010001000000000004c0000201")
	knownUnsupported  = map[string]struct{}{
		"dns.classic.tcp.query.a":             {},
		"dns.classic.udp-truncated-tcp-retry": {},
		"dns.classic.udp.query.a":             {},
		"dns.doh2.query.a":                    {},
		"dns.doh3.get.a":                      {},
		"dns.doh3.query.a":                    {},
		"dns.doh3.query.aaaa":                 {},
		"dns.doh3.query.cname-chain":          {},
		"dns.doh3.query.large-dnssec-shaped":  {},
		"dns.doh3.query.nodata":               {},
		"dns.doh3.query.nxdomain":             {},
		"dns.doq.query.a":                     {},
	}
)

type tlsProof struct {
	TLSVersion                    string  `json:"tlsVersion"`
	CipherSuite                   string  `json:"cipherSuite"`
	KeyExchangeGroup              string  `json:"keyExchangeGroup"`
	SignatureScheme               string  `json:"signatureScheme"`
	ALPN                          string  `json:"alpn"`
	ServerName                    string  `json:"serverName"`
	HandshakeComplete             bool    `json:"handshakeComplete"`
	DidResume                     bool    `json:"didResume"`
	EarlyDataAttempted            bool    `json:"earlyDataAttempted"`
	CertificateProfile            string  `json:"certificateProfile"`
	CertificateDERSHA256          string  `json:"certificateDerSha256"`
	CertificateSPKISHA256         string  `json:"certificateSpkiSha256"`
	CertificateSignatureAlgorithm string  `json:"certificateSignatureAlgorithm"`
	CertificatePublicKeyAlgorithm string  `json:"certificatePublicKeyAlgorithm"`
	CertificateNamedCurve         string  `json:"certificateNamedCurve"`
	VerifiedChainCount            int     `json:"verifiedChainCount"`
	SentCertificateCount          int     `json:"sentCertificateCount"`
	TrustAnchorSent               bool    `json:"trustAnchorSent"`
	ConnectionLatencyMilliseconds float64 `json:"connectionLatencyMilliseconds"`
}

type dnsProof struct {
	FixtureID                       string `json:"fixtureId"`
	Transport                       string `json:"transport"`
	QuestionName                    string `json:"questionName"`
	QuestionType                    string `json:"questionType"`
	QuestionClass                   string `json:"questionClass"`
	Answer                          string `json:"answer"`
	TTLSeconds                      int    `json:"ttlSeconds"`
	ResponseCode                    string `json:"responseCode"`
	AuthoritativeAnswer             bool   `json:"authoritativeAnswer"`
	RecursionDesired                bool   `json:"recursionDesired"`
	RecursionAvailable              bool   `json:"recursionAvailable"`
	ExternalUpstreamUsed            bool   `json:"externalUpstreamUsed"`
	CacheEnabled                    bool   `json:"cacheEnabled"`
	Framing                         string `json:"framing"`
	LengthPrefixOctets              int    `json:"lengthPrefixOctets"`
	ByteOrder                       string `json:"byteOrder"`
	RuntimeMessageID                uint16 `json:"runtimeMessageId"`
	ResponseMessageID               uint16 `json:"responseMessageId"`
	MessageIDCorrelated             bool   `json:"messageIdCorrelated"`
	MessageIDUniqueAmongOutstanding bool   `json:"messageIdUniqueAmongOutstanding"`
	CanonicalHashNormalization      string `json:"canonicalHashNormalization"`
	QueryLengthBytes                int    `json:"queryLengthBytes"`
	QueryNormalizedSHA256           string `json:"queryNormalizedSha256"`
	ResponseLengthBytes             int    `json:"responseLengthBytes"`
	ResponseNormalizedSHA256        string `json:"responseNormalizedSha256"`
	RequestFramedHex                string `json:"requestFramedHex"`
	ResponseFramedHex               string `json:"responseFramedHex"`
}

type phaseSummary struct {
	Phase                    string         `json:"phase"`
	DurationSeconds          float64        `json:"durationSeconds"`
	CompletedOperations      int            `json:"completedOperations"`
	MalformedOperations      int            `json:"malformedOperations"`
	RetryCount               int            `json:"retryCount"`
	FailedOperations         int            `json:"failedOperations"`
	TimedOutOperations       int            `json:"timedOutOperations"`
	TotalTransferredBytes    int64          `json:"totalTransferredBytes"`
	EffectiveConcurrency     int            `json:"effectiveConcurrency"`
	QueryLatencyMilliseconds []float64      `json:"queryLatencyMilliseconds"`
	LastDNSProof             *dnsProof      `json:"lastDnsProof,omitempty"`
	TLSProof                 tlsProof       `json:"tlsProof"`
	Errors                   map[string]int `json:"errors"`
}

type metrics struct {
	QueriesPerSecond      float64 `json:"queriesPerSecond"`
	QueryLatencyMean      float64 `json:"queryLatencyMeanMs"`
	QueryLatencyP50       float64 `json:"queryLatencyP50Ms"`
	QueryLatencyP75       float64 `json:"queryLatencyP75Ms"`
	QueryLatencyP90       float64 `json:"queryLatencyP90Ms"`
	QueryLatencyP95       float64 `json:"queryLatencyP95Ms"`
	QueryLatencyP99       float64 `json:"queryLatencyP99Ms"`
	ConnectionLatency     float64 `json:"connectionLatencyMs"`
	CompletedOperations   int     `json:"completedOperations"`
	MalformedOperations   int     `json:"malformedOperations"`
	RetryCount            int     `json:"retryCount"`
	FailedOperations      int     `json:"failedOperations"`
	TimedOutOperations    int     `json:"timedOutOperations"`
	TotalTransferredBytes int64   `json:"totalTransferredBytes"`
	EffectiveConcurrency  int     `json:"effectiveConcurrency"`
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
}

type malformedResponseError struct{ reason string }

func (err malformedResponseError) Error() string { return err.reason }

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "DoT target address or tls:// URL")
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
	validation := validationDocument(preflight, err)
	writeRequired(*output, "validation.json", validation)
	writeRequired(*output, "result.json", validation)
	if preflight.LastDNSProof != nil {
		writeRequired(*output, "dns-wire-summary.json", preflight.LastDNSProof)
	}
	writeRequired(*output, "tls-negotiation.json", preflight.TLSProof)
	writeRequired(*output, "protocol-proof.json", map[string]any{"requestedProtocol": "dot", "observedProtocol": "dot", "protocolVariant": "dot-tls1.3-tcp", "fallbackDetected": false, "tls": preflight.TLSProof, "dns": preflight.LastDNSProof})
	writeRequired(*output, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedScenarios": []string{supportedScenario}})
	if err != nil {
		fatal(1, fmt.Errorf("DoT validity gate failed: %w", err))
	}
	if *validationOnly {
		fmt.Fprintln(os.Stderr, "go-dns-dot-executor validity gate passed")
		return
	}

	config, err := loadConfig()
	if err != nil {
		fatal(2, err)
	}
	warmup, err := runPhase(address, roots, time.Second, false)
	writeRequired(*output, "dns-warmup-summary.json", warmup)
	if err != nil || warmup.CompletedOperations == 0 || hasFailures(warmup) {
		fatal(1, fmt.Errorf("DoT warmup rejected: %w", err))
	}
	measured, err := runPhase(address, roots, 5*time.Second, false)
	writeRequired(*output, "dns-load-summary.json", measured)
	if err != nil || measured.CompletedOperations == 0 || hasFailures(measured) {
		fatal(1, fmt.Errorf("DoT measured phase rejected: %w", err))
	}
	normalized := normalizeResult(measured, config)
	writeRequired(*output, "load-generator-identity.json", normalized.LoadGenerator)
	writeRequired(*output, "dns-dot-executor-result.json", normalized)
	writeRequired(*output, "result.json", normalized)
	data, _ := json.MarshalIndent(normalized, "", "  ")
	fmt.Println(string(data))
}

func checkIdentityOrExit(output string) {
	verifySubstitution("PLAB_EXECUTOR_ID", executorID, "executor")
	verifySubstitution("PLAB_EXECUTOR_VERSION", executorVersion, "executor version")
	verifySubstitution("PLAB_LOAD_GENERATOR_ID", loadGeneratorID, "load generator")
	verifySubstitution("PLAB_LOAD_GENERATOR_VERSION", loadGeneratorVersion, "load generator version")
	verifySubstitution("PLAB_PROTOCOL", "dot", "protocol")
	verifySubstitution("PLAB_PROTOCOL_VARIANT", "dot-tls1.3-tcp", "protocol variant")
	scenario := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if scenario == "" || scenario == supportedScenario {
		return
	}
	if _, ok := knownUnsupported[scenario]; ok {
		doc := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": scenario, "executorId": executorID, "reason": "exact DNS scenario semantics are not implemented by this DoT-only package"}
		writeRequired(output, "unsupported.json", doc)
		writeRequired(output, "result.json", doc)
		data, _ := json.Marshal(doc)
		fmt.Println(string(data))
		os.Exit(3)
	}
	fatal(2, fmt.Errorf("unknown scenario identity %q", scenario))
}

func loadConfig() (map[string]any, error) {
	if strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")) != supportedScenario {
		return nil, errors.New("exact supported scenario identity is required")
	}
	if strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID")) != supportedProfile {
		return nil, fmt.Errorf("supports load profile %q only", supportedProfile)
	}
	values := map[string]int{"connections": envInt("PLAB_CONNECTIONS"), "concurrency": envInt("PLAB_CONCURRENCY"), "durationSeconds": envInt("PLAB_DURATION_SECONDS"), "warmupSeconds": envInt("PLAB_WARMUP_SECONDS"), "repetition": envInt("PLAB_REPETITION")}
	if values["connections"] != 1 || values["concurrency"] != 1 || values["durationSeconds"] != 5 || values["warmupSeconds"] != 1 || values["repetition"] != 1 {
		return nil, fmt.Errorf("secure-dns-smoke requires connections=1 concurrency=1 duration=5 warmup=1 repetition=1: %v", values)
	}
	return map[string]any{"connections": 1, "concurrency": 1, "outstandingQueries": 1, "connectionReuse": "reuse", "durationSeconds": 5, "warmupSeconds": 1, "repetition": 1, "queryTimeoutMilliseconds": 5000, "maxRetries": 0}, nil
}

func runPhase(address string, roots *x509.CertPool, duration time.Duration, once bool) (phaseSummary, error) {
	summary := phaseSummary{Phase: "measured", EffectiveConcurrency: 1, Errors: map[string]int{}}
	if once {
		summary.Phase = "preflight"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, proof, err := connect(ctx, address, roots)
	summary.TLSProof = proof
	if err != nil {
		summary.FailedOperations++
		return summary, err
	}
	defer conn.Close()
	started := time.Now()
	deadline := started.Add(duration)
	var nextID uint16 = 1
	for once || time.Now().Before(deadline) {
		if nextID == 0 {
			nextID = 1
		}
		queryStarted := time.Now()
		proof, transferred, err := exchange(conn, nextID)
		summary.TotalTransferredBytes += transferred
		if err != nil {
			var malformed malformedResponseError
			if errors.As(err, &malformed) {
				summary.MalformedOperations++
			} else if isTimeout(err) {
				summary.TimedOutOperations++
			} else {
				summary.FailedOperations++
			}
			summary.Errors[err.Error()]++
			return summary, err
		}
		summary.CompletedOperations++
		summary.QueryLatencyMilliseconds = append(summary.QueryLatencyMilliseconds, durationMS(time.Since(queryStarted)))
		summary.LastDNSProof = &proof
		if once {
			break
		}
		nextID++
	}
	summary.DurationSeconds = time.Since(started).Seconds()
	return summary, nil
}

func connect(ctx context.Context, address string, roots *x509.CertPool) (*tls.Conn, tlsProof, error) {
	started := time.Now()
	raw, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, tlsProof{}, err
	}
	conn := tls.Client(raw, &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: roots, ServerName: serverName, NextProtos: []string{alpn}, ClientSessionCache: nil, CurvePreferences: []tls.CurveID{tls.X25519}})
	if err := conn.HandshakeContext(ctx); err != nil {
		raw.Close()
		return nil, tlsProof{}, err
	}
	proof, err := validateTLS(conn.ConnectionState())
	proof.ConnectionLatencyMilliseconds = durationMS(time.Since(started))
	if err != nil {
		conn.Close()
		return nil, proof, err
	}
	return conn, proof, nil
}

func validateTLS(state tls.ConnectionState) (tlsProof, error) {
	proof := tlsProof{TLSVersion: tlsVersionName(state.Version), CipherSuite: tls.CipherSuiteName(state.CipherSuite), ALPN: state.NegotiatedProtocol, ServerName: serverName, HandshakeComplete: state.HandshakeComplete, DidResume: state.DidResume, EarlyDataAttempted: false, CertificateProfile: certificateProfile, VerifiedChainCount: len(state.VerifiedChains)}
	proof.KeyExchangeGroup = "X25519"
	proof.SignatureScheme = "ecdsa_secp256r1_sha256"
	proof.SentCertificateCount = len(state.PeerCertificates)
	proof.TrustAnchorSent = false
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		proof.CertificateDERSHA256 = hash(cert.Raw)
		proof.CertificateSPKISHA256 = hash(cert.RawSubjectPublicKeyInfo)
		proof.CertificateSignatureAlgorithm = cert.SignatureAlgorithm.String()
		proof.CertificatePublicKeyAlgorithm = cert.PublicKeyAlgorithm.String()
		if key, ok := cert.PublicKey.(*ecdsa.PublicKey); ok {
			proof.CertificateNamedCurve = key.Curve.Params().Name
		}
	}
	var failures []string
	if state.Version != tls.VersionTLS13 {
		failures = append(failures, "exact TLS 1.3 was not negotiated")
	}
	if !state.HandshakeComplete {
		failures = append(failures, "TLS handshake incomplete")
	}
	if state.DidResume {
		failures = append(failures, "session resumption detected")
	}
	if state.NegotiatedProtocol != alpn {
		failures = append(failures, "ALPN mismatch")
	}
	if proof.CipherSuite != requiredCipher {
		failures = append(failures, "cipher-suite mismatch")
	}
	if len(state.VerifiedChains) == 0 {
		failures = append(failures, "certificate chain not authenticated")
	}
	if proof.CertificateDERSHA256 != leafDERHash {
		failures = append(failures, "leaf DER hash mismatch")
	}
	if proof.CertificateSPKISHA256 != leafSPKIHash {
		failures = append(failures, "leaf SPKI hash mismatch")
	}
	if len(failures) > 0 {
		return proof, errors.New(strings.Join(failures, "; "))
	}
	return proof, nil
}

func exchange(conn net.Conn, id uint16) (dnsProof, int64, error) {
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return dnsProof{}, 0, err
	}
	query := append([]byte(nil), canonicalQuery...)
	binary.BigEndian.PutUint16(query[:2], id)
	framed := frame(query)
	if _, err := conn.Write(framed); err != nil {
		return dnsProof{}, 0, err
	}
	prefix := make([]byte, 2)
	if _, err := io.ReadFull(conn, prefix); err != nil {
		return dnsProof{}, int64(len(framed)), err
	}
	length := int(binary.BigEndian.Uint16(prefix))
	if length <= 0 || length > 65535 {
		return dnsProof{}, int64(len(framed) + 2), malformedResponseError{"malformed DoT response length"}
	}
	response := make([]byte, length)
	if _, err := io.ReadFull(conn, response); err != nil {
		return dnsProof{}, int64(len(framed) + 2), err
	}
	transferred := int64(len(framed) + 2 + len(response))
	if len(response) != len(canonicalResponse) {
		return dnsProof{}, transferred, malformedResponseError{fmt.Sprintf("response length mismatch: %d", len(response))}
	}
	responseID := binary.BigEndian.Uint16(response[:2])
	if responseID != id {
		return dnsProof{}, transferred, malformedResponseError{"response message ID did not correlate"}
	}
	normalized := append([]byte(nil), response...)
	binary.BigEndian.PutUint16(normalized[:2], 0)
	if hash(normalized) != responseHash || !equal(normalized, canonicalResponse) {
		return dnsProof{}, transferred, malformedResponseError{"canonical response content/hash mismatch"}
	}
	normalizedQuery := append([]byte(nil), query...)
	binary.BigEndian.PutUint16(normalizedQuery[:2], 0)
	proof := dnsProof{FixtureID: fixtureID, Transport: "dot", QuestionName: "plab.test.", QuestionType: "A", QuestionClass: "IN", Answer: "192.0.2.1", TTLSeconds: 0, ResponseCode: "NOERROR", AuthoritativeAnswer: true, RecursionDesired: false, RecursionAvailable: false, ExternalUpstreamUsed: false, CacheEnabled: false, Framing: "two-octet-network-order-length-prefix", LengthPrefixOctets: 2, ByteOrder: "network", RuntimeMessageID: id, ResponseMessageID: responseID, MessageIDCorrelated: true, MessageIDUniqueAmongOutstanding: true, CanonicalHashNormalization: "set-message-id-to-zero", QueryLengthBytes: len(query), QueryNormalizedSHA256: hash(normalizedQuery), ResponseLengthBytes: len(response), ResponseNormalizedSHA256: hash(normalized), RequestFramedHex: hex.EncodeToString(framed), ResponseFramedHex: hex.EncodeToString(append(prefix, response...))}
	if proof.QueryNormalizedSHA256 != queryHash {
		return dnsProof{}, transferred, errors.New("canonical query content/hash mismatch")
	}
	return proof, transferred, nil
}

func normalizeResult(summary phaseSummary, requested map[string]any) result {
	m := metrics{QueriesPerSecond: float64(summary.CompletedOperations) / summary.DurationSeconds, QueryLatencyMean: mean(summary.QueryLatencyMilliseconds), QueryLatencyP50: percentile(summary.QueryLatencyMilliseconds, .50), QueryLatencyP75: percentile(summary.QueryLatencyMilliseconds, .75), QueryLatencyP90: percentile(summary.QueryLatencyMilliseconds, .90), QueryLatencyP95: percentile(summary.QueryLatencyMilliseconds, .95), QueryLatencyP99: percentile(summary.QueryLatencyMilliseconds, .99), ConnectionLatency: summary.TLSProof.ConnectionLatencyMilliseconds, CompletedOperations: summary.CompletedOperations, MalformedOperations: summary.MalformedOperations, RetryCount: summary.RetryCount, FailedOperations: summary.FailedOperations, TimedOutOperations: summary.TimedOutOperations, TotalTransferredBytes: summary.TotalTransferredBytes, EffectiveConcurrency: summary.EffectiveConcurrency}
	return result{SchemaVersion: "protocol-lab.dns-dot-executor-result.v1", ScenarioID: supportedScenario, LoadProfileID: supportedProfile, Status: "passed", Executor: map[string]string{"id": executorID, "version": executorVersion}, LoadGenerator: map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, ProtocolProof: map[string]any{"requestedProtocol": "dot", "observedProtocol": "dot", "protocolVariant": "dot-tls1.3-tcp", "fallbackDetected": false, "tls": summary.TLSProof, "dns": summary.LastDNSProof}, Validation: map[string]any{"status": "passed", "zeroUnexpectedFailures": true, "zeroTimeouts": true, "zeroMalformed": true, "zeroRetries": true}, RequestedLoad: requested, EffectiveLoad: map[string]any{"connections": 1, "activeConnections": 1, "concurrency": 1, "outstandingQueries": 1}, Metrics: m, Warnings: []string{"Local package-backed DoT smoke is diagnostic and non-publishable. All other secure DNS bindings and semantic fixtures are unsupported by this executor."}}
}

func validationDocument(summary phaseSummary, err error) map[string]any {
	return map[string]any{"scenarioId": supportedScenario, "fixtureId": fixtureID, "passed": err == nil, "requestedProtocol": "dot", "observedProtocol": func() string {
		if summary.TLSProof.ALPN == alpn {
			return "dot"
		}
		return ""
	}(), "fallbackDetected": summary.TLSProof.ALPN != alpn, "completedOperations": summary.CompletedOperations, "malformedOperations": summary.MalformedOperations, "retryCount": summary.RetryCount, "failedOperations": summary.FailedOperations, "timedOutOperations": summary.TimedOutOperations, "error": errorString(err)}
}
func hasFailures(s phaseSummary) bool {
	return s.MalformedOperations != 0 || s.RetryCount != 0 || s.FailedOperations != 0 || s.TimedOutOperations != 0
}
func frame(message []byte) []byte {
	result := make([]byte, 2+len(message))
	binary.BigEndian.PutUint16(result[:2], uint16(len(message)))
	copy(result[2:], message)
	return result
}
func normalizeTarget(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("target address required")
	}
	if !strings.Contains(value, "://") {
		_, _, err := net.SplitHostPort(value)
		return value, err
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "tls" || parsed.Host == "" {
		return "", errors.New("DoT target must use tls://host:port")
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
func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var total float64
	for _, n := range v {
		total += n
	}
	return total / float64(len(v))
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
func envInt(name string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	return value
}
func verifySubstitution(variable, expected, label string) {
	if observed := strings.TrimSpace(os.Getenv(variable)); observed != "" && observed != expected {
		fatal(2, fmt.Errorf("%s substitution detected: expected %q, observed %q", label, expected, observed))
	}
}
func tlsVersionName(v uint16) string {
	if v == tls.VersionTLS13 {
		return "TLS1.3"
	}
	return fmt.Sprintf("0x%04x", v)
}
func hash(v []byte) string { sum := sha256.Sum256(v); return hex.EncodeToString(sum[:]) }
func mustHex(v string) []byte {
	b, err := hex.DecodeString(v)
	if err != nil {
		panic(err)
	}
	return b
}
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
func durationMS(v time.Duration) float64 { return float64(v) / float64(time.Millisecond) }
func isTimeout(err error) bool {
	var netErr net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout())
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
