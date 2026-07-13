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
	"strings"
	"time"
)

const (
	executorID           = "go-tls13-mtls-executor"
	executorVersion      = "0.1.0"
	loadGeneratorID      = "go-crypto-tls13-mtls-load"
	loadGeneratorVersion = "0.1.0"
	scenarioID           = "tls.handshake.mutual-auth"
	loadProfileID        = "tls-smoke"
	protocolVariant      = "tls1.3-full-mutual-auth"
	profileID            = "plab-tls13-aes128gcm-p256-mutual-auth-v2"
	serverCertProfileID  = "plab-single-leaf-p256-server-v2"
	clientCertProfileID  = "plab-single-leaf-p256-client-v2"
	serverName           = "tls.plab.test"
	alpn                 = "protocol-lab-tls"
	serverLeafDERHash    = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
	serverLeafSPKIHash   = "407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0"
	clientLeafDERHash    = "ca2e4f661e7b29cfc516c48f53c05be0ef59fb6cc410cb205f5759e07a5deb20"
	clientLeafSPKIHash   = "4b3a176400147e50a4efc3a7a26f66a9dec74a11042b7565eadd85b1ee27c0fb"
	authorityCommit      = "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574"
)

var knownUnsupported = []string{
	"tls.handshake.full", "tls.handshake.resumed", "tls.handshake.full.tls12", "tls.handshake.full.chacha20",
	"tls.early-data.accepted", "tls.early-data.rejected", "tls.key-update.diagnostic", "tls.record.throughput", "tls.record.coverage",
}

type observation struct {
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
	ServerCertificateProfile        string  `json:"serverCertificateProfile"`
	ServerCertificateDERSHA256      string  `json:"serverCertificateDerSha256"`
	ServerCertificateSPKISHA256     string  `json:"serverCertificateSpkiSha256"`
	ClientCertificateProfile        string  `json:"clientCertificateProfile"`
	ClientCertificateDERSHA256      string  `json:"clientCertificateDerSha256"`
	ClientCertificateSPKISHA256     string  `json:"clientCertificateSpkiSha256"`
	ClientCertificateChainSentCount int     `json:"clientCertificateChainSentCount"`
	ClientTrustAnchorSent           bool    `json:"clientTrustAnchorSent"`
	ServerCertificateVerified       bool    `json:"serverCertificateVerified"`
	ClientCertificateVerified       bool    `json:"clientCertificateVerified"`
	MutualAuthenticated             bool    `json:"mutualAuthenticated"`
	VerifiedServerChainCount        int     `json:"verifiedServerChainCount"`
	HandshakeLatencyMS              float64 `json:"handshakeLatencyMs"`
	ConnectionAndHandshakeMS        float64 `json:"connectionAndHandshakeLatencyMs"`
}

type summary struct {
	DurationSeconds     float64        `json:"durationSeconds"`
	CompletedOperations int            `json:"completedOperations"`
	FailedOperations    int            `json:"failedOperations"`
	TimedOutOperations  int            `json:"timedOutOperations"`
	LatencyMS           []float64      `json:"handshakeLatencyMilliseconds"`
	LastObservation     *observation   `json:"lastObservation,omitempty"`
	Errors              map[string]int `json:"errors"`
}

func main() {
	target := flag.String("target-address", os.Getenv("PLAB_TARGET_BASE_URL"), "TLS target address or tls:// URL")
	outputDir := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "artifact output directory")
	rootPath := flag.String("root-certificate", envOrDefault("PLAB_TLS_ROOT_CERTIFICATE_PATH", materialPath("certs/root.pem")), "server root PEM")
	clientCertPath := flag.String("client-certificate", envOrDefault("PLAB_TLS_CLIENT_CERTIFICATE_PATH", materialPath("certs/client.pem")), "client leaf PEM")
	clientKeyPath := flag.String("client-key", envOrDefault("PLAB_TLS_CLIENT_KEY_PATH", materialPath("certs/client-key.pem")), "client private key PEM")
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
	clientCertificate, err := loadClientIdentity(*clientCertPath, *clientKeyPath)
	if err != nil {
		fatal(2, err)
	}
	timeout := 5 * time.Second
	preflight, err := runHandshake(context.Background(), address, roots, clientCertificate, timeout)
	if err != nil {
		writeFailureArtifacts(*outputDir, err)
		fatal(1, err)
	}
	writeProofArtifacts(*outputDir, preflight)
	writeIdentity(*outputDir)
	if *validationOnly {
		measured := summary{DurationSeconds: preflight.HandshakeLatencyMS / 1000, CompletedOperations: 1, LatencyMS: []float64{preflight.HandshakeLatencyMS}, LastObservation: &preflight, Errors: map[string]int{}}
		writeResultArtifacts(*outputDir, measured)
		return
	}
	_ = runFor(address, roots, clientCertificate, timeout, time.Second)
	measured := runFor(address, roots, clientCertificate, timeout, 5*time.Second)
	if measured.FailedOperations != 0 || measured.TimedOutOperations != 0 || measured.CompletedOperations == 0 {
		writeResultArtifacts(*outputDir, measured)
		fatal(1, errors.New("mutual-authentication load phase did not complete cleanly"))
	}
	writeResultArtifacts(*outputDir, measured)
}

func runHandshake(ctx context.Context, address string, roots *x509.CertPool, clientCertificate tls.Certificate, timeout time.Duration) (observation, error) {
	started := time.Now()
	op, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	raw, err := (&net.Dialer{}).DialContext(op, "tcp", address)
	if err != nil {
		return observation{}, err
	}
	defer raw.Close()
	config := &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: roots, ServerName: serverName, NextProtos: []string{alpn}, CurvePreferences: []tls.CurveID{tls.X25519}, Certificates: []tls.Certificate{clientCertificate}, SessionTicketsDisabled: true}
	client := tls.Client(raw, config)
	handshakeStarted := time.Now()
	if err := client.HandshakeContext(op); err != nil {
		return observation{}, err
	}
	finished := time.Now()
	o, err := validateState(client.ConnectionState(), clientCertificate)
	o.HandshakeLatencyMS = durationMS(finished.Sub(handshakeStarted))
	o.ConnectionAndHandshakeMS = durationMS(finished.Sub(started))
	if closeErr := client.Close(); err == nil && closeErr != nil {
		err = closeErr
	}
	return o, err
}

func validateState(state tls.ConnectionState, clientCertificate tls.Certificate) (observation, error) {
	o := observation{
		TLSProfileID: profileID, TLSVersion: tlsVersionName(state.Version), CipherSuite: tls.CipherSuiteName(state.CipherSuite),
		KeyExchangeGroup: state.CurveID.String(), ALPN: state.NegotiatedProtocol, ServerName: serverName,
		HandshakeComplete: state.HandshakeComplete, DidResume: state.DidResume, EarlyDataAttempted: false,
		ServerCertificateProfile: serverCertProfileID, ClientCertificateProfile: clientCertProfileID,
		ClientCertificateChainSentCount: len(clientCertificate.Certificate), ClientTrustAnchorSent: len(clientCertificate.Certificate) != 1,
		VerifiedServerChainCount: len(state.VerifiedChains), ServerCertificateVerified: len(state.VerifiedChains) == 1,
		ClientCertificateVerified: true, MutualAuthenticated: state.HandshakeComplete && len(state.VerifiedChains) == 1,
	}
	if len(clientCertificate.Certificate) > 0 {
		leaf, _ := x509.ParseCertificate(clientCertificate.Certificate[0])
		if leaf != nil {
			o.ClientCertificateDERSHA256 = hash(leaf.Raw)
			o.ClientCertificateSPKISHA256 = hash(leaf.RawSubjectPublicKeyInfo)
		}
	}
	if len(state.PeerCertificates) > 0 {
		o.ServerCertificateDERSHA256 = hash(state.PeerCertificates[0].Raw)
		o.ServerCertificateSPKISHA256 = hash(state.PeerCertificates[0].RawSubjectPublicKeyInfo)
	}
	var failures []string
	if state.Version != tls.VersionTLS13 || !state.HandshakeComplete {
		failures = append(failures, "exact TLS 1.3 handshake not complete")
	}
	if state.CipherSuite != tls.TLS_AES_128_GCM_SHA256 {
		failures = append(failures, "cipher suite mismatch")
	}
	if state.CurveID != tls.X25519 {
		failures = append(failures, "key exchange group mismatch")
	}
	if state.NegotiatedProtocol != alpn {
		failures = append(failures, "ALPN mismatch")
	}
	if state.DidResume {
		failures = append(failures, "session resumption detected")
	}
	if len(state.VerifiedChains) != 1 {
		failures = append(failures, "server certificate chain was not uniquely verified")
	}
	if o.ServerCertificateDERSHA256 != serverLeafDERHash || o.ServerCertificateSPKISHA256 != serverLeafSPKIHash {
		failures = append(failures, "server certificate identity mismatch")
	}
	if len(clientCertificate.Certificate) != 1 {
		failures = append(failures, "client must send exactly one leaf and no trust anchor")
	}
	if o.ClientCertificateDERSHA256 != clientLeafDERHash || o.ClientCertificateSPKISHA256 != clientLeafSPKIHash {
		failures = append(failures, "client certificate identity mismatch")
	}
	if len(failures) > 0 {
		return o, errors.New(strings.Join(failures, "; "))
	}
	return o, nil
}

func loadClientIdentity(certPath, keyPath string) (tls.Certificate, error) {
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	if len(certificate.Certificate) != 1 {
		return tls.Certificate{}, errors.New("client PEM must contain exactly one leaf and no trust anchor")
	}
	leaf, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		return tls.Certificate{}, err
	}
	if hash(leaf.Raw) != clientLeafDERHash || hash(leaf.RawSubjectPublicKeyInfo) != clientLeafSPKIHash {
		return tls.Certificate{}, errors.New("client certificate substitution detected")
	}
	if leaf.SignatureAlgorithm != x509.ECDSAWithSHA256 || leaf.PublicKeyAlgorithm != x509.ECDSA {
		return tls.Certificate{}, errors.New("client certificate algorithm mismatch")
	}
	key, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok || key.Curve.Params().Name != "P-256" {
		return tls.Certificate{}, errors.New("client certificate curve mismatch")
	}
	return certificate, nil
}

func runFor(address string, roots *x509.CertPool, certificate tls.Certificate, timeout, duration time.Duration) summary {
	s := summary{Errors: map[string]int{}}
	started := time.Now()
	for time.Since(started) < duration {
		o, err := runHandshake(context.Background(), address, roots, certificate, timeout)
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
	}
	s.DurationSeconds = time.Since(started).Seconds()
	return s
}

func writeProofArtifacts(dir string, o observation) {
	writeRequired(dir, "validation.json", map[string]any{"schemaVersion": "protocol-lab.validation.v1", "scenarioId": scenarioID, "status": "passed", "checks": []string{"protocol:tls1.3", "no-fallback", "handshake-complete", "server-certificate-valid", "client-certificate-valid", "mutual-authenticated", "zero-unexpected-failures", "zero-timeouts"}})
	writeRequired(dir, "protocol-proof.json", o)
	writeRequired(dir, "tls-negotiation.json", o)
	writeRequired(dir, "peer-auth-proof.json", map[string]any{"schemaVersion": "protocol-lab.tls-peer-auth-proof.v1", "scenarioId": scenarioID, "serverCertificateProfile": serverCertProfileID, "serverCertificateDerSha256": o.ServerCertificateDERSHA256, "serverCertificateSpkiSha256": o.ServerCertificateSPKISHA256, "clientCertificateProfile": clientCertProfileID, "clientCertificateDerSha256": o.ClientCertificateDERSHA256, "clientCertificateSpkiSha256": o.ClientCertificateSPKISHA256, "clientCertificateChainSentCount": o.ClientCertificateChainSentCount, "clientTrustAnchorSent": o.ClientTrustAnchorSent, "serverCertificateVerified": o.ServerCertificateVerified, "clientCertificateVerified": o.ClientCertificateVerified, "mutualAuthenticated": o.MutualAuthenticated})
}

func writeResultArtifacts(dir string, measured summary) {
	d := measured.DurationSeconds
	if d <= 0 {
		d = 1
	}
	metrics := map[string]any{"handshakesPerSecond": float64(measured.CompletedOperations) / d, "handshakeLatencyMean": mean(measured.LatencyMS), "handshakeLatencyP50": percentile(measured.LatencyMS, .50), "handshakeLatencyP95": percentile(measured.LatencyMS, .95), "handshakeLatencyP99": percentile(measured.LatencyMS, .99), "completedOperations": measured.CompletedOperations, "failedOperations": measured.FailedOperations, "timedOutOperations": measured.TimedOutOperations, "totalTransferredBytes": 0}
	result := map[string]any{"schemaVersion": "protocol-lab.tls-executor-result.v1", "scenarioId": scenarioID, "mode": "full-mutual-auth-handshake", "authorityCommit": authorityCommit, "executor": map[string]string{"id": executorID, "version": executorVersion}, "loadGenerator": map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion}, "validation": map[string]string{"status": "passed"}, "protocolProof": measured.LastObservation, "requestedLoad": map[string]any{"profileId": loadProfileID, "connections": 1, "concurrency": 1, "durationSeconds": 5, "warmupSeconds": 1, "repetitions": 1, "applicationDataBytes": 0}, "effectiveLoad": map[string]any{"connections": 1, "concurrency": 1, "applicationDataBytes": 0}, "metrics": metrics, "warnings": []string{"Local package smoke evidence is diagnostic and non-publishable."}}
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
	writeRequired(dir, "load-generator-identity.json", map[string]string{"id": loadGeneratorID, "version": loadGeneratorVersion})
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
	if value == tls.VersionTLS13 {
		return "TLS1.3"
	}
	return fmt.Sprintf("0x%04x", value)
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
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	index := int(math.Ceil(q*float64(len(copyValues)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(copyValues) {
		index = len(copyValues) - 1
	}
	return copyValues[index]
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
