package dnsclassic

import (
	"crypto/sha256"
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
	Version           = "0.1.0"
	ProfileID         = "dns-classic-diagnostic"
	aFixtureID        = "dns.plab-test-a-v2.canonical"
	largeFixtureID    = "dns.dnskey-plab-test-large-edns-dnssec-shaped.canonical"
	aQueryHash        = "c46b9fb76019b5a644d0884b17e816cb7c3076275d9468c27d180f70488eb8ec"
	aResponseHash     = "9d488461675ad5ab9f74c7b203861e1ad17521e413a407a25e6611012a595620"
	largeQueryHash    = "7445dfa148164c2ee02186b962f735847e9d62f46414287ebf5cf6a3dfce9e4f"
	truncatedHash     = "753eff72120531d638e839e088ca22875e550b89f554a11af50d4423a512710d"
	largeResponseHash = "1cc5bafd114a4f34d824c01a685b04e82cae6110c648046c461d3688782e8665"
)

var (
	aQuery            = mustHex("00000000000100000000000004706c616204746573740000010001")
	aResponse         = mustHex("00008400000100010000000004706c616204746573740000010001c00c00010001000000000004c0000201")
	largeQuery        = mustHex("00000000000100000000000106646e736b657904706c6162047465737400003000010000290200000080000000")
	truncatedResponse = mustHex("00008600000100000000000106646e736b657904706c6162047465737400003000010000290200000080000000")
	largeResponse     = mustHex("00008400000100070000000106646e736b657904706c616204746573740000300001c00c003000010000000000440101030d01010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101c00c003000010000000000440101030d02020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202c00c003000010000000000440101030d03030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303c00c003000010000000000440101030d04040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404c00c003000010000000000440101030d05050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505c00c003000010000000000440101030d06060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606c00c002e000100000000005d00300d030000000077359400713fb300303904706c6162047465737400a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a50000290200000080000000")
)

type Config struct {
	ExecutorID, LoadGeneratorID, Mode string
	Supported                         []string
}
type proof struct {
	FixtureID                       string `json:"fixtureId"`
	Transport                       string `json:"transport"`
	QuestionName                    string `json:"questionName"`
	QuestionType                    string `json:"questionType"`
	QuestionClass                   string `json:"questionClass"`
	Answer                          string `json:"answer"`
	ResponseCode                    string `json:"responseCode"`
	Framing                         string `json:"framing"`
	TTLSeconds                      int    `json:"ttlSeconds"`
	AuthoritativeAnswer             bool   `json:"authoritativeAnswer"`
	RecursionDesired                bool   `json:"recursionDesired"`
	RecursionAvailable              bool   `json:"recursionAvailable"`
	ExternalUpstreamUsed            bool   `json:"externalUpstreamUsed"`
	CacheEnabled                    bool   `json:"cacheEnabled"`
	RuntimeMessageID                uint16 `json:"runtimeMessageId"`
	ResponseMessageID               uint16 `json:"responseMessageId"`
	MessageIDCorrelated             bool   `json:"messageIdCorrelated"`
	MessageIDUniqueAmongOutstanding bool   `json:"messageIdUniqueAmongOutstanding"`
	CanonicalHashNormalization      string `json:"canonicalHashNormalization"`
	QueryLengthBytes                int    `json:"queryLengthBytes"`
	ResponseLengthBytes             int    `json:"responseLengthBytes"`
	QueryNormalizedSHA256           string `json:"queryNormalizedSha256"`
	ResponseNormalizedSHA256        string `json:"responseNormalizedSha256"`
	RequestWireHex                  string `json:"requestWireHex"`
	ResponseWireHex                 string `json:"responseWireHex"`
	UDPAdvertisedPayloadBytes       int    `json:"udpAdvertisedPayloadBytes,omitempty"`
	UDPTruncatedResponseLength      int    `json:"udpTruncatedResponseLength,omitempty"`
	RetryCount                      int    `json:"retryCount"`
	TCPResponseLength               int    `json:"tcpResponseLength,omitempty"`
	UDPTruncated                    bool   `json:"udpTruncated"`
	RetryQuestionIdentical          bool   `json:"retryQuestionIdentical,omitempty"`
	RetryMessageIDNew               bool   `json:"retryMessageIdNew,omitempty"`
	TCPResponsePrefixHex            string `json:"tcpResponsePrefixHex,omitempty"`
}
type summary struct {
	Phase                         string         `json:"phase"`
	DurationSeconds               float64        `json:"durationSeconds"`
	ConnectionLatencyMilliseconds float64        `json:"connectionLatencyMilliseconds"`
	CompletedOperations           int            `json:"completedOperations"`
	MalformedOperations           int            `json:"malformedOperations"`
	RetryCount                    int            `json:"retryCount"`
	FailedOperations              int            `json:"failedOperations"`
	TimedOutOperations            int            `json:"timedOutOperations"`
	TotalTransferredBytes         int64          `json:"totalTransferredBytes"`
	EffectiveConcurrency          int            `json:"effectiveConcurrency"`
	Latencies                     []float64      `json:"queryLatencyMilliseconds"`
	LastProof                     *proof         `json:"lastDnsProof,omitempty"`
	Errors                        map[string]int `json:"errors"`
}
type metricSet struct {
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
	Metrics       metricSet         `json:"metrics"`
	Warnings      []string          `json:"warnings"`
}
type malformedError struct{ reason string }

func (e malformedError) Error() string { return e.reason }

func Run(cfg Config) {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "classic DNS target address")
	output := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "artifact directory")
	validationOnly := flag.Bool("validation-only", false, "run one validity operation")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s %s\n", cfg.ExecutorID, Version)
		return
	}
	if *output == "" {
		*output = "artifacts"
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		fatal(1, err)
	}
	scenario := checkIdentity(cfg, *output)
	address, err := normalizeTarget(*target, cfg.Mode)
	if err != nil {
		fatal(2, err)
	}
	preflight, err := runPhase(address, scenario, 0, true)
	validation := validationDoc(cfg, scenario, preflight, err)
	write(*output, "validation.json", validation)
	write(*output, "result.json", validation)
	if preflight.LastProof != nil {
		write(*output, "dns-wire-summary.json", preflight.LastProof)
	}
	write(*output, "protocol-proof.json", map[string]any{"requestedProtocol": protocolVariant(scenario), "observedProtocol": protocolVariant(scenario), "fallbackDetected": false, "dns": preflight.LastProof})
	write(*output, "executor-identity.json", map[string]any{"id": cfg.ExecutorID, "version": Version, "role": "client-test-executor", "supportedScenarios": cfg.Supported})
	if err != nil {
		fatal(1, fmt.Errorf("classic DNS validity gate failed: %w", err))
	}
	if *validationOnly {
		fmt.Fprintln(os.Stderr, cfg.ExecutorID, "validity gate passed")
		return
	}
	requested, err := loadConfig()
	if err != nil {
		fatal(2, err)
	}
	warmup, err := runPhase(address, scenario, time.Second, false)
	write(*output, "dns-warmup-summary.json", warmup)
	if err != nil || hasFailures(warmup) {
		fatal(1, fmt.Errorf("warmup rejected: %w", err))
	}
	measured, err := runPhase(address, scenario, 5*time.Second, false)
	write(*output, "dns-load-summary.json", measured)
	if err != nil || hasFailures(measured) {
		fatal(1, fmt.Errorf("measured phase rejected: %w", err))
	}
	normalized := normalize(cfg, scenario, measured, requested)
	write(*output, "load-generator-identity.json", normalized.LoadGenerator)
	write(*output, "dns-classic-executor-result.json", normalized)
	write(*output, "result.json", normalized)
	data, _ := json.MarshalIndent(normalized, "", "  ")
	fmt.Println(string(data))
}

func checkIdentity(cfg Config, output string) string {
	verify("PLAB_EXECUTOR_ID", cfg.ExecutorID)
	verify("PLAB_EXECUTOR_VERSION", Version)
	verify("PLAB_LOAD_GENERATOR_ID", cfg.LoadGeneratorID)
	verify("PLAB_LOAD_GENERATOR_VERSION", Version)
	scenario := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	for _, id := range cfg.Supported {
		if scenario == id {
			return scenario
		}
	}
	if _, ok := allDNS[scenario]; ok {
		doc := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": scenario, "executorId": cfg.ExecutorID, "reason": "exact DNS scenario semantics are not implemented by this binding-specific executor"}
		write(output, "unsupported.json", doc)
		write(output, "result.json", doc)
		data, _ := json.Marshal(doc)
		fmt.Println(string(data))
		os.Exit(3)
	}
	fatal(2, fmt.Errorf("unknown scenario identity %q", scenario))
	return ""
}

var allDNS = map[string]struct{}{"dns.dot.query.a": {}, "dns.doh2.query.a": {}, "dns.doh3.get.a": {}, "dns.doh3.query.a": {}, "dns.doh3.query.aaaa": {}, "dns.doh3.query.cname-chain": {}, "dns.doh3.query.large-dnssec-shaped": {}, "dns.doh3.query.nodata": {}, "dns.doh3.query.nxdomain": {}, "dns.doq.query.a": {}, "dns.classic.udp.query.a": {}, "dns.classic.tcp.query.a": {}, "dns.classic.udp-truncated-tcp-retry": {}}

func runPhase(address, scenario string, duration time.Duration, once bool) (summary, error) {
	s := summary{Phase: "measured", EffectiveConcurrency: 1, Errors: map[string]int{}}
	if once {
		s.Phase = "preflight"
	}
	client, latency, err := openPhaseClient(address, scenario)
	s.ConnectionLatencyMilliseconds = latency
	if err != nil {
		s.FailedOperations++
		return s, err
	}
	defer client.close()
	started := time.Now()
	deadline := started.Add(duration)
	var next uint16 = 1
	for once || time.Now().Before(deadline) {
		begin := time.Now()
		p, n, retries, err := client.exchange(scenario, next)
		s.TotalTransferredBytes += n
		s.RetryCount += retries
		if err != nil {
			var malformed malformedError
			if errors.As(err, &malformed) {
				s.MalformedOperations++
			} else if isTimeout(err) {
				s.TimedOutOperations++
			} else {
				s.FailedOperations++
			}
			s.Errors[err.Error()]++
			return s, err
		}
		s.CompletedOperations++
		s.Latencies = append(s.Latencies, float64(time.Since(begin).Microseconds())/1000)
		s.LastProof = &p
		if once {
			break
		}
		next += 2
		if next == 0 {
			next = 1
		}
	}
	s.DurationSeconds = time.Since(started).Seconds()
	return s, nil
}

type phaseClient struct {
	udp *net.UDPConn
	tcp net.Conn
}

func openPhaseClient(address, scenario string) (phaseClient, float64, error) {
	started := time.Now()
	var c phaseClient
	var err error
	if scenario != "dns.classic.tcp.query.a" {
		remote, e := net.ResolveUDPAddr("udp", address)
		if e != nil {
			return c, 0, e
		}
		c.udp, err = net.DialUDP("udp", nil, remote)
		if err != nil {
			return c, 0, err
		}
	}
	if scenario != "dns.classic.udp.query.a" {
		c.tcp, err = net.DialTimeout("tcp", address, 5*time.Second)
		if err != nil {
			c.close()
			return c, 0, err
		}
	}
	return c, float64(time.Since(started).Microseconds()) / 1000, nil
}
func (c *phaseClient) close() {
	if c.udp != nil {
		_ = c.udp.Close()
	}
	if c.tcp != nil {
		_ = c.tcp.Close()
	}
}
func (c *phaseClient) exchange(scenario string, id uint16) (proof, int64, int, error) {
	switch scenario {
	case "dns.classic.udp.query.a":
		return exchangeUDP(c.udp, aQuery, aResponse, aFixtureID, aQueryHash, aResponseHash, id, false)
	case "dns.classic.tcp.query.a":
		return exchangeTCP(c.tcp, aQuery, aResponse, aFixtureID, aQueryHash, aResponseHash, id)
	case "dns.classic.udp-truncated-tcp-retry":
		p, n, _, err := exchangeUDP(c.udp, largeQuery, truncatedResponse, largeFixtureID, largeQueryHash, truncatedHash, id, true)
		if err != nil {
			return p, n, 0, err
		}
		newID := id + 1
		if newID == 0 {
			newID = 1
		}
		tcpProof, tcpBytes, _, err := exchangeTCP(c.tcp, largeQuery, largeResponse, largeFixtureID, largeQueryHash, largeResponseHash, newID)
		n += tcpBytes
		if err != nil {
			return p, n, 1, err
		}
		p.Transport = "udp-tcp"
		p.RetryCount = 1
		p.UDPAdvertisedPayloadBytes = 512
		p.UDPTruncatedResponseLength = 45
		p.UDPTruncated = true
		p.RetryQuestionIdentical = true
		p.RetryMessageIDNew = newID != id
		p.TCPResponseLength = tcpProof.ResponseLengthBytes
		p.TCPResponsePrefixHex = "0276"
		p.ResponseLengthBytes = tcpProof.ResponseLengthBytes
		p.ResponseNormalizedSHA256 = tcpProof.ResponseNormalizedSHA256
		p.ResponseWireHex = tcpProof.ResponseWireHex
		return p, n, 1, nil
	default:
		return proof{}, 0, 0, errors.New("unsupported scenario dispatch")
	}
}

func exchangeUDP(conn *net.UDPConn, queryTemplate, responseTemplate []byte, fixtureID, queryDigest, responseDigest string, id uint16, expectTC bool) (proof, int64, int, error) {
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	query := withID(queryTemplate, id)
	if _, err := conn.Write(query); err != nil {
		return proof{}, 0, 0, err
	}
	response := make([]byte, 65535)
	n, err := conn.Read(response)
	if err != nil {
		return proof{}, int64(len(query)), 0, err
	}
	response = response[:n]
	transferred := int64(len(query) + len(response))
	if len(response) != len(responseTemplate) {
		return proof{}, transferred, 0, malformedError{fmt.Sprintf("UDP response length %d", len(response))}
	}
	p, err := validate(query, response, fixtureID, "udp", "bare-dns-message-datagram", queryDigest, responseDigest, id)
	if err != nil {
		return p, transferred, 0, err
	}
	tc := response[2]&0x02 != 0
	if tc != expectTC {
		return p, transferred, 0, malformedError{"UDP TC flag mismatch"}
	}
	p.UDPTruncated = tc
	return p, transferred, 0, nil
}

func exchangeTCP(conn net.Conn, queryTemplate, responseTemplate []byte, fixtureID, queryDigest, responseDigest string, id uint16) (proof, int64, int, error) {
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	query := withID(queryTemplate, id)
	framed := frame(query)
	if _, err := conn.Write(framed); err != nil {
		return proof{}, 0, 0, err
	}
	prefix := make([]byte, 2)
	if _, err := io.ReadFull(conn, prefix); err != nil {
		return proof{}, int64(len(framed)), 0, err
	}
	length := int(binary.BigEndian.Uint16(prefix))
	response := make([]byte, length)
	if _, err := io.ReadFull(conn, response); err != nil {
		return proof{}, int64(len(framed) + 2), 0, err
	}
	transferred := int64(len(framed) + 2 + len(response))
	if len(response) != len(responseTemplate) {
		return proof{}, transferred, 0, malformedError{fmt.Sprintf("TCP response length %d", len(response))}
	}
	p, err := validate(query, response, fixtureID, "tcp", "two-octet-network-order-length-prefix", queryDigest, responseDigest, id)
	p.RequestWireHex = hex.EncodeToString(framed)
	p.ResponseWireHex = hex.EncodeToString(append(prefix, response...))
	return p, transferred, 0, err
}

func validate(query, response []byte, fixtureID, transport, framing, queryDigest, responseDigest string, id uint16) (proof, error) {
	if len(response) < 12 {
		return proof{}, malformedError{"short DNS response"}
	}
	responseID := binary.BigEndian.Uint16(response[:2])
	nq := withID(query, 0)
	nr := withID(response, 0)
	p := proof{FixtureID: fixtureID, Transport: transport, QuestionName: questionName(fixtureID), QuestionType: questionType(fixtureID), QuestionClass: "IN", Answer: answer(fixtureID), TTLSeconds: 0, ResponseCode: "NOERROR", Framing: framing, AuthoritativeAnswer: response[2]&0x04 != 0, RecursionDesired: false, RecursionAvailable: response[3]&0x80 != 0, ExternalUpstreamUsed: false, CacheEnabled: false, RuntimeMessageID: id, ResponseMessageID: responseID, MessageIDCorrelated: responseID == id, MessageIDUniqueAmongOutstanding: true, CanonicalHashNormalization: "set-message-id-to-zero", QueryLengthBytes: len(query), ResponseLengthBytes: len(response), QueryNormalizedSHA256: hash(nq), ResponseNormalizedSHA256: hash(nr), RequestWireHex: hex.EncodeToString(query), ResponseWireHex: hex.EncodeToString(response)}
	if p.QueryNormalizedSHA256 != queryDigest {
		return p, malformedError{"query hash mismatch"}
	}
	if !p.MessageIDCorrelated {
		return p, malformedError{"message ID mismatch"}
	}
	if p.ResponseNormalizedSHA256 != responseDigest {
		return p, malformedError{"response hash mismatch"}
	}
	if !p.AuthoritativeAnswer || p.RecursionAvailable {
		return p, malformedError{"authority flags mismatch"}
	}
	return p, nil
}

func validationDoc(cfg Config, scenario string, s summary, err error) map[string]any {
	passed := err == nil && s.CompletedOperations == 1 && !hasFailures(s)
	checks := []map[string]any{{"id": "exact-binding", "passed": passed}, {"id": "canonical-fixture", "passed": passed}, {"id": "message-id-correlation-and-normalization", "passed": passed}, {"id": "local-authority-no-recursion-cache-upstream", "passed": passed}, {"id": "zero-unexpected-outcomes", "passed": passed}}
	if scenario == "dns.classic.udp-truncated-tcp-retry" {
		checks = append(checks, map[string]any{"id": "udp-tc-single-identical-question-tcp-retry", "passed": passed})
	}
	return map[string]any{"schemaVersion": "protocol-lab.validation.v1", "scenarioId": scenario, "status": map[bool]string{true: "passed", false: "failed"}[passed], "executor": map[string]string{"id": cfg.ExecutorID, "version": Version}, "checks": checks, "completedOperations": s.CompletedOperations, "malformedOperations": s.MalformedOperations, "retryCount": s.RetryCount, "failedOperations": s.FailedOperations, "timedOutOperations": s.TimedOutOperations}
}

func normalize(cfg Config, scenario string, s summary, requested map[string]any) result {
	duration := s.DurationSeconds
	if duration <= 0 {
		duration = 1
	}
	m := metricSet{QueriesPerSecond: float64(s.CompletedOperations) / duration, QueryLatencyMean: mean(s.Latencies), QueryLatencyP50: percentile(s.Latencies, .5), QueryLatencyP75: percentile(s.Latencies, .75), QueryLatencyP90: percentile(s.Latencies, .90), QueryLatencyP95: percentile(s.Latencies, .95), QueryLatencyP99: percentile(s.Latencies, .99), ConnectionLatency: s.ConnectionLatencyMilliseconds, CompletedOperations: s.CompletedOperations, MalformedOperations: s.MalformedOperations, RetryCount: s.RetryCount, FailedOperations: s.FailedOperations, TimedOutOperations: s.TimedOutOperations, TotalTransferredBytes: s.TotalTransferredBytes, EffectiveConcurrency: 1}
	return result{SchemaVersion: "protocol-lab.dns-classic-executor-result.v1", ScenarioID: scenario, LoadProfileID: ProfileID, Status: "passed", Executor: map[string]string{"id": cfg.ExecutorID, "version": Version}, LoadGenerator: map[string]string{"id": cfg.LoadGeneratorID, "version": Version}, ProtocolProof: map[string]any{"requestedProtocol": protocolVariant(scenario), "observedProtocol": protocolVariant(scenario), "fallbackDetected": false, "dns": s.LastProof}, Validation: map[string]any{"status": "passed", "required": true}, RequestedLoad: requested, EffectiveLoad: map[string]any{"connections": 1, "concurrency": 1, "outstandingQueries": 1, "connectionReuse": "reuse", "durationSeconds": duration}, Metrics: m, Warnings: []string{"diagnostic-only; non-publishable classic DNS calibration"}}
}

func loadConfig() (map[string]any, error) {
	if strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID")) != ProfileID {
		return nil, fmt.Errorf("supports load profile %q only", ProfileID)
	}
	values := map[string]int{"connections": envInt("PLAB_CONNECTIONS"), "concurrency": envInt("PLAB_CONCURRENCY"), "durationSeconds": envInt("PLAB_DURATION_SECONDS"), "warmupSeconds": envInt("PLAB_WARMUP_SECONDS"), "repetition": envInt("PLAB_REPETITION")}
	if values["connections"] != 1 || values["concurrency"] != 1 || values["durationSeconds"] != 5 || values["warmupSeconds"] != 1 || values["repetition"] != 1 {
		return nil, fmt.Errorf("dns-classic-diagnostic requires 1/1/5/1/1: %v", values)
	}
	return map[string]any{"connections": 1, "concurrency": 1, "outstandingQueries": 1, "connectionReuse": "reuse", "durationSeconds": 5, "warmupSeconds": 1, "repetition": 1, "queryTimeoutMilliseconds": 5000, "maxRetries": 0}, nil
}

func normalizeTarget(value, mode string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("target address required")
	}
	if strings.Contains(value, "://") {
		u, err := url.Parse(value)
		if err != nil {
			return "", err
		}
		expected := "udp"
		if mode == "tcp" {
			expected = "tcp"
		}
		if u.Scheme != expected {
			return "", fmt.Errorf("%s executor rejects target scheme %q", mode, u.Scheme)
		}
		value = u.Host
	}
	if _, _, err := net.SplitHostPort(value); err != nil {
		return "", err
	}
	return value, nil
}
func protocolVariant(s string) string {
	switch s {
	case "dns.classic.udp.query.a":
		return "dns-udp"
	case "dns.classic.tcp.query.a":
		return "dns-tcp"
	default:
		return "dns-udp-tcp-retry"
	}
}
func questionName(f string) string {
	if f == aFixtureID {
		return "plab.test."
	}
	return "dnskey.plab.test."
}
func questionType(f string) string {
	if f == aFixtureID {
		return "A"
	}
	return "DNSKEY"
}
func answer(f string) string {
	if f == aFixtureID {
		return "192.0.2.1"
	}
	return "DNSKEY-and-RRSIG-shaped-record-set"
}
func verify(name, expected string) {
	if actual := strings.TrimSpace(os.Getenv(name)); actual != "" && actual != expected {
		fatal(2, fmt.Errorf("%s substitution rejected: expected %q observed %q", name, expected, actual))
	}
}
func envInt(name string) int {
	v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil {
		return 0
	}
	return v
}
func frame(m []byte) []byte {
	b := make([]byte, 2+len(m))
	binary.BigEndian.PutUint16(b[:2], uint16(len(m)))
	copy(b[2:], m)
	return b
}
func withID(m []byte, id uint16) []byte {
	b := append([]byte(nil), m...)
	binary.BigEndian.PutUint16(b[:2], id)
	return b
}
func hash(v []byte) string { s := sha256.Sum256(v); return fmt.Sprintf("%x", s) }
func mustHex(v string) []byte {
	b, e := hex.DecodeString(v)
	if e != nil {
		panic(e)
	}
	return b
}
func hasFailures(s summary) bool {
	return s.CompletedOperations == 0 || s.MalformedOperations != 0 || s.FailedOperations != 0 || s.TimedOutOperations != 0
}
func isTimeout(err error) bool { var n net.Error; return errors.As(err, &n) && n.Timeout() }
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
func percentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	i := int(math.Ceil(p*float64(len(c)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(c) {
		i = len(c) - 1
	}
	return c[i]
}
func write(root, name string, v any) {
	path := filepath.Join(root, name)
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal(2, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		fatal(2, err)
	}
}
func fatal(code int, err error) { fmt.Fprintln(os.Stderr, err); os.Exit(code) }
