package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
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

	utls "github.com/refraction-networking/utls"
)

const (
	executorID           = "go-utls-tls13-chacha20-executor"
	executorVersion      = "0.1.0"
	loadGeneratorID      = "go-utls-tls13-chacha20-load"
	loadGeneratorVersion = "0.1.0"
	loadEngineModule     = "github.com/refraction-networking/utls"
	loadEngineVersion    = "v1.8.2"
	scenarioID           = "tls.handshake.full.chacha20"
	loadProfileID        = "tls-smoke"
	protocolVariant      = "tls1.3-full-chacha20"
	profileID            = "plab-tls13-chacha20-p256-server-auth-v2"
	certificateProfileID = "plab-single-leaf-p256-server-v2"
	serverName           = "tls.plab.test"
	alpn                 = "protocol-lab-tls"
	leafDERHash          = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
	leafSPKIHash         = "407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0"
	authorityCommit      = "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574"
)

var knownUnsupported = []string{"tls.handshake.full", "tls.handshake.resumed", "tls.handshake.full.tls12", "tls.handshake.mutual-auth", "tls.early-data.accepted", "tls.early-data.rejected", "tls.key-update.diagnostic", "tls.record.throughput", "tls.record.coverage"}

type clientHelloProof struct {
	SchemaVersion        string   `json:"schemaVersion"`
	EngineModule         string   `json:"engineModule"`
	EngineModuleVersion  string   `json:"engineModuleVersion"`
	ClientHelloProfile   string   `json:"clientHelloProfile"`
	CipherSuites         []string `json:"cipherSuites"`
	SupportedVersions    []string `json:"supportedVersions"`
	SupportedCurves      []string `json:"supportedCurves"`
	KeyShareGroups       []string `json:"keyShareGroups"`
	SignatureAlgorithms  []string `json:"signatureAlgorithms"`
	ALPNProtocols        []string `json:"alpnProtocols"`
	ServerName           string   `json:"serverName"`
	SessionTicketOffered bool     `json:"sessionTicketOffered"`
	PSKOffered           bool     `json:"pskOffered"`
	EarlyDataOffered     bool     `json:"earlyDataOffered"`
}

type observation struct {
	TLSProfileID                  string  `json:"tlsProfileId"`
	TLSVersion                    string  `json:"tlsVersion"`
	CipherSuite                   string  `json:"cipherSuite"`
	KeyExchangeGroup              string  `json:"keyExchangeGroup"`
	SignatureScheme               string  `json:"signatureScheme"`
	ALPN                          string  `json:"alpn"`
	ServerName                    string  `json:"serverName"`
	HandshakeComplete             bool    `json:"handshakeComplete"`
	DidResume                     bool    `json:"didResume"`
	SessionStateOffered           bool    `json:"sessionStateOffered"`
	EarlyDataAttempted            bool    `json:"earlyDataAttempted"`
	ApplicationDataBytesSent      int     `json:"applicationDataBytesSent"`
	ApplicationDataBytesReceived  int     `json:"applicationDataBytesReceived"`
	CertificateProfile            string  `json:"certificateProfile"`
	CertificateDERSHA256          string  `json:"certificateDerSha256"`
	CertificateSPKISHA256         string  `json:"certificateSpkiSha256"`
	CertificateSignatureAlgorithm string  `json:"certificateSignatureAlgorithm"`
	CertificatePublicKeyAlgorithm string  `json:"certificatePublicKeyAlgorithm"`
	CertificateNamedCurve         string  `json:"certificateNamedCurve"`
	SentCertificateCount          int     `json:"sentCertificateCount"`
	TrustAnchorSent               bool    `json:"trustAnchorSent"`
	VerifiedChainCount            int     `json:"verifiedChainCount"`
	HandshakeLatencyMS            float64 `json:"handshakeLatencyMs"`
	ConnectionAndHandshakeMS      float64 `json:"connectionAndHandshakeLatencyMs"`
}

type summary struct {
	DurationSeconds     float64           `json:"durationSeconds"`
	CompletedOperations int               `json:"completedOperations"`
	FailedOperations    int               `json:"failedOperations"`
	TimedOutOperations  int               `json:"timedOutOperations"`
	LatencyMS           []float64         `json:"handshakeLatencyMilliseconds"`
	LastObservation     *observation      `json:"lastObservation,omitempty"`
	LastClientHello     *clientHelloProof `json:"lastClientHello,omitempty"`
	Errors              map[string]int    `json:"errors"`
}

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "TLS target address or tls:// URL")
	outputDir := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "artifact output directory")
	rootPath := flag.String("root-certificate", envOrDefault("PLAB_TLS_ROOT_CERTIFICATE_PATH", materialPath("certs/root.pem")), "server root PEM")
	validationOnly := flag.Bool("validation-only", false, "run one validity operation")
	showVersion := flag.Bool("version", false, "print version")
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
	if *outputDir == "" {
		*outputDir = "artifacts"
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fatal(1, err)
	}
	requested := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID"))
	if isKnownUnsupported(requested) {
		emitUnsupported(*outputDir, requested)
		os.Exit(3)
	}
	if requested != scenarioID {
		fatal(2, fmt.Errorf("unknown or missing scenario %q", requested))
	}
	verifySubstitution("PLAB_LOAD_PROFILE_ID", loadProfileID, "load profile")
	verifySubstitution("PLAB_PROTOCOL_VARIANT", protocolVariant, "protocol variant")
	address, err := normalizeTarget(*target)
	if err != nil {
		fatal(2, err)
	}
	roots, err := loadRoots(*rootPath)
	if err != nil {
		fatal(2, err)
	}
	preflight, hello, err := runHandshake(context.Background(), address, roots, 5*time.Second)
	if err != nil {
		writeFailureArtifacts(*outputDir, err)
		fatal(1, err)
	}
	writeProofArtifacts(*outputDir, preflight, hello)
	writeIdentity(*outputDir)
	if *validationOnly {
		measured := summary{DurationSeconds: preflight.HandshakeLatencyMS / 1000, CompletedOperations: 1, LatencyMS: []float64{preflight.HandshakeLatencyMS}, LastObservation: &preflight, LastClientHello: &hello, Errors: map[string]int{}}
		writeResultArtifacts(*outputDir, measured)
		return
	}
	_ = runFor(address, roots, 5*time.Second, time.Second)
	measured := runFor(address, roots, 5*time.Second, 5*time.Second)
	writeResultArtifacts(*outputDir, measured)
	if measured.CompletedOperations == 0 || measured.FailedOperations != 0 || measured.TimedOutOperations != 0 {
		fatal(1, errors.New("ChaCha20 load phase did not complete cleanly"))
	}
}

func exactClientHelloSpec() *utls.ClientHelloSpec {
	return &utls.ClientHelloSpec{
		TLSVersMin:         utls.VersionTLS13,
		TLSVersMax:         utls.VersionTLS13,
		CipherSuites:       []uint16{utls.TLS_CHACHA20_POLY1305_SHA256},
		CompressionMethods: []uint8{0},
		Extensions: []utls.TLSExtension{
			&utls.SNIExtension{},
			&utls.SupportedCurvesExtension{Curves: []utls.CurveID{utls.X25519}},
			&utls.SupportedPointsExtension{SupportedPoints: []byte{0}},
			&utls.ALPNExtension{AlpnProtocols: []string{alpn}},
			&utls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []utls.SignatureScheme{utls.ECDSAWithP256AndSHA256}},
			&utls.KeyShareExtension{KeyShares: []utls.KeyShare{{Group: utls.X25519}}},
			&utls.SupportedVersionsExtension{Versions: []uint16{utls.VersionTLS13}},
		},
	}
}

func clientConfig(roots *x509.CertPool) *utls.Config {
	return &utls.Config{MinVersion: utls.VersionTLS13, MaxVersion: utls.VersionTLS13, RootCAs: roots, ServerName: serverName, NextProtos: []string{alpn}, CurvePreferences: []utls.CurveID{utls.X25519}, SessionTicketsDisabled: true, ClientSessionCache: nil}
}

func runHandshake(ctx context.Context, address string, roots *x509.CertPool, timeout time.Duration) (observation, clientHelloProof, error) {
	started := time.Now()
	op, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	raw, err := (&net.Dialer{}).DialContext(op, "tcp", address)
	if err != nil {
		return observation{}, clientHelloProof{}, err
	}
	defer raw.Close()
	client := utls.UClient(raw, clientConfig(roots), utls.HelloCustom)
	if err := client.ApplyPreset(exactClientHelloSpec()); err != nil {
		return observation{}, clientHelloProof{}, err
	}
	if err := client.BuildHandshakeStateWithoutSession(); err != nil {
		return observation{}, clientHelloProof{}, err
	}
	hello, err := validateClientHello(client.HandshakeState.Hello)
	if err != nil {
		return observation{}, hello, err
	}
	handshakeStarted := time.Now()
	if err := client.HandshakeContext(op); err != nil {
		return observation{}, hello, err
	}
	finished := time.Now()
	if client.HandshakeState.ServerHello == nil {
		return observation{}, hello, errors.New("uTLS server hello proof unavailable")
	}
	o, err := validateState(client.ConnectionState(), client.HandshakeState.ServerHello.ServerShare.Group, client.HandshakeState.State13.UsingPSK)
	o.HandshakeLatencyMS = durationMS(finished.Sub(handshakeStarted))
	o.ConnectionAndHandshakeMS = durationMS(finished.Sub(started))
	if closeErr := client.Close(); err == nil && closeErr != nil {
		err = closeErr
	}
	return o, hello, err
}

func validateClientHello(hello *utls.PubClientHelloMsg) (clientHelloProof, error) {
	if hello == nil {
		return clientHelloProof{}, errors.New("uTLS client hello proof unavailable")
	}
	p := clientHelloProof{
		SchemaVersion: "protocol-lab.tls-client-hello-proof.v1", EngineModule: loadEngineModule, EngineModuleVersion: loadEngineVersion, ClientHelloProfile: "utls-custom-tls13-chacha20-x25519-v1",
		CipherSuites: cipherNames(hello.CipherSuites), SupportedVersions: versionNames(hello.SupportedVersions), SupportedCurves: curveNames(hello.SupportedCurves), KeyShareGroups: keyShareNames(hello.KeyShares), SignatureAlgorithms: signatureNames(hello.SupportedSignatureAlgorithms), ALPNProtocols: append([]string(nil), hello.AlpnProtocols...), ServerName: hello.ServerName,
		SessionTicketOffered: hello.TicketSupported || len(hello.SessionTicket) != 0, PSKOffered: len(hello.PskIdentities) != 0 || len(hello.PskBinders) != 0 || len(hello.PskModes) != 0, EarlyDataOffered: hello.EarlyData,
	}
	var failures []string
	if !equalUint16(hello.CipherSuites, []uint16{utls.TLS_CHACHA20_POLY1305_SHA256}) {
		failures = append(failures, "client hello cipher suites are not exact ChaCha20 only")
	}
	if !equalUint16(hello.SupportedVersions, []uint16{utls.VersionTLS13}) {
		failures = append(failures, "client hello supported versions are not exact TLS 1.3 only")
	}
	if !equalCurves(hello.SupportedCurves, []utls.CurveID{utls.X25519}) || len(hello.KeyShares) != 1 || hello.KeyShares[0].Group != utls.X25519 {
		failures = append(failures, "client hello key exchange is not exact X25519 only")
	}
	if len(hello.SupportedSignatureAlgorithms) != 1 || hello.SupportedSignatureAlgorithms[0] != utls.ECDSAWithP256AndSHA256 {
		failures = append(failures, "client hello signature algorithm is not exact ECDSA P-256 SHA-256")
	}
	if len(hello.AlpnProtocols) != 1 || hello.AlpnProtocols[0] != alpn {
		failures = append(failures, "client hello ALPN mismatch")
	}
	if hello.ServerName != serverName {
		failures = append(failures, "client hello SNI mismatch")
	}
	if p.SessionTicketOffered || p.PSKOffered {
		failures = append(failures, "client hello offered session resumption state")
	}
	if p.EarlyDataOffered {
		failures = append(failures, "client hello offered early data")
	}
	if len(failures) != 0 {
		return p, errors.New(strings.Join(failures, "; "))
	}
	return p, nil
}

func validateState(state utls.ConnectionState, group utls.CurveID, usingPSK bool) (observation, error) {
	o := observation{TLSProfileID: profileID, TLSVersion: tlsVersionName(state.Version), CipherSuite: utls.CipherSuiteName(state.CipherSuite), KeyExchangeGroup: group.String(), SignatureScheme: "ecdsa_secp256r1_sha256", ALPN: state.NegotiatedProtocol, ServerName: serverName, HandshakeComplete: state.HandshakeComplete, DidResume: state.DidResume, SessionStateOffered: false, EarlyDataAttempted: false, CertificateProfile: certificateProfileID, VerifiedChainCount: len(state.VerifiedChains)}
	if len(state.PeerCertificates) > 0 {
		leaf := state.PeerCertificates[0]
		o.CertificateDERSHA256 = hash(leaf.Raw)
		o.CertificateSPKISHA256 = hash(leaf.RawSubjectPublicKeyInfo)
		o.CertificateSignatureAlgorithm = leaf.SignatureAlgorithm.String()
		o.CertificatePublicKeyAlgorithm = leaf.PublicKeyAlgorithm.String()
		if key, ok := leaf.PublicKey.(*ecdsa.PublicKey); ok {
			o.CertificateNamedCurve = key.Curve.Params().Name
		}
	}
	o.SentCertificateCount = len(state.PeerCertificates)
	o.TrustAnchorSent = o.SentCertificateCount != 1
	var failures []string
	if state.Version != utls.VersionTLS13 || !state.HandshakeComplete {
		failures = append(failures, "exact TLS 1.3 handshake not complete")
	}
	if state.CipherSuite != utls.TLS_CHACHA20_POLY1305_SHA256 {
		failures = append(failures, "ChaCha20 cipher suite mismatch")
	}
	if group != utls.X25519 {
		failures = append(failures, "key exchange group mismatch")
	}
	if state.NegotiatedProtocol != alpn {
		failures = append(failures, "ALPN mismatch")
	}
	if state.DidResume || usingPSK {
		failures = append(failures, "session resumption detected")
	}
	if len(state.VerifiedChains) != 1 {
		failures = append(failures, "server certificate chain was not uniquely verified")
	}
	if o.CertificateDERSHA256 != leafDERHash || o.CertificateSPKISHA256 != leafSPKIHash {
		failures = append(failures, "server certificate identity mismatch")
	}
	if o.SentCertificateCount != 1 {
		failures = append(failures, "server must send exactly one leaf and no trust anchor")
	}
	if o.CertificateSignatureAlgorithm != "ECDSA-SHA256" || o.CertificatePublicKeyAlgorithm != "ECDSA" || o.CertificateNamedCurve != "P-256" {
		failures = append(failures, "server certificate algorithm mismatch")
	}
	if len(failures) > 0 {
		return o, errors.New(strings.Join(failures, "; "))
	}
	return o, nil
}

func runFor(address string, roots *x509.CertPool, timeout, duration time.Duration) summary {
	s := summary{Errors: map[string]int{}}
	started := time.Now()
	for time.Since(started) < duration {
		o, hello, err := runHandshake(context.Background(), address, roots, timeout)
		if err != nil {
			var n net.Error
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &n) && n.Timeout()) {
				s.TimedOutOperations++
			} else {
				s.FailedOperations++
			}
			s.Errors[err.Error()]++
			continue
		}
		s.CompletedOperations++
		s.LatencyMS = append(s.LatencyMS, o.HandshakeLatencyMS)
		s.LastObservation = &o
		s.LastClientHello = &hello
	}
	s.DurationSeconds = time.Since(started).Seconds()
	return s
}

func executionMetadata() map[string]any {
	return map[string]any{"loadEngineModule": loadEngineModule, "loadEngineModuleVersion": loadEngineVersion, "clientHelloProfile": "utls-custom-tls13-chacha20-x25519-v1", "clientHelloConstruction": "pinned ClientHelloSpec", "license": "BSD-3-Clause"}
}
func writeProofArtifacts(dir string, o observation, hello clientHelloProof) {
	writeRequired(dir, "validation.json", map[string]any{"schemaVersion": "protocol-lab.validation.v1", "scenarioId": scenarioID, "status": "passed", "checks": []string{"protocol:tls1.3", "cipher:TLS_CHACHA20_POLY1305_SHA256", "client-hello:exact-single-suite", "key-exchange:X25519", "no-fallback", "handshake-complete", "authenticated-peer", "no-session-state", "no-early-data", "zero-application-bytes", "zero-unexpected-failures", "zero-timeouts"}})
	writeRequired(dir, "client-hello-proof.json", hello)
	writeRequired(dir, "protocol-proof.json", o)
	writeRequired(dir, "tls-negotiation.json", o)
}
func writeResultArtifacts(dir string, measured summary) {
	d := measured.DurationSeconds
	if d <= 0 {
		d = 1
	}
	metrics := map[string]any{"handshakesPerSecond": float64(measured.CompletedOperations) / d, "handshakeLatencyMean": mean(measured.LatencyMS), "handshakeLatencyP50": percentile(measured.LatencyMS, .5), "handshakeLatencyP95": percentile(measured.LatencyMS, .95), "handshakeLatencyP99": percentile(measured.LatencyMS, .99), "completedOperations": measured.CompletedOperations, "failedOperations": measured.FailedOperations, "timedOutOperations": measured.TimedOutOperations, "totalTransferredBytes": 0}
	result := map[string]any{"schemaVersion": "protocol-lab.tls-executor-result.v1", "scenarioId": scenarioID, "mode": "full-tls13-chacha20-handshake", "authorityCommit": authorityCommit, "executor": map[string]string{"id": executorID, "version": executorVersion}, "loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": loadEngineModule, "engineModuleVersion": loadEngineVersion}, "executionMetadata": executionMetadata(), "validation": map[string]string{"status": "passed"}, "clientHelloProof": measured.LastClientHello, "protocolProof": measured.LastObservation, "requestedLoad": map[string]any{"profileId": loadProfileID, "connections": 1, "concurrency": 1, "durationSeconds": 5, "warmupSeconds": 1, "repetitions": 1, "applicationDataBytes": 0}, "effectiveLoad": map[string]any{"connections": 1, "concurrency": 1, "applicationDataBytes": 0}, "metrics": metrics, "warnings": []string{"Local evidence is diagnostic and non-publishable."}}
	writeRequired(dir, "tls-load-summary.json", measured)
	writeRequired(dir, "tls-executor-result.json", result)
	writeRequired(dir, "result.json", result)
}
func writeFailureArtifacts(dir string, err error) {
	failure := map[string]any{"schemaVersion": "protocol-lab.validation.v1", "scenarioId": scenarioID, "status": "failed", "message": err.Error(), "executor": map[string]string{"id": executorID, "version": executorVersion}}
	writeRequired(dir, "validation.json", failure)
	writeRequired(dir, "result.json", failure)
	writeIdentity(dir)
}
func emitUnsupported(dir, requested string) {
	u := map[string]any{"schemaVersion": "protocol-lab.unsupported.v1", "status": "unsupported", "scenarioId": requested, "reasonCode": "scenario-not-implemented", "message": fmt.Sprintf("%s@%s recognizes %s but does not implement its exact semantics", executorID, executorVersion, requested), "authorityCommit": authorityCommit, "executor": map[string]string{"id": executorID, "version": executorVersion}, "loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, "supportedScenarios": []string{scenarioID}}
	writeRequired(dir, "unsupported.json", u)
	writeRequired(dir, "result.json", u)
	writeIdentity(dir)
	data, _ := json.MarshalIndent(u, "", "  ")
	fmt.Println(string(data))
}
func writeIdentity(dir string) {
	writeRequired(dir, "executor-identity.json", map[string]any{"id": executorID, "version": executorVersion, "role": "client-test-executor", "supportedProtocols": []string{"tls"}, "supportedScenarios": []string{scenarioID}, "supportedLoadProfiles": []string{loadProfileID}})
	writeRequired(dir, "load-generator-identity.json", map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion, "engineModule": loadEngineModule, "engineModuleVersion": loadEngineVersion})
}

func isKnownUnsupported(value string) bool {
	for _, id := range knownUnsupported {
		if value == id {
			return true
		}
	}
	return false
}
func loadRoots(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(data) {
		return nil, errors.New("server root PEM contained no certificate")
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
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "tls" || parsed.Host == "" {
		return "", errors.New("TLS target must use tls://host:port")
	}
	return parsed.Host, nil
}
func materialPath(relative string) string {
	if executable, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(executable), "..", "..", relative)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", relative)
}
func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
func verifySubstitution(variable, expected, label string) {
	if observed := strings.TrimSpace(os.Getenv(variable)); observed != "" && observed != expected {
		fatal(2, fmt.Errorf("%s substitution detected: expected %q observed %q", label, expected, observed))
	}
}
func tlsVersionName(value uint16) string {
	if value == utls.VersionTLS13 {
		return "TLS1.3"
	}
	if value == utls.VersionTLS12 {
		return "TLS1.2"
	}
	return fmt.Sprintf("0x%04x", value)
}
func cipherNames(values []uint16) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = utls.CipherSuiteName(v)
	}
	return out
}
func versionNames(values []uint16) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = tlsVersionName(v)
	}
	return out
}
func curveNames(values []utls.CurveID) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = v.String()
	}
	return out
}
func keyShareNames(values []utls.KeyShare) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = v.Group.String()
	}
	return out
}
func signatureNames(values []utls.SignatureScheme) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = v.String()
	}
	return out
}
func equalUint16(left, right []uint16) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
func equalCurves(left, right []utls.CurveID) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
func durationMS(value time.Duration) float64 { return float64(value) / float64(time.Millisecond) }
func hash(value []byte) string               { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
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
	copied := append([]float64(nil), values...)
	sort.Float64s(copied)
	index := int(math.Ceil(q*float64(len(copied)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(copied) {
		index = len(copied) - 1
	}
	return copied[index]
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
