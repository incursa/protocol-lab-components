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
	executorID       = "go-http1-executor"
	executorVersion  = "0.3.0"
	requestedFamily  = "h1"
	requestedVersion = "HTTP/1.1"
)

type scenarioExpectation struct {
	ScenarioID          string
	Path                string
	ExpectedStatus      int
	ExpectedContentType string
	ExpectedBody        []byte
}

type validationResult struct {
	ExecutorID             string        `json:"executorId"`
	ExecutorVersion        string        `json:"executorVersion"`
	TargetURL              string        `json:"targetUrl"`
	RequestedProtocol      string        `json:"requestedProtocol"`
	RequestedVersion       string        `json:"requestedProtocolVersion"`
	ObservedVersions       []string      `json:"observedProtocolVersions"`
	FallbackDetected       bool          `json:"fallbackDetected"`
	StartedAtUTC           string        `json:"startedAtUtc"`
	DurationMS             int64         `json:"durationMs"`
	Passed                 bool          `json:"passed"`
	UnexpectedFailureCount int           `json:"unexpectedFailureCount"`
	TimeoutCount           int           `json:"timeoutCount"`
	Checks                 []checkResult `json:"checks"`
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

type protocolProof struct {
	ExecutorID        string        `json:"executorId"`
	ExecutorVersion   string        `json:"executorVersion"`
	RequestedProtocol string        `json:"requestedProtocol"`
	RequestedVersion  string        `json:"requestedProtocolVersion"`
	ObservedVersions  []string      `json:"observedProtocolVersions"`
	FallbackDetected  bool          `json:"fallbackDetected"`
	Passed            bool          `json:"passed"`
	Checks            []checkResult `json:"checks"`
}

type executorIdentity struct {
	ExecutorID              string   `json:"executorId"`
	ExecutorVersion         string   `json:"executorVersion"`
	Role                    string   `json:"role"`
	SupportedProtocols      []string `json:"supportedProtocols"`
	SupportedScenarios      []string `json:"supportedScenarios"`
	LoadGenerationSupported bool     `json:"loadGenerationSupported"`
	StdoutStderrOwner       string   `json:"stdoutStderrOwner"`
}

func main() {
	targetURL := flag.String("target-url", os.Getenv("PLAB_TARGET_BASE_URL"), "Target base URL.")
	outputDir := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "Artifact output directory.")
	timeout := flag.Duration("timeout", 10*time.Second, "Per-request timeout.")
	validationOnly := flag.Bool("validation-only", false, "Run the exact HTTP/1.1 validity gate without load generation.")
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

	started := time.Now().UTC()
	client := newHTTP1Client(*timeout)
	ctx := context.Background()
	expectations := defaultExpectations()
	checks := make([]checkResult, 0, len(expectations))
	for _, expectation := range expectations {
		checks = append(checks, runCheck(ctx, client, *targetURL, expectation))
	}

	passed := true
	fallbackDetected := false
	unexpectedFailures := 0
	timeouts := 0
	observedVersionSet := map[string]struct{}{}
	for _, check := range checks {
		if !check.Passed {
			passed = false
			unexpectedFailures++
		}
		if check.FallbackDetected {
			fallbackDetected = true
		}
		if check.TimedOut {
			timeouts++
		}
		if check.ObservedVersion != "" {
			observedVersionSet[check.ObservedVersion] = struct{}{}
		}
	}
	observedVersions := make([]string, 0, len(observedVersionSet))
	for version := range observedVersionSet {
		observedVersions = append(observedVersions, version)
	}
	sort.Strings(observedVersions)

	result := validationResult{
		ExecutorID:             executorID,
		ExecutorVersion:        executorVersion,
		TargetURL:              *targetURL,
		RequestedProtocol:      requestedFamily,
		RequestedVersion:       requestedVersion,
		ObservedVersions:       observedVersions,
		FallbackDetected:       fallbackDetected,
		StartedAtUTC:           started.Format(time.RFC3339Nano),
		DurationMS:             time.Since(started).Milliseconds(),
		Passed:                 passed,
		UnexpectedFailureCount: unexpectedFailures,
		TimeoutCount:           timeouts,
		Checks:                 checks,
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fatal(1, err.Error())
	}
	for _, artifact := range []struct {
		name  string
		value any
	}{
		{"validation.json", result},
		{"result.json", result},
		{"protocol-proof.json", protocolProof{
			ExecutorID: executorID, ExecutorVersion: executorVersion,
			RequestedProtocol: requestedFamily, RequestedVersion: requestedVersion,
			ObservedVersions: observedVersions, FallbackDetected: fallbackDetected,
			Passed: passed, Checks: checks,
		}},
		{"executor-identity.json", executorIdentity{
			ExecutorID: executorID, ExecutorVersion: executorVersion, Role: "client-test-executor",
			SupportedProtocols:      []string{requestedFamily},
			SupportedScenarios:      []string{"http1.core.plaintext", "http1.core.json"},
			LoadGenerationSupported: true,
			StdoutStderrOwner:       "invoking-runner-or-package-host",
		}},
	} {
		if err := writeJSON(filepath.Join(*outputDir, artifact.name), artifact.value); err != nil {
			fatal(1, err.Error())
		}
	}

	if passed {
		if *validationOnly {
			fmt.Fprintln(os.Stderr, "go-http1-executor validation passed with exact HTTP/1.1 protocol proof")
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
		loadResult, err := runOhaLoad(*targetURL, *outputDir, expectation, config)
		if err != nil {
			fatal(1, err.Error())
		}
		if err := writeJSON(filepath.Join(*outputDir, "load-generator-identity.json"), loadResult.LoadGenerator); err != nil {
			fatal(1, err.Error())
		}
		if err := writeExecutorResult(*outputDir, loadResult); err != nil {
			fatal(1, err.Error())
		}
		return
	}

	fmt.Fprintln(os.Stderr, "go-http1-executor validation failed")
	os.Exit(1)
}

func defaultExpectations() []scenarioExpectation {
	return []scenarioExpectation{
		{ScenarioID: "http1.core.plaintext", Path: "/plaintext", ExpectedStatus: http.StatusOK, ExpectedContentType: "text/plain", ExpectedBody: []byte("Hello, World!")},
		{ScenarioID: "http1.core.json", Path: "/json", ExpectedStatus: http.StatusOK, ExpectedContentType: "application/json", ExpectedBody: []byte(`{"message":"Hello, World!"}`)},
	}
}

func newHTTP1Client(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:              http.ProxyFromEnvironment,
		ForceAttemptHTTP2:  false,
		DisableCompression: true,
		DialContext:        (&net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}).DialContext,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("redirects are not permitted by the executor validity gate")
		},
	}
}

func runCheck(ctx context.Context, client *http.Client, baseURL string, expectation scenarioExpectation) checkResult {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Proto = requestedVersion
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Header.Set("User-Agent", executorID+"/"+executorVersion)
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := client.Do(req)
	if err != nil {
		result.TimedOut = isTimeout(err)
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	result.ObservedStatus = resp.StatusCode
	result.ObservedContentType = resp.Header.Get("Content-Type")
	result.ObservedVersion = resp.Proto
	result.FallbackDetected = !isExactHTTP11(resp.ProtoMajor, resp.ProtoMinor)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.ObservedPayloadLength = len(body)
	result.ObservedPayloadSHA256 = sha256Hex(body)

	problems := make([]string, 0, 5)
	if result.FallbackDetected {
		problems = append(problems, fmt.Sprintf("expected exact HTTP/1.1, observed %s", resp.Proto))
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

func mediaTypeMatches(observed, expected string) bool {
	observed = strings.TrimSpace(strings.SplitN(observed, ";", 2)[0])
	return strings.EqualFold(observed, expected)
}

func isExactHTTP11(major, minor int) bool {
	return major == 1 && minor == 1
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func sha256Hex(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
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
		return "", errors.New("go-http1-executor 0.3.0 supports cleartext HTTP/1.1 targets only")
	}
	parsed.Path = requestPath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
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
