package main

import (
	"bytes"
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
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	executorID           = "go-dns-doh2-executor"
	executorVersion      = "0.2.2"
	loadGeneratorID      = "go-dns-doh2-load"
	loadGeneratorVersion = "0.2.2"
	strictScenario       = "dns.doh2.query.a"
	interopScenario      = "dns.doh2.interoperability.query.a"
	supportedProfile     = "secure-dns-smoke"
	serverName           = "dns.plab.test"
	fixtureID            = "dns.plab-test-a.canonical"
	mediaType            = "application/dns-message"
	certificateProfile   = "plab-secure-dns-single-leaf-p256-v1"
	strictTLSProfile     = "plab-secure-dns-tls13-v1"
	interopTLSProfile    = "plab-secure-dns-interoperability-v2"
	interopCertProfile   = "plab-secure-dns-interoperability-leaf-v2"
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
		"dns.classic.tcp.query.a": {}, "dns.classic.udp-truncated-tcp-retry": {}, "dns.classic.udp.query.a": {},
		"dns.doh3.get.a": {}, "dns.doh3.query.a": {}, "dns.doh3.query.aaaa": {}, "dns.doh3.query.cname-chain": {},
		"dns.doh3.query.large-dnssec-shaped": {}, "dns.doh3.query.nodata": {}, "dns.doh3.query.nxdomain": {},
		"dns.doh3.interoperability.query.a": {}, "dns.doq.query.a": {}, "dns.doq.interoperability.query.a": {},
		"dns.dot.query.a": {}, "dns.dot.interoperability.query.a": {},
	}
	supportedScenarios = map[string]struct{}{strictScenario: {}, interopScenario: {}}
)

type tlsProof struct {
	TLSProfileID                  string            `json:"tlsProfileId"`
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

type dnsProof struct {
	FixtureID                    string `json:"fixtureId"`
	Transport                    string `json:"transport"`
	QuestionName                 string `json:"questionName"`
	QuestionType                 string `json:"questionType"`
	QuestionClass                string `json:"questionClass"`
	Answer                       string `json:"answer"`
	TTLSeconds                   int    `json:"ttlSeconds"`
	ResponseCode                 string `json:"responseCode"`
	AuthoritativeAnswer          bool   `json:"authoritativeAnswer"`
	RecursionDesired             bool   `json:"recursionDesired"`
	RecursionAvailable           bool   `json:"recursionAvailable"`
	ExternalUpstreamUsed         bool   `json:"externalUpstreamUsed"`
	CacheEnabled                 bool   `json:"cacheEnabled"`
	Framing                      string `json:"framing"`
	AuthorityMode                string `json:"authorityMode"`
	RequestMessageID             uint16 `json:"requestMessageId"`
	RuntimeMessageID             uint16 `json:"runtimeMessageId"`
	ResponseMessageID            uint16 `json:"responseMessageId"`
	CanonicalHashNormalization   string `json:"canonicalHashNormalization"`
	QueryLengthBytes             int    `json:"queryLengthBytes"`
	QueryNormalizedSHA256        string `json:"queryNormalizedSha256"`
	ResponseRawLengthBytes       int    `json:"responseRawLengthBytes"`
	ResponseCanonicalLengthBytes int    `json:"responseCanonicalLengthBytes"`
	ResponseLengthBytes          int    `json:"responseLengthBytes"`
	ResponseNormalizedSHA256     string `json:"responseNormalizedSha256"`
	ResponseCompressionDiffered  bool   `json:"responseCompressionDiffered"`
	ResponseCanonicalized        bool   `json:"responseCanonicalized"`
	RawResponseEqualityRequired  bool   `json:"rawResponseEqualityRequired"`
}

type httpProof struct {
	Version               string `json:"httpVersion"`
	Method                string `json:"method"`
	Scheme                string `json:"scheme"`
	Authority             string `json:"authority"`
	Path                  string `json:"path"`
	StatusCode            int    `json:"responseStatus"`
	RequestAccept         string `json:"requestAccept"`
	RequestContentType    string `json:"requestContentType"`
	RequestCacheControl   string `json:"requestCacheControl"`
	ResponseContentType   string `json:"responseContentType"`
	ResponseCacheControl  string `json:"responseCacheControl"`
	BodyFraming           string `json:"bodyFraming"`
	ConnectionReused      bool   `json:"connectionReused"`
	HTTP1FallbackDetected bool   `json:"http1FallbackDetected"`
}

type exchangeProof struct {
	DNS                           dnsProof  `json:"dns"`
	HTTP                          httpProof `json:"http"`
	TLS                           tlsProof  `json:"tls"`
	QueryLatencyMilliseconds      float64   `json:"queryLatencyMilliseconds"`
	TimeToFirstByteMilliseconds   float64   `json:"timeToFirstByteMilliseconds"`
	TransferredBytes              int64     `json:"transferredBytes"`
	ConnectionLatencyMilliseconds float64   `json:"connectionLatencyMilliseconds"`
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
type malformedResponseError struct{ reason string }

func (err malformedResponseError) Error() string { return err.reason }

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "DoH2 target HTTPS base URL")
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
	endpoint, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	roots, err := loadRoots(*rootPath)
	if err != nil {
		fatal(2, err)
	}

	preflight, err := runPhase(endpoint, roots, 0, true)
	writeEvidence(*output, preflight, err)
	writeRequired(*output, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedScenarios": []string{strictScenario, interopScenario}})
	if err != nil {
		fatal(1, fmt.Errorf("DoH2 validity gate failed: %w", err))
	}
	if *validationOnly {
		fmt.Fprintln(os.Stderr, "go-dns-doh2-executor validity gate passed")
		return
	}

	config, err := loadConfig()
	if err != nil {
		fatal(2, err)
	}
	warmup, err := runPhase(endpoint, roots, time.Second, false)
	writeRequired(*output, "dns-warmup-summary.json", warmup)
	if err != nil || warmup.CompletedOperations == 0 || hasFailures(warmup) {
		fatal(1, fmt.Errorf("DoH2 warmup rejected: %w", err))
	}
	measured, err := runPhase(endpoint, roots, 5*time.Second, false)
	writeRequired(*output, "dns-load-summary.json", measured)
	if err != nil || measured.CompletedOperations == 0 || hasFailures(measured) {
		fatal(1, fmt.Errorf("DoH2 measured phase rejected: %w", err))
	}
	normalized := normalizeResult(measured, config)
	writeRequired(*output, "load-generator-identity.json", normalized.LoadGenerator)
	writeRequired(*output, "dns-doh2-executor-result.json", normalized)
	writeRequired(*output, "result.json", normalized)
	data, _ := json.MarshalIndent(normalized, "", "  ")
	fmt.Println(string(data))
}

func checkIdentityOrExit(output string) {
	verifySubstitution("PLAB_EXECUTOR_ID", executorID, "executor")
	verifySubstitution("PLAB_EXECUTOR_VERSION", executorVersion, "executor version")
	verifySubstitution("PLAB_LOAD_GENERATOR_ID", loadGeneratorID, "load generator")
	verifySubstitution("PLAB_LOAD_GENERATOR_VERSION", loadGeneratorVersion, "load generator version")
	verifySubstitution("PLAB_PROTOCOL", "doh2", "protocol")
	verifySubstitution("PLAB_PROTOCOL_VARIANT", protocolVariant(), "protocol variant")
	scenario := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if scenario == "" {
		scenario = strictScenario
	}
	if _, ok := supportedScenarios[scenario]; ok {
		return
	}
	if _, ok := knownUnsupported[scenario]; ok {
		doc := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": scenario, "executorId": executorID, "reason": "exact DNS scenario semantics are not implemented by this DoH2-only package"}
		writeRequired(output, "unsupported.json", doc)
		writeRequired(output, "result.json", doc)
		data, _ := json.Marshal(doc)
		fmt.Println(string(data))
		os.Exit(3)
	}
	fatal(2, fmt.Errorf("unknown scenario identity %q", scenario))
}

func loadConfig() (map[string]any, error) {
	if _, ok := supportedScenarios[selectedScenario()]; !ok {
		return nil, errors.New("exact supported scenario identity is required")
	}
	if strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID")) != supportedProfile {
		return nil, fmt.Errorf("supports load profile %q only", supportedProfile)
	}
	values := map[string]int{"connections": envInt("PLAB_CONNECTIONS"), "concurrency": envInt("PLAB_CONCURRENCY"), "streamsPerConnection": envInt("PLAB_STREAMS_PER_CONNECTION"), "durationSeconds": envInt("PLAB_DURATION_SECONDS"), "warmupSeconds": envInt("PLAB_WARMUP_SECONDS"), "repetition": envInt("PLAB_REPETITION")}
	if values["connections"] != 1 || values["concurrency"] != 1 || values["streamsPerConnection"] != 1 || values["durationSeconds"] != 5 || values["warmupSeconds"] != 1 || values["repetition"] != 1 {
		return nil, fmt.Errorf("secure-dns-smoke requires connections=1 concurrency=1 duration=5 warmup=1 repetition=1: %v", values)
	}
	return map[string]any{"connections": 1, "concurrency": 1, "outstandingQueries": 1, "streamsPerConnection": 1, "connectionReuse": "reuse", "durationSeconds": 5, "warmupSeconds": 1, "repetition": 1, "queryTimeoutMilliseconds": 5000, "maxRetries": 0}, nil
}

func runPhase(endpoint string, roots *x509.CertPool, duration time.Duration, once bool) (phaseSummary, error) {
	summary := phaseSummary{Phase: "measured", EffectiveConcurrency: 1, EffectiveStreams: 1, ActiveConnections: 1, Errors: map[string]int{}}
	if once {
		summary.Phase = "preflight"
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: roots, ServerName: serverName, NextProtos: []string{"h2"}, ClientSessionCache: nil, CurvePreferences: []tls.CurveID{tls.X25519}}, ForceAttemptHTTP2: true, DisableKeepAlives: false, MaxIdleConns: 1, MaxIdleConnsPerHost: 1, MaxConnsPerHost: 1, IdleConnTimeout: 30 * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	// Establish and validate the single connection before the measured window.
	first, err := exchange(client, endpoint, false)
	summary.ConnectionLatencyMilliseconds = first.ConnectionLatencyMilliseconds
	if err != nil {
		classify(&summary, err)
		return summary, err
	}
	first.TLS.ConnectionLatencyMilliseconds = summary.ConnectionLatencyMilliseconds
	if once {
		record(&summary, first)
		summary.DurationSeconds = first.QueryLatencyMilliseconds / 1000
		return summary, nil
	}
	started := time.Now()
	deadline := started.Add(duration)
	for time.Now().Before(deadline) {
		proof, err := exchange(client, endpoint, true)
		if err != nil {
			classify(&summary, err)
			return summary, err
		}
		record(&summary, proof)
	}
	summary.DurationSeconds = time.Since(started).Seconds()
	return summary, nil
}

func exchange(client *http.Client, endpoint string, reused bool) (exchangeProof, error) {
	started := time.Now()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(canonicalQuery))
	if err != nil {
		return exchangeProof{}, err
	}
	request.Host = serverName
	request.Header.Set("Accept", mediaType)
	request.Header.Set("Content-Type", mediaType)
	request.Header.Set("Cache-Control", "no-cache")
	connectionLatency := float64(0)
	firstByteLatency := float64(0)
	observedReused := false
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			observedReused = info.Reused
			if !info.Reused {
				connectionLatency = durationMS(time.Since(started))
			}
		},
		GotFirstResponseByte: func() { firstByteLatency = durationMS(time.Since(started)) },
	}
	request = request.WithContext(httptrace.WithClientTrace(request.Context(), trace))
	response, err := client.Do(request)
	if err != nil {
		return exchangeProof{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 65536))
	if err != nil {
		return exchangeProof{}, err
	}
	tlsValue, tlsErr := validateTLS(response.TLS)
	if tlsErr != nil {
		return exchangeProof{}, tlsErr
	}
	httpValue := httpProof{Version: response.Proto, Method: http.MethodPost, Scheme: "https", Authority: serverName, Path: "/dns-query", StatusCode: response.StatusCode, RequestAccept: mediaType, RequestContentType: mediaType, RequestCacheControl: "no-cache", ResponseContentType: response.Header.Get("Content-Type"), ResponseCacheControl: response.Header.Get("Cache-Control"), BodyFraming: "bare-dns-message-body", ConnectionReused: observedReused, HTTP1FallbackDetected: response.ProtoMajor == 1}
	if observedReused != reused {
		return exchangeProof{}, malformedResponseError{fmt.Sprintf("HTTP/2 connection reuse mismatch: expected %t, observed %t", reused, observedReused)}
	}
	if response.ProtoMajor != 2 || response.Proto != "HTTP/2.0" {
		return exchangeProof{}, malformedResponseError{"HTTP/2 protocol proof mismatch or fallback"}
	}
	if response.StatusCode != http.StatusOK || httpValue.ResponseContentType != mediaType {
		return exchangeProof{}, malformedResponseError{"DoH2 response status/header mismatch"}
	}
	if selectedScenario() == strictScenario && httpValue.ResponseCacheControl != "no-store" {
		return exchangeProof{}, malformedResponseError{"strict DoH2 response requires Cache-Control: no-store"}
	}
	dnsValue, err := validateDNS(body)
	if err != nil {
		return exchangeProof{}, err
	}
	return exchangeProof{DNS: dnsValue, HTTP: httpValue, TLS: tlsValue, QueryLatencyMilliseconds: durationMS(time.Since(started)), TimeToFirstByteMilliseconds: firstByteLatency, TransferredBytes: int64(len(canonicalQuery) + len(body)), ConnectionLatencyMilliseconds: connectionLatency}, nil
}

func validateTLS(state *tls.ConnectionState) (tlsProof, error) {
	if state == nil {
		return tlsProof{}, errors.New("missing TLS state")
	}
	proof := tlsProof{TLSProfileID: tlsProfileID(), TLSVersion: tlsVersionName(state.Version), CipherSuite: tls.CipherSuiteName(state.CipherSuite), KeyExchangeGroup: "X25519", SignatureScheme: "ecdsa_secp256r1_sha256", ALPN: state.NegotiatedProtocol, ServerName: serverName, HandshakeComplete: state.HandshakeComplete, DidResume: state.DidResume, EarlyDataAttempted: false, CertificateProfile: selectedCertificateProfile(), VerifiedChainCount: len(state.VerifiedChains), SentCertificateCount: len(state.PeerCertificates), TrustAnchorSent: false, PlatformProvenance: runtimeProvenance(), AccelerationProvenance: accelerationProvenance()}
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
	if state.NegotiatedProtocol != "h2" {
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

func validateDNS(raw []byte) (dnsProof, error) {
	if len(raw) < 12 || binary.BigEndian.Uint16(raw[:2]) != 0 {
		return dnsProof{}, malformedResponseError{"response message ID must be zero"}
	}
	flags := binary.BigEndian.Uint16(raw[2:4])
	if flags != 0x8400 {
		return dnsProof{}, malformedResponseError{"response flags/rcode mismatch"}
	}
	if binary.BigEndian.Uint16(raw[4:6]) != 1 || binary.BigEndian.Uint16(raw[6:8]) != 1 || binary.BigEndian.Uint16(raw[8:10]) != 0 || binary.BigEndian.Uint16(raw[10:12]) != 0 {
		return dnsProof{}, malformedResponseError{"response section counts mismatch"}
	}
	// The public normalization procedure permits alternate name compression. Parse the exact semantics, then canonical-reserialize.
	name, next, err := readName(raw, 12)
	if err != nil || name != "plab.test." {
		return dnsProof{}, malformedResponseError{"response question name mismatch"}
	}
	if next+4 > len(raw) || binary.BigEndian.Uint16(raw[next:next+2]) != 1 || binary.BigEndian.Uint16(raw[next+2:next+4]) != 1 {
		return dnsProof{}, malformedResponseError{"response question type/class mismatch"}
	}
	next += 4
	owner, next, err := readName(raw, next)
	if err != nil || owner != "plab.test." {
		return dnsProof{}, malformedResponseError{"answer owner mismatch"}
	}
	if next+10 > len(raw) {
		return dnsProof{}, malformedResponseError{"truncated answer"}
	}
	typ := binary.BigEndian.Uint16(raw[next : next+2])
	class := binary.BigEndian.Uint16(raw[next+2 : next+4])
	ttl := binary.BigEndian.Uint32(raw[next+4 : next+8])
	length := int(binary.BigEndian.Uint16(raw[next+8 : next+10]))
	next += 10
	if typ != 1 || class != 1 || ttl != 0 || length != 4 || next+4 != len(raw) || !bytes.Equal(raw[next:next+4], []byte{192, 0, 2, 1}) {
		return dnsProof{}, malformedResponseError{"A answer semantics mismatch"}
	}
	canonical := canonicalResponse
	normalizedHash := hash(canonical)
	if len(canonical) != 43 || normalizedHash != responseHash {
		return dnsProof{}, errors.New("internal canonical response fixture mismatch")
	}
	return dnsProof{FixtureID: fixtureID, Transport: "doh2", QuestionName: "plab.test.", QuestionType: "A", QuestionClass: "IN", Answer: "192.0.2.1", TTLSeconds: 0, ResponseCode: "NOERROR", AuthoritativeAnswer: true, RecursionDesired: false, RecursionAvailable: false, ExternalUpstreamUsed: false, CacheEnabled: false, Framing: "bare-dns-message-body", AuthorityMode: "local-fixture-authoritative", RequestMessageID: 0, RuntimeMessageID: 0, ResponseMessageID: 0, CanonicalHashNormalization: "identity", QueryLengthBytes: len(canonicalQuery), QueryNormalizedSHA256: hash(canonicalQuery), ResponseRawLengthBytes: len(raw), ResponseCanonicalLengthBytes: len(canonical), ResponseLengthBytes: len(canonical), ResponseNormalizedSHA256: normalizedHash, ResponseCompressionDiffered: !bytes.Equal(raw, canonical), ResponseCanonicalized: true, RawResponseEqualityRequired: false}, nil
}

func readName(message []byte, offset int) (string, int, error) {
	var labels []string
	next := offset
	jumped := false
	seen := map[int]bool{}
	for steps := 0; steps < 32; steps++ {
		if offset >= len(message) {
			return "", 0, errors.New("truncated DNS name")
		}
		if seen[offset] {
			return "", 0, errors.New("DNS compression loop")
		}
		seen[offset] = true
		b := message[offset]
		if b == 0 {
			if !jumped {
				next = offset + 1
			}
			return strings.Join(labels, ".") + ".", next, nil
		}
		if b&0xc0 == 0xc0 {
			if offset+1 >= len(message) {
				return "", 0, errors.New("truncated DNS pointer")
			}
			if !jumped {
				next = offset + 2
				jumped = true
			}
			offset = int(b&0x3f)<<8 | int(message[offset+1])
			continue
		}
		if b&0xc0 != 0 || int(b) > 63 || offset+1+int(b) > len(message) {
			return "", 0, errors.New("invalid DNS label")
		}
		labels = append(labels, string(message[offset+1:offset+1+int(b)]))
		offset += 1 + int(b)
		if !jumped {
			next = offset
		}
	}
	return "", 0, errors.New("DNS name exceeded parser bound")
}

func writeEvidence(output string, summary phaseSummary, err error) {
	validation := validationDocument(summary, err)
	writeRequired(output, "validation.json", validation)
	writeRequired(output, "result.json", validation)
	var dns any
	var httpValue any
	var tlsValue any
	if summary.LastProof != nil {
		dns = summary.LastProof.DNS
		httpValue = summary.LastProof.HTTP
		tlsValue = summary.LastProof.TLS
		writeRequired(output, "dns-wire-summary.json", dns)
		writeRequired(output, "http-summary.json", httpValue)
		writeRequired(output, "tls-negotiation.json", tlsValue)
	}
	writeRequired(output, "protocol-proof.json", map[string]any{"requestedProtocol": "doh2", "observedProtocol": observedProtocol(summary), "protocolVariant": protocolVariant(), "fallbackDetected": observedProtocol(summary) != "doh2", "tls": tlsValue, "http": httpValue, "dns": dns})
}
func normalizeResult(s phaseSummary, requested map[string]any) result {
	m := metrics{QueriesPerSecond: float64(s.CompletedOperations) / s.DurationSeconds, QueryLatencyMean: mean(s.QueryLatencies), QueryLatencyP50: percentile(s.QueryLatencies, .50), QueryLatencyP75: percentile(s.QueryLatencies, .75), QueryLatencyP90: percentile(s.QueryLatencies, .90), QueryLatencyP95: percentile(s.QueryLatencies, .95), QueryLatencyP99: percentile(s.QueryLatencies, .99), TimeToFirstByte: mean(s.TimeToFirstByte), ConnectionLatency: s.ConnectionLatencyMilliseconds, CompletedOperations: s.CompletedOperations, MalformedOperations: s.MalformedOperations, RetryCount: s.RetryCount, FailedOperations: s.FailedOperations, TimedOutOperations: s.TimedOutOperations, TotalTransferredBytes: s.TotalTransferredBytes, EffectiveConcurrency: s.EffectiveConcurrency, EffectiveStreams: s.EffectiveStreams}
	proof := s.LastProof
	proof.TLS.ConnectionLatencyMilliseconds = s.ConnectionLatencyMilliseconds
	artifacts := []string{"validation.json", "protocol-proof.json", "dns-wire-summary.json", "http-summary.json", "tls-negotiation.json", "dns-doh2-executor-result.json", "dns-load-summary.json", "dns-warmup-summary.json", "executor-identity.json", "load-generator-identity.json"}
	return result{SchemaVersion: "protocol-lab.dns-doh2-executor-result.v1", ScenarioID: selectedScenario(), LoadProfileID: supportedProfile, Status: "passed", Executor: map[string]string{"id": executorID, "version": executorVersion}, LoadGenerator: map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, ProtocolProof: map[string]any{"requestedProtocol": "doh2", "observedProtocol": "doh2", "protocolVariant": protocolVariant(), "fallbackDetected": false, "tls": proof.TLS, "http": proof.HTTP, "dns": proof.DNS}, Validation: map[string]any{"status": "passed", "zeroUnexpectedFailures": true, "zeroTimeouts": true, "zeroMalformed": true, "zeroRetries": true, "localAuthoritativeOnly": true, "externalUpstreamUsed": false, "cacheEnabled": false}, RequestedLoad: requested, EffectiveLoad: map[string]any{"connections": 1, "activeConnections": s.ActiveConnections, "concurrency": 1, "outstandingQueries": 1, "streamsPerConnection": 1, "activeStreams": s.EffectiveStreams}, Metrics: m, Warnings: []string{"Local package-backed DoH2 smoke is diagnostic and non-publishable. Every other DNS binding or semantic fixture is unsupported by this executor."}, Artifacts: artifacts}
}
func validationDocument(s phaseSummary, err error) map[string]any {
	return map[string]any{"scenarioId": selectedScenario(), "fixtureId": fixtureID, "passed": err == nil, "requestedProtocol": "doh2", "observedProtocol": observedProtocol(s), "protocolVariant": protocolVariant(), "fallbackDetected": observedProtocol(s) != "doh2", "completedOperations": s.CompletedOperations, "malformedOperations": s.MalformedOperations, "retryCount": s.RetryCount, "failedOperations": s.FailedOperations, "timedOutOperations": s.TimedOutOperations, "externalUpstreamUsed": false, "cacheEnabled": false, "error": errorString(err)}
}
func observedProtocol(s phaseSummary) string {
	if s.LastProof != nil && s.LastProof.HTTP.Version == "HTTP/2.0" && s.LastProof.TLS.ALPN == "h2" {
		return "doh2"
	}
	return ""
}
func record(s *phaseSummary, p exchangeProof) {
	s.CompletedOperations++
	s.TotalTransferredBytes += p.TransferredBytes
	s.QueryLatencies = append(s.QueryLatencies, p.QueryLatencyMilliseconds)
	s.TimeToFirstByte = append(s.TimeToFirstByte, p.TimeToFirstByteMilliseconds)
	s.LastProof = &p
}
func classify(s *phaseSummary, err error) {
	var malformed malformedResponseError
	if errors.As(err, &malformed) {
		s.MalformedOperations++
	} else if isTimeout(err) {
		s.TimedOutOperations++
	} else {
		s.FailedOperations++
	}
	s.Errors[err.Error()]++
}
func hasFailures(s phaseSummary) bool {
	return s.MalformedOperations != 0 || s.RetryCount != 0 || s.FailedOperations != 0 || s.TimedOutOperations != 0
}
func normalizeTarget(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("target address required")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("DoH2 target must be an https URL without query or fragment")
	}
	if parsed.Path != "" && parsed.Path != "/" && parsed.Path != "/dns-query" {
		return "", errors.New("DoH2 target path must be /dns-query")
	}
	parsed.Path = "/dns-query"
	return parsed.String(), nil
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
func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func mustHex(value string) []byte {
	result, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return result
}
func durationMS(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }
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
func selectedScenario() string {
	id := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if id == "" {
		return strictScenario
	}
	return id
}
func protocolVariant() string {
	if selectedScenario() == interopScenario {
		return "doh-h2-rfc8484-interoperability"
	}
	return "doh-h2-tls-alpn"
}
func tlsProfileID() string {
	if selectedScenario() == interopScenario {
		return interopTLSProfile
	}
	return strictTLSProfile
}
func selectedCertificateProfile() string {
	if selectedScenario() == interopScenario {
		return interopCertProfile
	}
	return certificateProfile
}
func runtimeProvenance() map[string]string {
	return map[string]string{"goos": runtime.GOOS, "goarch": runtime.GOARCH, "goVersion": runtime.Version()}
}
func accelerationProvenance() map[string]string { return map[string]string{"mode": "not-reported"} }
func fatal(code int, err error)                 { fmt.Fprintln(os.Stderr, err); os.Exit(code) }
