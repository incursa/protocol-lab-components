package main

import (
	"bytes"
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
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

const (
	executorID           = "go-dns-doh3-executor"
	executorVersion      = "0.2.0"
	loadGeneratorID      = "go-dns-doh3-load"
	loadGeneratorVersion = "0.2.0"
	profileID            = "secure-dns-smoke"
	serverName           = "dns.plab.test"
	mediaType            = "application/dns-message"
	certificateProfile   = "plab-secure-dns-single-leaf-p256-v1"
	requiredCipher       = "TLS_AES_128_GCM_SHA256"
	leafDERHash          = "b57bdd3eb90b36455900c17de9ff9a02c623e1f6b27626ad7821a40e35e8251c"
	leafSPKIHash         = "cfa6d5d08ee2071e28fd96205b088156fa71b460262470e5b994624b0537cf25"
)

var knownUnsupported = map[string]struct{}{
	"dns.classic.tcp.query.a": {}, "dns.classic.udp-truncated-tcp-retry": {}, "dns.classic.udp.query.a": {},
	"dns.doh2.query.a": {}, "dns.doh2.interoperability.query.a": {}, "dns.doq.query.a": {}, "dns.doq.interoperability.query.a": {},
	"dns.dot.query.a": {}, "dns.dot.interoperability.query.a": {},
}

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
	Version           string `json:"version"`
	VersionNumber     uint32 `json:"versionNumber"`
	ALPN              string `json:"alpn"`
	UsedZeroRTT       bool   `json:"usedZeroRtt"`
	SupportsDatagrams bool   `json:"supportsDatagrams"`
	Binding           string `json:"binding"`
	ConnectionReused  bool   `json:"connectionReused"`
}
type httpProof struct {
	Version              string `json:"httpVersion"`
	Method               string `json:"method"`
	Scheme               string `json:"scheme"`
	Authority            string `json:"authority"`
	Path                 string `json:"path"`
	RequestTarget        string `json:"requestTarget"`
	StatusCode           int    `json:"responseStatus"`
	RequestAccept        string `json:"requestAccept"`
	RequestContentType   string `json:"requestContentType"`
	RequestCacheControl  string `json:"requestCacheControl"`
	ResponseContentType  string `json:"responseContentType"`
	ResponseCacheControl string `json:"responseCacheControl"`
	BodyFraming          string `json:"bodyFraming"`
	ConnectionReused     bool   `json:"connectionReused"`
	FallbackDetected     bool   `json:"fallbackDetected"`
	GETQueryEncoding     string `json:"getQueryEncoding,omitempty"`
	GETRequestBody       string `json:"getRequestBody,omitempty"`
}
type dnsProof struct {
	FixtureID                    string `json:"fixtureId"`
	Transport                    string `json:"transport"`
	QuestionName                 string `json:"questionName"`
	QuestionType                 string `json:"questionType"`
	QuestionClass                string `json:"questionClass"`
	Answer                       string `json:"answer"`
	ResponseCode                 string `json:"responseCode"`
	AuthoritativeAnswer          bool   `json:"authoritativeAnswer"`
	RecursionDesired             bool   `json:"recursionDesired"`
	RecursionAvailable           bool   `json:"recursionAvailable"`
	ExternalUpstreamUsed         bool   `json:"externalUpstreamUsed"`
	CacheEnabled                 bool   `json:"cacheEnabled"`
	Framing                      string `json:"framing"`
	AuthorityMode                string `json:"authorityMode"`
	RequestMessageID             uint16 `json:"requestMessageId"`
	ResponseMessageID            uint16 `json:"responseMessageId"`
	CanonicalHashNormalization   string `json:"canonicalHashNormalization"`
	QueryLengthBytes             int    `json:"queryLengthBytes"`
	QueryNormalizedSHA256        string `json:"queryNormalizedSha256"`
	ResponseRawLengthBytes       int    `json:"responseRawLengthBytes"`
	ResponseCanonicalLengthBytes int    `json:"responseCanonicalLengthBytes"`
	ResponseNormalizedSHA256     string `json:"responseNormalizedSha256"`
	ResponseCompressionDiffered  bool   `json:"responseCompressionDiffered"`
	ResponseCanonicalized        bool   `json:"responseCanonicalized"`
	RawResponseEqualityRequired  bool   `json:"rawResponseEqualityRequired"`
	DNSSECSignatureValidity      string `json:"dnssecSignatureValidity,omitempty"`
}
type exchangeProof struct {
	DNS                         dnsProof  `json:"dns"`
	HTTP                        httpProof `json:"http"`
	TLS                         tlsProof  `json:"tls"`
	QUIC                        quicProof `json:"quic"`
	QueryLatencyMilliseconds    float64   `json:"queryLatencyMilliseconds"`
	TimeToFirstByteMilliseconds float64   `json:"timeToFirstByteMilliseconds"`
	TransferredBytes            int64     `json:"transferredBytes"`
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

func (e malformedResponseError) Error() string { return e.reason }

type phaseClient struct {
	client            *http.Client
	transport         *http3.Transport
	connection        *quic.Conn
	dialCount         atomic.Int32
	connectionLatency float64
}

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "DoH3 target HTTPS URL")
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
	if err := os.MkdirAll(*output, 0755); err != nil {
		fatal(1, err)
	}
	f := checkIdentityOrExit(*output)
	endpoint, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	roots, err := loadRoots(*rootPath)
	if err != nil {
		fatal(2, err)
	}
	preflight, err := runPhase(endpoint, roots, f, 0, true)
	writeEvidence(*output, f, preflight, err)
	writeRequired(*output, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedScenarios": sortedScenarioIDs()})
	if err != nil {
		fatal(1, fmt.Errorf("DoH3 validity gate failed: %w", err))
	}
	if *validationOnly {
		fmt.Fprintln(os.Stderr, "go-dns-doh3-executor validity gate passed")
		return
	}
	requested, err := loadConfig()
	if err != nil {
		fatal(2, err)
	}
	warmup, err := runPhase(endpoint, roots, f, time.Second, false)
	writeRequired(*output, "dns-warmup-summary.json", warmup)
	if err != nil || warmup.CompletedOperations == 0 || hasFailures(warmup) {
		fatal(1, fmt.Errorf("DoH3 warmup rejected: %w", err))
	}
	measured, err := runPhase(endpoint, roots, f, 5*time.Second, false)
	writeRequired(*output, "dns-load-summary.json", measured)
	if err != nil || measured.CompletedOperations == 0 || hasFailures(measured) {
		fatal(1, fmt.Errorf("DoH3 measured phase rejected: %w", err))
	}
	normalized := normalizeResult(f, measured, requested)
	writeRequired(*output, "load-generator-identity.json", normalized.LoadGenerator)
	writeRequired(*output, "dns-doh3-executor-result.json", normalized)
	writeRequired(*output, "result.json", normalized)
	data, _ := json.MarshalIndent(normalized, "", "  ")
	fmt.Println(string(data))
}

func checkIdentityOrExit(output string) fixture {
	requireIdentity("PLAB_EXECUTOR_ID", executorID, "executor")
	requireIdentity("PLAB_EXECUTOR_VERSION", executorVersion, "executor version")
	requireIdentity("PLAB_LOAD_GENERATOR_ID", loadGeneratorID, "load generator")
	requireIdentity("PLAB_LOAD_GENERATOR_VERSION", loadGeneratorVersion, "load generator version")
	requireIdentity("PLAB_PROTOCOL", "doh3", "protocol")
	requireIdentity("PLAB_PROTOCOL_VARIANT", "doh-h3-quic-v1", "protocol variant")
	id := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if f, ok := fixtures[id]; ok {
		return f
	}
	if _, ok := knownUnsupported[id]; ok {
		doc := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": id, "executorId": executorID, "reason": "exact scenario belongs to an independent non-DoH3 DNS lane"}
		writeRequired(output, "unsupported.json", doc)
		writeRequired(output, "result.json", doc)
		data, _ := json.Marshal(doc)
		fmt.Println(string(data))
		os.Exit(3)
	}
	fatal(2, fmt.Errorf("unknown scenario identity %q", id))
	return fixture{}
}

func loadConfig() (map[string]any, error) {
	if strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID")) != profileID {
		return nil, fmt.Errorf("supports load profile %q only", profileID)
	}
	values := map[string]int{"connections": envInt("PLAB_CONNECTIONS"), "concurrency": envInt("PLAB_CONCURRENCY"), "durationSeconds": envInt("PLAB_DURATION_SECONDS"), "warmupSeconds": envInt("PLAB_WARMUP_SECONDS"), "repetition": envInt("PLAB_REPETITION")}
	if values["connections"] != 1 || values["concurrency"] != 1 || values["durationSeconds"] != 5 || values["warmupSeconds"] != 1 || values["repetition"] != 1 {
		return nil, fmt.Errorf("secure-dns-smoke mismatch: %v", values)
	}
	return map[string]any{"connections": 1, "concurrency": 1, "outstandingQueries": 1, "connectionReuse": "reuse", "durationSeconds": 5, "warmupSeconds": 1, "repetition": 1, "queryTimeoutMilliseconds": 5000, "maxRetries": 0}, nil
}

func newPhaseClient(endpoint string, roots *x509.CertPool) (*phaseClient, error) {
	p := &phaseClient{}
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: roots, ServerName: serverName, NextProtos: []string{http3.NextProtoH3}, ClientSessionCache: nil, CurvePreferences: []tls.CurveID{tls.X25519}}
	quicCfg := &quic.Config{Versions: []quic.Version{quic.Version1}, HandshakeIdleTimeout: 5 * time.Second, MaxIdleTimeout: 30 * time.Second, KeepAlivePeriod: 5 * time.Second, EnableDatagrams: false}
	p.transport = &http3.Transport{TLSClientConfig: tlsCfg, QUICConfig: quicCfg, DisableCompression: true}
	p.transport.Dial = func(ctx context.Context, addr string, tc *tls.Config, qc *quic.Config) (*quic.Conn, error) {
		started := time.Now()
		conn, err := quic.DialAddr(ctx, addr, tc, qc)
		if err == nil {
			p.connection = conn
			p.connectionLatency = durationMS(time.Since(started))
			p.dialCount.Add(1)
		}
		return conn, err
	}
	p.client = &http.Client{Transport: p.transport, Timeout: 5 * time.Second}
	_ = endpoint
	return p, nil
}

func runPhase(endpoint string, roots *x509.CertPool, f fixture, duration time.Duration, once bool) (phaseSummary, error) {
	s := phaseSummary{Phase: "measured", EffectiveConcurrency: 1, EffectiveStreams: 1, ActiveConnections: 1, Errors: map[string]int{}}
	if once {
		s.Phase = "preflight"
	}
	p, _ := newPhaseClient(endpoint, roots)
	defer p.transport.Close()
	first, err := exchange(p, endpoint, f, false)
	s.ConnectionLatencyMilliseconds = p.connectionLatency
	if err != nil {
		classify(&s, err)
		return s, err
	}
	if once {
		record(&s, first)
		s.DurationSeconds = first.QueryLatencyMilliseconds / 1000
		return s, nil
	}
	started := time.Now()
	deadline := started.Add(duration)
	for time.Now().Before(deadline) {
		proof, e := exchange(p, endpoint, f, true)
		if e != nil {
			classify(&s, e)
			return s, e
		}
		record(&s, proof)
	}
	s.DurationSeconds = time.Since(started).Seconds()
	return s, nil
}

func exchange(p *phaseClient, endpoint string, f fixture, expectReused bool) (exchangeProof, error) {
	started := time.Now()
	method := f.Method
	target := endpoint
	var body io.Reader
	if method == http.MethodGet {
		target += "?dns=" + getValue(f)
	} else {
		body = bytes.NewReader(bytesOf(f.QueryHex))
	}
	req, err := http.NewRequestWithContext(context.Background(), method, target, body)
	if err != nil {
		return exchangeProof{}, err
	}
	req.Host = serverName
	req.Header.Set("Accept", mediaType)
	req.Header.Set("Cache-Control", "no-cache")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", mediaType)
	}
	ttfb := float64(0)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{GotFirstResponseByte: func() { ttfb = durationMS(time.Since(started)) }}))
	before := p.dialCount.Load()
	resp, err := p.client.Do(req)
	if err != nil {
		return exchangeProof{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return exchangeProof{}, err
	}
	reused := before > 0
	if reused != expectReused {
		return exchangeProof{}, malformedResponseError{fmt.Sprintf("HTTP/3 connection reuse mismatch expected=%t observed=%t", expectReused, reused)}
	}
	tlsValue, err := validateTLS(resp.TLS, p.connectionLatency)
	if err != nil {
		return exchangeProof{}, err
	}
	quicValue, err := validateQUIC(p.connection, reused)
	if err != nil {
		return exchangeProof{}, err
	}
	requestTarget := "/dns-query"
	if method == http.MethodGet {
		requestTarget += "?dns=" + getValue(f)
	}
	httpValue := httpProof{Version: resp.Proto, Method: method, Scheme: "https", Authority: serverName, Path: "/dns-query", RequestTarget: requestTarget, StatusCode: resp.StatusCode, RequestAccept: mediaType, RequestContentType: req.Header.Get("Content-Type"), RequestCacheControl: "no-cache", ResponseContentType: resp.Header.Get("Content-Type"), ResponseCacheControl: resp.Header.Get("Cache-Control"), BodyFraming: "bare-dns-message-body", ConnectionReused: reused, FallbackDetected: resp.ProtoMajor != 3}
	if method == http.MethodGet {
		httpValue.GETQueryEncoding = "base64url-unpadded"
		httpValue.GETRequestBody = "absent"
	}
	if resp.ProtoMajor != 3 || resp.Proto != "HTTP/3.0" || resp.StatusCode != 200 || httpValue.ResponseContentType != mediaType || httpValue.ResponseCacheControl != "no-store" {
		return exchangeProof{}, malformedResponseError{"DoH3 protocol/status/header proof mismatch or fallback"}
	}
	dnsValue, err := validateDNS(raw, f)
	if err != nil {
		return exchangeProof{}, err
	}
	return exchangeProof{DNS: dnsValue, HTTP: httpValue, TLS: tlsValue, QUIC: quicValue, QueryLatencyMilliseconds: durationMS(time.Since(started)), TimeToFirstByteMilliseconds: ttfb, TransferredBytes: int64(len(bytesOf(f.QueryHex)) + len(raw))}, nil
}

func validateTLS(state *tls.ConnectionState, latency float64) (tlsProof, error) {
	if state == nil {
		return tlsProof{}, errors.New("missing TLS state")
	}
	p := tlsProof{TLSVersion: tlsVersionName(state.Version), CipherSuite: tls.CipherSuiteName(state.CipherSuite), KeyExchangeGroup: "X25519", SignatureScheme: "ecdsa_secp256r1_sha256", ALPN: state.NegotiatedProtocol, ServerName: serverName, HandshakeComplete: state.HandshakeComplete, DidResume: state.DidResume, EarlyDataAttempted: false, CertificateProfile: certificateProfile, VerifiedChainCount: len(state.VerifiedChains), SentCertificateCount: len(state.PeerCertificates), TrustAnchorSent: false, ConnectionLatencyMilliseconds: latency, PlatformProvenance: runtimeProvenance(), AccelerationProvenance: accelerationProvenance()}
	if len(state.PeerCertificates) > 0 {
		c := state.PeerCertificates[0]
		p.CertificateDERSHA256 = hashOf(c.Raw)
		p.CertificateSPKISHA256 = hashOf(c.RawSubjectPublicKeyInfo)
		p.CertificateSignatureAlgorithm = c.SignatureAlgorithm.String()
		p.CertificatePublicKeyAlgorithm = c.PublicKeyAlgorithm.String()
		if k, ok := c.PublicKey.(*ecdsa.PublicKey); ok {
			p.CertificateNamedCurve = k.Curve.Params().Name
		}
	}
	if state.Version != tls.VersionTLS13 || !state.HandshakeComplete || state.DidResume || state.NegotiatedProtocol != http3.NextProtoH3 || p.CipherSuite != requiredCipher || len(state.VerifiedChains) == 0 || p.CertificateDERSHA256 != leafDERHash || p.CertificateSPKISHA256 != leafSPKIHash {
		return p, errors.New("TLS 1.3 / h3 / certificate identity gate failed")
	}
	return p, nil
}
func validateQUIC(conn *quic.Conn, reused bool) (quicProof, error) {
	if conn == nil {
		return quicProof{}, errors.New("missing QUIC connection")
	}
	s := conn.ConnectionState()
	p := quicProof{Version: s.Version.String(), VersionNumber: uint32(s.Version), ALPN: s.TLS.NegotiatedProtocol, UsedZeroRTT: s.Used0RTT, SupportsDatagrams: s.SupportsDatagrams.Local || s.SupportsDatagrams.Remote, Binding: "http3-rfc9114-quic-v1", ConnectionReused: reused}
	if s.Version != quic.Version1 || s.Used0RTT || s.TLS.NegotiatedProtocol != http3.NextProtoH3 {
		return p, errors.New("exact QUIC v1 HTTP/3 binding gate failed")
	}
	return p, nil
}
func validateDNS(raw []byte, f fixture) (dnsProof, error) {
	if !semanticEqual(raw, f) {
		return dnsProof{}, malformedResponseError{"DNS response semantic fixture mismatch"}
	}
	canonical := bytesOf(f.ResponseHex)
	p := dnsProof{FixtureID: f.FixtureID, Transport: "doh3", QuestionName: f.QuestionName, QuestionType: f.QuestionType, QuestionClass: "IN", Answer: f.Answer, ResponseCode: f.RCode, AuthoritativeAnswer: true, RecursionDesired: false, RecursionAvailable: false, ExternalUpstreamUsed: false, CacheEnabled: false, Framing: "bare-dns-message-body", AuthorityMode: "local-fixture-authoritative", RequestMessageID: 0, ResponseMessageID: 0, CanonicalHashNormalization: "identity", QueryLengthBytes: len(bytesOf(f.QueryHex)), QueryNormalizedSHA256: hashOf(bytesOf(f.QueryHex)), ResponseRawLengthBytes: len(raw), ResponseCanonicalLengthBytes: len(canonical), ResponseNormalizedSHA256: hashOf(canonical), ResponseCompressionDiffered: !bytes.Equal(raw, canonical), ResponseCanonicalized: true, RawResponseEqualityRequired: false}
	if f.ScenarioID == "dns.doh3.query.large-dnssec-shaped" {
		p.DNSSECSignatureValidity = "not-claimed"
	}
	if p.QueryNormalizedSHA256 != f.QueryHash || p.ResponseNormalizedSHA256 != f.ResponseHash {
		return dnsProof{}, errors.New("canonical DNS hash mismatch")
	}
	return p, nil
}

func writeEvidence(output string, f fixture, s phaseSummary, err error) {
	validation := validationDocument(f, s, err)
	writeRequired(output, "validation.json", validation)
	writeRequired(output, "result.json", validation)
	var dnsValue, httpValue, tlsValue, quicValue any
	if s.LastProof != nil {
		dnsValue = s.LastProof.DNS
		httpValue = s.LastProof.HTTP
		tlsValue = s.LastProof.TLS
		quicValue = s.LastProof.QUIC
		writeRequired(output, "dns-wire-summary.json", dnsValue)
		writeRequired(output, "http-summary.json", httpValue)
		writeRequired(output, "quic-summary.json", quicValue)
		writeRequired(output, "tls-negotiation.json", tlsValue)
	}
	writeRequired(output, "protocol-proof.json", map[string]any{"requestedProtocol": "doh3", "observedProtocol": observedProtocol(s), "protocolVariant": "doh-h3-quic-v1", "fallbackDetected": observedProtocol(s) != "doh3", "dns": dnsValue, "http": httpValue, "tls": tlsValue, "quic": quicValue})
}
func normalizeResult(f fixture, s phaseSummary, requested map[string]any) result {
	m := metrics{QueriesPerSecond: float64(s.CompletedOperations) / s.DurationSeconds, QueryLatencyMean: mean(s.QueryLatencies), QueryLatencyP50: percentile(s.QueryLatencies, .50), QueryLatencyP75: percentile(s.QueryLatencies, .75), QueryLatencyP90: percentile(s.QueryLatencies, .90), QueryLatencyP95: percentile(s.QueryLatencies, .95), QueryLatencyP99: percentile(s.QueryLatencies, .99), TimeToFirstByte: mean(s.TimeToFirstByte), ConnectionLatency: s.ConnectionLatencyMilliseconds, CompletedOperations: s.CompletedOperations, MalformedOperations: s.MalformedOperations, RetryCount: s.RetryCount, FailedOperations: s.FailedOperations, TimedOutOperations: s.TimedOutOperations, TotalTransferredBytes: s.TotalTransferredBytes, EffectiveConcurrency: s.EffectiveConcurrency, EffectiveStreams: s.EffectiveStreams}
	p := s.LastProof
	return result{SchemaVersion: "protocol-lab.dns-doh3-executor-result.v1", ScenarioID: f.ScenarioID, LoadProfileID: profileID, Status: "passed", Executor: map[string]string{"id": executorID, "version": executorVersion}, LoadGenerator: map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, ProtocolProof: map[string]any{"requestedProtocol": "doh3", "observedProtocol": "doh3", "protocolVariant": "doh-h3-quic-v1", "fallbackDetected": false, "dns": p.DNS, "http": p.HTTP, "tls": p.TLS, "quic": p.QUIC}, Validation: map[string]any{"status": "passed", "zeroUnexpectedFailures": true, "zeroTimeouts": true, "zeroMalformed": true, "zeroRetries": true, "localAuthoritativeOnly": true, "externalUpstreamUsed": false, "cacheEnabled": false}, RequestedLoad: requested, EffectiveLoad: map[string]any{"connections": 1, "activeConnections": s.ActiveConnections, "concurrency": 1, "outstandingQueries": 1, "streamsPerConnection": 1, "activeStreams": s.EffectiveStreams}, Metrics: m, Warnings: []string{"Local package-backed DoH3 smoke is diagnostic and non-publishable. DNSSEC-shaped material is structural; cryptographic signature validity is not claimed."}, Artifacts: []string{"validation.json", "protocol-proof.json", "dns-wire-summary.json", "http-summary.json", "quic-summary.json", "tls-negotiation.json", "dns-doh3-executor-result.json", "dns-load-summary.json", "dns-warmup-summary.json", "executor-identity.json", "load-generator-identity.json"}}
}
func validationDocument(f fixture, s phaseSummary, err error) map[string]any {
	return map[string]any{"scenarioId": f.ScenarioID, "fixtureId": f.FixtureID, "passed": err == nil, "requestedProtocol": "doh3", "observedProtocol": observedProtocol(s), "protocolVariant": "doh-h3-quic-v1", "fallbackDetected": observedProtocol(s) != "doh3", "completedOperations": s.CompletedOperations, "malformedOperations": s.MalformedOperations, "retryCount": s.RetryCount, "failedOperations": s.FailedOperations, "timedOutOperations": s.TimedOutOperations, "externalUpstreamUsed": false, "cacheEnabled": false, "error": errorString(err)}
}
func observedProtocol(s phaseSummary) string {
	if s.LastProof != nil && s.LastProof.HTTP.Version == "HTTP/3.0" && s.LastProof.TLS.ALPN == "h3" && s.LastProof.QUIC.Version == "v1" {
		return "doh3"
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
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	if u.Scheme != "https" || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("DoH3 target must be https without query or fragment")
	}
	if u.Path != "" && u.Path != "/" && u.Path != "/dns-query" {
		return "", errors.New("DoH3 target path must be /dns-query")
	}
	u.Path = "/dns-query"
	return u.String(), nil
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
func sortedScenarioIDs() []string {
	ids := make([]string, 0, len(fixtures))
	for id := range fixtures {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	total := float64(0)
	for _, x := range v {
		total += x
	}
	return total / float64(len(v))
}
func percentile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	i := int(math.Ceil(q*float64(len(c)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(c) {
		i = len(c) - 1
	}
	return c[i]
}
func envInt(name string) int { v, _ := strconv.Atoi(strings.TrimSpace(os.Getenv(name))); return v }
func requireIdentity(name, expected, label string) {
	if observed := strings.TrimSpace(os.Getenv(name)); observed != expected {
		fatal(2, fmt.Errorf("%s identity mismatch expected=%q observed=%q", label, expected, observed))
	}
}
func tlsVersionName(v uint16) string {
	if v == tls.VersionTLS13 {
		return "TLS1.3"
	}
	return fmt.Sprintf("0x%04x", v)
}
func durationMS(v time.Duration) float64 { return float64(v) / float64(time.Millisecond) }
func isTimeout(err error) bool {
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
		err = os.WriteFile(filepath.Join(dir, name), data, 0644)
	}
	if err != nil {
		fatal(1, err)
	}
}
func runtimeProvenance() map[string]string {
	return map[string]string{"goos": runtime.GOOS, "goarch": runtime.GOARCH, "goVersion": runtime.Version()}
}
func accelerationProvenance() map[string]string { return map[string]string{"mode": "not-reported"} }
func fatal(code int, err error)                 { fmt.Fprintln(os.Stderr, err); os.Exit(code) }
