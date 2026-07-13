package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	executorID                = "go-http2-executor"
	executorVersion           = "0.3.0"
	loadGeneratorID           = "go-x-net-http2-h2c-load"
	loadGeneratorVersion      = "0.3.0"
	http2EngineModule         = "golang.org/x/net/http2"
	http2EngineModuleVersion  = "v0.57.0"
	requestedFamily           = "h2"
	requestedVersion          = "HTTP/2.0"
	requestedExecutionVariant = "http2-h2c-prior-knowledge"
)

type scenarioExpectation struct {
	ScenarioID          string
	Path                string
	ExpectedStatus      int
	ExpectedContentType string
	ExpectedBody        []byte
}

type checkResult struct {
	ScenarioID            string `json:"scenarioId"`
	Method                string `json:"method"`
	Path                  string `json:"path"`
	Passed                bool   `json:"passed"`
	ExpectedStatus        int    `json:"expectedStatus"`
	ObservedStatus        int    `json:"observedStatus"`
	ExpectedContentType   string `json:"expectedContentType"`
	ObservedContentType   string `json:"observedContentType,omitempty"`
	ExpectedPayloadLength int    `json:"expectedPayloadLength"`
	ObservedPayloadLength int    `json:"observedPayloadLength"`
	ExpectedPayloadSHA256 string `json:"expectedPayloadSha256"`
	ObservedPayloadSHA256 string `json:"observedPayloadSha256,omitempty"`
	RequestedVersion      string `json:"requestedProtocolVersion"`
	ObservedVersion       string `json:"observedProtocolVersion,omitempty"`
	FallbackDetected      bool   `json:"fallbackDetected"`
	TimedOut              bool   `json:"timedOut"`
	Error                 string `json:"error,omitempty"`
}

type validationResult struct {
	ExecutorID              string        `json:"executorId"`
	ExecutorVersion         string        `json:"executorVersion"`
	TargetURL               string        `json:"targetUrl"`
	RequestedProtocol       string        `json:"requestedProtocol"`
	RequestedVersion        string        `json:"requestedProtocolVersion"`
	ExecutionVariant        string        `json:"executionVariant"`
	ObservedVersions        []string      `json:"observedProtocolVersions"`
	ObservedConnectionCount int           `json:"observedConnectionCount"`
	FallbackDetected        bool          `json:"fallbackDetected"`
	StartedAtUTC            string        `json:"startedAtUtc"`
	DurationMS              int64         `json:"durationMs"`
	Passed                  bool          `json:"passed"`
	UnexpectedFailureCount  int           `json:"unexpectedFailureCount"`
	TimeoutCount            int           `json:"timeoutCount"`
	Checks                  []checkResult `json:"checks"`
}

type protocolProof struct {
	ExecutorID              string        `json:"executorId"`
	ExecutorVersion         string        `json:"executorVersion"`
	RequestedProtocol       string        `json:"requestedProtocol"`
	RequestedVersion        string        `json:"requestedProtocolVersion"`
	ExecutionVariant        string        `json:"executionVariant"`
	ObservedVersions        []string      `json:"observedProtocolVersions"`
	ObservedConnectionCount int           `json:"observedConnectionCount"`
	FallbackDetected        bool          `json:"fallbackDetected"`
	Passed                  bool          `json:"passed"`
	Checks                  []checkResult `json:"checks"`
}

type executorIdentity struct {
	ExecutorID              string   `json:"executorId"`
	ExecutorVersion         string   `json:"executorVersion"`
	Role                    string   `json:"role"`
	SupportedProtocols      []string `json:"supportedProtocols"`
	SupportedVariants       []string `json:"supportedExecutionVariants"`
	SupportedScenarios      []string `json:"supportedScenarios"`
	SupportedLoadProfiles   []string `json:"supportedLoadProfiles"`
	LoadGenerationSupported bool     `json:"loadGenerationSupported"`
	StdoutStderrOwner       string   `json:"stdoutStderrOwner"`
}

type roundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

func main() {
	targetURL := flag.String("target-url", os.Getenv("PLAB_TARGET_BASE_URL"), "Target base URL.")
	outputDir := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "Artifact output directory.")
	timeout := flag.Duration("timeout", 10*time.Second, "Per-request timeout.")
	validationOnly := flag.Bool("validation-only", false, "Run the exact HTTP/2 h2c validity gate without load generation.")
	showVersion := flag.Bool("version", false, "Print executor version and exit.")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s %s\n", executorID, executorVersion)
		return
	}

	if strings.TrimSpace(*targetURL) == "" {
		fatal(2, "target-url or PLAB_TARGET_BASE_URL is required")
	}
	if *timeout <= 0 {
		fatal(2, "timeout must be greater than zero")
	}
	if strings.TrimSpace(*outputDir) == "" {
		*outputDir = "artifacts"
	}
	if expected := strings.TrimSpace(os.Getenv("PLAB_EXECUTOR_ID")); expected != "" && expected != executorID {
		fatal(2, fmt.Sprintf("executor substitution detected: expected %q, running %q", expected, executorID))
	}
	if expected := strings.TrimSpace(os.Getenv("PLAB_EXECUTOR_VERSION")); expected != "" && expected != executorVersion {
		fatal(2, fmt.Sprintf("executor version substitution detected: expected %q, running %q", expected, executorVersion))
	}
	if requested := strings.TrimSpace(os.Getenv("PLAB_PROTOCOL")); requested != "" && requested != requestedFamily {
		fatal(2, fmt.Sprintf("protocol substitution detected: expected %q, requested %q", requestedFamily, requested))
	}
	if variant := strings.TrimSpace(os.Getenv("PLAB_PROTOCOL_VARIANT")); variant != "" && variant != requestedExecutionVariant {
		fatal(2, fmt.Sprintf("execution variant %q is unsupported; expected %q", variant, requestedExecutionVariant))
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fatal(1, err.Error())
	}

	expectations, err := selectedExpectations(strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")))
	if err != nil {
		fatal(2, err.Error())
	}
	started := time.Now().UTC()
	connection, err := dialH2C(context.Background(), *targetURL, *timeout)
	checks := make([]checkResult, 0, len(expectations))
	observedConnections := 0
	if err != nil {
		for _, expectation := range expectations {
			checks = append(checks, failedDialCheck(expectation, err))
		}
	} else {
		observedConnections = 1
		defer connection.Close()
		for _, expectation := range expectations {
			checks = append(checks, runCheck(context.Background(), connection.client, *targetURL, expectation, *timeout))
		}
	}

	passed, fallbackDetected, unexpectedFailures, timeouts, observedVersions := summarizeChecks(checks)
	passed = passed && observedConnections == 1
	if observedConnections != 1 {
		unexpectedFailures++
	}
	result := validationResult{
		ExecutorID: executorID, ExecutorVersion: executorVersion, TargetURL: *targetURL,
		RequestedProtocol: requestedFamily, RequestedVersion: requestedVersion,
		ExecutionVariant: requestedExecutionVariant, ObservedVersions: observedVersions,
		ObservedConnectionCount: observedConnections, FallbackDetected: fallbackDetected,
		StartedAtUTC: started.Format(time.RFC3339Nano), DurationMS: time.Since(started).Milliseconds(),
		Passed: passed, UnexpectedFailureCount: unexpectedFailures, TimeoutCount: timeouts, Checks: checks,
	}
	artifacts := []struct {
		name  string
		value any
	}{
		{"validation.json", result},
		{"result.json", result},
		{"protocol-proof.json", protocolProof{
			ExecutorID: executorID, ExecutorVersion: executorVersion,
			RequestedProtocol: requestedFamily, RequestedVersion: requestedVersion,
			ExecutionVariant: requestedExecutionVariant, ObservedVersions: observedVersions,
			ObservedConnectionCount: observedConnections, FallbackDetected: fallbackDetected,
			Passed: passed, Checks: checks,
		}},
		{"executor-identity.json", executorIdentity{
			ExecutorID: executorID, ExecutorVersion: executorVersion, Role: "client-test-executor",
			SupportedProtocols: []string{requestedFamily}, SupportedVariants: []string{requestedExecutionVariant},
			SupportedScenarios:    []string{"http2.core.plaintext", "http2.core.json"},
			SupportedLoadProfiles: []string{"http2-smoke", "http2-diagnostic", "http2-comparison"}, LoadGenerationSupported: true,
			StdoutStderrOwner: "invoking-runner-or-package-host",
		}},
	}
	for _, artifact := range artifacts {
		if err := writeJSON(filepath.Join(*outputDir, artifact.name), artifact.value); err != nil {
			fatal(1, err.Error())
		}
	}

	if !passed {
		fatal(1, "go-http2-executor validation failed")
	}
	if *validationOnly {
		fmt.Fprintln(os.Stderr, "go-http2-executor validation passed with exact HTTP/2 h2c prior-knowledge proof")
		return
	}
	config, err := loadConfigFromEnvironment()
	if err != nil {
		fatal(2, err.Error())
	}
	expectation, err := findExpectation(config.ScenarioID)
	if err != nil {
		fatal(2, err.Error())
	}
	loadResult, err := runH2CLoad(*targetURL, *outputDir, expectation, config)
	if err != nil {
		fatal(1, err.Error())
	}
	if err := writeJSON(filepath.Join(*outputDir, "load-generator-identity.json"), loadResult.LoadGenerator); err != nil {
		fatal(1, err.Error())
	}
	if err := writeExecutorResult(*outputDir, loadResult); err != nil {
		fatal(1, err.Error())
	}
}

func defaultExpectations() []scenarioExpectation {
	return []scenarioExpectation{
		{ScenarioID: "http2.core.plaintext", Path: "/plaintext", ExpectedStatus: http.StatusOK, ExpectedContentType: "text/plain", ExpectedBody: []byte("Hello, World!")},
		{ScenarioID: "http2.core.json", Path: "/json", ExpectedStatus: http.StatusOK, ExpectedContentType: "application/json", ExpectedBody: []byte(`{"message":"Hello, World!"}`)},
	}
}

func selectedExpectations(scenarioID string) ([]scenarioExpectation, error) {
	if scenarioID == "" {
		return defaultExpectations(), nil
	}
	expectation, err := findExpectation(scenarioID)
	if err != nil {
		return nil, err
	}
	return []scenarioExpectation{expectation}, nil
}

func findExpectation(scenarioID string) (scenarioExpectation, error) {
	for _, expectation := range defaultExpectations() {
		if expectation.ScenarioID == scenarioID {
			return expectation, nil
		}
	}
	return scenarioExpectation{}, fmt.Errorf("scenario %q is unsupported by %s", scenarioID, executorID)
}

func failedDialCheck(expectation scenarioExpectation, err error) checkResult {
	return checkResult{
		ScenarioID: expectation.ScenarioID, Method: http.MethodGet, Path: expectation.Path,
		ExpectedStatus: expectation.ExpectedStatus, ExpectedContentType: expectation.ExpectedContentType,
		ExpectedPayloadLength: len(expectation.ExpectedBody), ExpectedPayloadSHA256: sha256Hex(expectation.ExpectedBody),
		RequestedVersion: requestedVersion, TimedOut: isTimeout(err), Error: err.Error(),
	}
}

func runCheck(ctx context.Context, client roundTripper, baseURL string, expectation scenarioExpectation, timeout time.Duration) checkResult {
	expectedHash := sha256Hex(expectation.ExpectedBody)
	result := checkResult{
		ScenarioID: expectation.ScenarioID, Method: http.MethodGet, Path: expectation.Path,
		ExpectedStatus: expectation.ExpectedStatus, ExpectedContentType: expectation.ExpectedContentType,
		ExpectedPayloadLength: len(expectation.ExpectedBody), ExpectedPayloadSHA256: expectedHash,
		RequestedVersion: requestedVersion,
	}
	requestURL, err := joinURL(baseURL, expectation.Path)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	requestContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestContext, http.MethodGet, requestURL, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Proto, req.ProtoMajor, req.ProtoMinor = requestedVersion, 2, 0
	req.Header.Set("User-Agent", executorID+"/"+executorVersion)
	req.Header.Set("Accept-Encoding", "identity")
	resp, err := client.RoundTrip(req)
	if err != nil {
		result.TimedOut = isTimeout(err) || errors.Is(err, context.DeadlineExceeded)
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	result.ObservedStatus = resp.StatusCode
	result.ObservedContentType = resp.Header.Get("Content-Type")
	result.ObservedVersion = resp.Proto
	result.FallbackDetected = !isExactHTTP2(resp.ProtoMajor, resp.ProtoMinor)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.ObservedPayloadLength = len(body)
	result.ObservedPayloadSHA256 = sha256Hex(body)
	problems := make([]string, 0, 5)
	if result.FallbackDetected {
		problems = append(problems, fmt.Sprintf("expected exact HTTP/2, observed %s", resp.Proto))
	}
	if resp.StatusCode != expectation.ExpectedStatus {
		problems = append(problems, fmt.Sprintf("expected status %d, observed %d", expectation.ExpectedStatus, resp.StatusCode))
	}
	if !mediaTypeMatches(result.ObservedContentType, expectation.ExpectedContentType) {
		problems = append(problems, fmt.Sprintf("expected content type %q, observed %q", expectation.ExpectedContentType, result.ObservedContentType))
	}
	if len(body) != len(expectation.ExpectedBody) {
		problems = append(problems, fmt.Sprintf("expected payload length %d, observed %d", len(expectation.ExpectedBody), len(body)))
	}
	if !bytes.Equal(body, expectation.ExpectedBody) {
		problems = append(problems, fmt.Sprintf("expected payload SHA-256 %s, observed %s", expectedHash, result.ObservedPayloadSHA256))
	}
	result.Passed = len(problems) == 0
	result.Error = strings.Join(problems, "; ")
	return result
}

func summarizeChecks(checks []checkResult) (bool, bool, int, int, []string) {
	passed := len(checks) > 0
	fallbackDetected, unexpectedFailures, timeouts := false, 0, 0
	versions := map[string]struct{}{}
	for _, check := range checks {
		if !check.Passed {
			passed = false
			unexpectedFailures++
		}
		fallbackDetected = fallbackDetected || check.FallbackDetected
		if check.TimedOut {
			timeouts++
		}
		if check.ObservedVersion != "" {
			versions[check.ObservedVersion] = struct{}{}
		}
	}
	observed := make([]string, 0, len(versions))
	for version := range versions {
		observed = append(observed, version)
	}
	sort.Strings(observed)
	return passed, fallbackDetected, unexpectedFailures, timeouts, observed
}

func joinURL(baseURL, requestPath string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("target-url must include scheme and host")
	}
	if !strings.EqualFold(parsed.Scheme, "http") {
		return "", errors.New("go-http2-executor 0.3.0 supports cleartext HTTP/2 h2c targets only")
	}
	parsed.Path, parsed.RawQuery, parsed.Fragment = requestPath, "", ""
	return parsed.String(), nil
}

func mediaTypeMatches(observed, expected string) bool {
	observed = strings.TrimSpace(strings.SplitN(observed, ";", 2)[0])
	return strings.EqualFold(observed, expected)
}

func isExactHTTP2(major, minor int) bool { return major == 2 && minor == 0 }

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func sha256Hex(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

func writeJSON(filePath string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, append(data, '\n'), 0o644)
}

func fatal(code int, message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(code)
}
