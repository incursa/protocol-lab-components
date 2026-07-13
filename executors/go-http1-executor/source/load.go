package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ohaVersion               = "1.15.0"
	ohaWindowsAMD64PGOSHA256 = "b1dac6c1272abbb4b2c52723d869f0bfdd7807e742955e46e5cb515a15114f6f"
	ohaLinuxAMD64PGOSHA256   = "54008b400a990998824c4f91952c18565433452b3b71c2f4b47d9aebfaa34d9c"
)

type loadConfig struct {
	ScenarioID           string
	Connections          int
	StreamsPerConnection int
	Duration             time.Duration
	Warmup               time.Duration
	Repetition           int
	RequestTimeout       time.Duration
	ExecutionTimeout     time.Duration
	OhaPath              string
}

type executorResult struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Executor      componentIdentity       `json:"executor"`
	LoadGenerator loadGeneratorIdentity   `json:"loadGenerator"`
	Validation    validitySummary         `json:"validation"`
	ProtocolProof normalizedProtocolProof `json:"protocolProof"`
	RequestedLoad normalizedLoadShape     `json:"requestedLoad"`
	EffectiveLoad normalizedLoadShape     `json:"effectiveLoad"`
	Metrics       normalizedHTTPMetrics   `json:"metrics"`
	Warnings      []string                `json:"warnings"`
}

type componentIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type loadGeneratorIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Command string `json:"command"`
}

type validitySummary struct {
	Status string `json:"status"`
}

type normalizedProtocolProof struct {
	RequestedProtocol    string `json:"requestedProtocol"`
	ObservedProtocol     string `json:"observedProtocol"`
	ExactProtocolMatched bool   `json:"exactProtocolMatched"`
	FallbackDetected     bool   `json:"fallbackDetected"`
}

type normalizedLoadShape struct {
	Connections          int     `json:"connections"`
	Concurrency          int     `json:"concurrency"`
	StreamsPerConnection int     `json:"streamsPerConnection"`
	DurationSeconds      float64 `json:"durationSeconds"`
	WarmupSeconds        float64 `json:"warmupSeconds"`
	Repetition           int     `json:"repetition"`
}

type normalizedHTTPMetrics struct {
	TotalRequests            int64            `json:"totalRequests"`
	SuccessfulRequests       int64            `json:"successfulRequests"`
	FailedRequests           int64            `json:"failedRequests"`
	TimeoutRequests          int64            `json:"timeoutRequests"`
	RequestsPerSecond        float64          `json:"requestsPerSecond"`
	BytesSent                int64            `json:"bytesSent"`
	BytesReceived            int64            `json:"bytesReceived"`
	ThroughputBytesPerSecond float64          `json:"throughputBytesPerSecond"`
	LatencyMeanMS            float64          `json:"latencyMeanMs"`
	LatencyP50MS             float64          `json:"latencyP50Ms"`
	LatencyP75MS             float64          `json:"latencyP75Ms"`
	LatencyP90MS             float64          `json:"latencyP90Ms"`
	LatencyP95MS             float64          `json:"latencyP95Ms"`
	LatencyP99MS             float64          `json:"latencyP99Ms"`
	TimeToFirstByteMeanMS    float64          `json:"timeToFirstByteMeanMs,omitempty"`
	StatusCodeCounts         map[string]int64 `json:"statusCodeCounts,omitempty"`
}

type ohaJSON struct {
	Summary struct {
		Total          float64 `json:"total"`
		Average        float64 `json:"average"`
		RequestsPerSec float64 `json:"requestsPerSec"`
		TotalData      int64   `json:"totalData"`
		SizePerSec     float64 `json:"sizePerSec"`
	} `json:"summary"`
	LatencyPercentiles map[string]float64 `json:"latencyPercentiles"`
	Details            struct {
		FirstByte struct {
			Average float64 `json:"average"`
		} `json:"firstByte"`
	} `json:"details"`
	StatusCodeDistribution map[string]int64 `json:"statusCodeDistribution"`
	ErrorDistribution      map[string]int64 `json:"errorDistribution"`
}

func loadConfigFromEnvironment() (loadConfig, error) {
	config := loadConfig{
		ScenarioID:           strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")),
		Connections:          envInt("PLAB_CONNECTIONS", 0),
		StreamsPerConnection: envInt("PLAB_STREAMS_PER_CONNECTION", 1),
		Duration:             time.Duration(envInt("PLAB_DURATION_SECONDS", 0)) * time.Second,
		Warmup:               time.Duration(envInt("PLAB_WARMUP_SECONDS", 0)) * time.Second,
		Repetition:           envInt("PLAB_REPETITION", 1),
		RequestTimeout:       10 * time.Second,
		ExecutionTimeout:     time.Duration(envInt("PLAB_TIMEOUT_SECONDS", 0)) * time.Second,
		OhaPath:              strings.TrimSpace(os.Getenv("PLAB_OHA_PATH")),
	}
	if config.ScenarioID == "" {
		return config, errors.New("PLAB_SCENARIO_ID is required for performance execution")
	}
	if config.Connections <= 0 {
		return config, errors.New("PLAB_CONNECTIONS must be greater than zero")
	}
	if config.StreamsPerConnection != 1 {
		return config, fmt.Errorf("HTTP/1.1 requires PLAB_STREAMS_PER_CONNECTION=1, observed %d", config.StreamsPerConnection)
	}
	if config.Duration <= 0 {
		return config, errors.New("PLAB_DURATION_SECONDS must be greater than zero")
	}
	if config.Warmup < 0 || config.Repetition <= 0 {
		return config, errors.New("warmup must be non-negative and repetition must be greater than zero")
	}
	if config.ExecutionTimeout <= 0 {
		config.ExecutionTimeout = config.Duration + config.Warmup + 30*time.Second
	}
	if config.OhaPath == "" {
		var err error
		config.OhaPath, err = resolvePackagedOhaPath()
		if err != nil {
			return config, err
		}
	}
	return config, nil
}

func runOhaLoad(targetBaseURL, outputDir string, expectation scenarioExpectation, config loadConfig) (executorResult, error) {
	if err := verifyOha(config.OhaPath); err != nil {
		return executorResult{}, err
	}
	versionOutput, err := runCommand(context.Background(), config.OhaPath, "--version")
	if err != nil {
		return executorResult{}, fmt.Errorf("capture oha version: %w", err)
	}
	observedVersion := strings.TrimSpace(string(versionOutput.Stdout))
	if observedVersion != "oha "+ohaVersion {
		return executorResult{}, fmt.Errorf("oha version mismatch: expected %q, observed %q", "oha "+ohaVersion, observedVersion)
	}

	targetURL, err := joinURL(targetBaseURL, expectation.Path)
	if err != nil {
		return executorResult{}, err
	}
	if config.Warmup > 0 {
		warmupArgs := buildOhaArguments(targetURL, config.Connections, config.Warmup, config.RequestTimeout)
		warmupContext, cancel := context.WithTimeout(context.Background(), config.Warmup+30*time.Second)
		warmup, warmupErr := runCommand(warmupContext, config.OhaPath, warmupArgs...)
		cancel()
		_ = os.WriteFile(filepath.Join(outputDir, "oha-warmup.stdout.json"), warmup.Stdout, 0o644)
		_ = os.WriteFile(filepath.Join(outputDir, "oha-warmup.stderr.txt"), warmup.Stderr, 0o644)
		if warmupErr != nil {
			return executorResult{}, fmt.Errorf("oha warmup failed: %w", warmupErr)
		}
	}

	args := buildOhaArguments(targetURL, config.Connections, config.Duration, config.RequestTimeout)
	commandText := formatCommand(config.OhaPath, args)
	measuredContext, cancel := context.WithTimeout(context.Background(), config.ExecutionTimeout)
	measured, measuredErr := runCommand(measuredContext, config.OhaPath, args...)
	cancel()
	if err := os.WriteFile(filepath.Join(outputDir, "oha.stdout.json"), measured.Stdout, 0o644); err != nil {
		return executorResult{}, err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "oha.stderr.txt"), measured.Stderr, 0o644); err != nil {
		return executorResult{}, err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "oha.command.txt"), []byte(commandText+"\n"), 0o644); err != nil {
		return executorResult{}, err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "oha.version.txt"), []byte(observedVersion+"\n"), 0o644); err != nil {
		return executorResult{}, err
	}
	if measuredErr != nil {
		return executorResult{}, fmt.Errorf("oha measured run failed: %w", measuredErr)
	}

	parsed, err := parseOhaJSON(measured.Stdout, expectation.ExpectedStatus)
	if err != nil {
		return executorResult{}, err
	}
	hash, err := fileSHA256(config.OhaPath)
	if err != nil {
		return executorResult{}, err
	}
	result := executorResult{
		SchemaVersion: "protocol-lab.http-executor-result.v1",
		Executor:      componentIdentity{ID: executorID, Version: executorVersion},
		LoadGenerator: loadGeneratorIdentity{ID: "oha", Version: ohaVersion, SHA256: hash, Command: commandText},
		Validation:    validitySummary{Status: "passed"},
		ProtocolProof: normalizedProtocolProof{
			RequestedProtocol: requestedVersion, ObservedProtocol: requestedVersion,
			ExactProtocolMatched: true, FallbackDetected: false,
		},
		RequestedLoad: normalizedLoadShape{
			Connections: config.Connections, Concurrency: config.Connections,
			StreamsPerConnection: 1, DurationSeconds: config.Duration.Seconds(),
			WarmupSeconds: config.Warmup.Seconds(), Repetition: config.Repetition,
		},
		EffectiveLoad: normalizedLoadShape{
			Connections: config.Connections, Concurrency: config.Connections,
			StreamsPerConnection: 1, DurationSeconds: parsed.Summary.Total,
			WarmupSeconds: config.Warmup.Seconds(), Repetition: config.Repetition,
		},
		Metrics:  normalizeOhaMetrics(parsed, expectation.ExpectedStatus),
		Warnings: []string{},
	}
	if result.Metrics.FailedRequests != 0 || result.Metrics.TimeoutRequests != 0 {
		return executorResult{}, fmt.Errorf(
			"oha recorded failed or timed-out requests (failed=%d timeout=%d)",
			result.Metrics.FailedRequests,
			result.Metrics.TimeoutRequests)
	}
	return result, nil
}

type commandOutput struct {
	Stdout []byte
	Stderr []byte
}

func runCommand(ctx context.Context, executable string, arguments ...string) (commandOutput, error) {
	command := exec.CommandContext(ctx, executable, arguments...)
	var stdout strings.Builder
	var stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return commandOutput{Stdout: []byte(stdout.String()), Stderr: []byte(stderr.String())}, err
}

func buildOhaArguments(targetURL string, connections int, duration, requestTimeout time.Duration) []string {
	return []string{
		"--no-tui", "--no-color", "--output-format", "json",
		"--http-version", "1.1", "--redirect", "0", "--disable-compression",
		"--wait-ongoing-requests-after-deadline",
		"-z", formatDuration(duration), "-c", strconv.Itoa(connections),
		"-t", formatDuration(requestTimeout), "-H", "Accept-Encoding: identity",
		targetURL,
	}
}

func parseOhaJSON(data []byte, expectedStatus int) (ohaJSON, error) {
	var parsed ohaJSON
	if err := json.Unmarshal(data, &parsed); err != nil {
		return parsed, fmt.Errorf("parse oha JSON: %w", err)
	}
	for _, percentile := range []string{"p50", "p75", "p90", "p95", "p99"} {
		if _, exists := parsed.LatencyPercentiles[percentile]; !exists {
			return parsed, fmt.Errorf("oha JSON did not contain latency percentile %s", percentile)
		}
	}
	if parsed.Summary.RequestsPerSec < 0 || parsed.Summary.TotalData < 0 || parsed.Summary.SizePerSec < 0 {
		return parsed, errors.New("oha JSON contained negative metrics")
	}
	if _, exists := parsed.StatusCodeDistribution[strconv.Itoa(expectedStatus)]; !exists {
		return parsed, fmt.Errorf("oha JSON did not contain expected status %d", expectedStatus)
	}
	return parsed, nil
}

func normalizeOhaMetrics(parsed ohaJSON, expectedStatus int) normalizedHTTPMetrics {
	var total, successful, failed, timedOut int64
	expected := strconv.Itoa(expectedStatus)
	for status, count := range parsed.StatusCodeDistribution {
		total += count
		if status == expected {
			successful += count
		} else {
			failed += count
		}
	}
	for name, count := range parsed.ErrorDistribution {
		total += count
		if strings.Contains(strings.ToLower(name), "timeout") || strings.Contains(strings.ToLower(name), "deadline") {
			timedOut += count
		} else {
			failed += count
		}
	}
	return normalizedHTTPMetrics{
		TotalRequests: total, SuccessfulRequests: successful, FailedRequests: failed,
		TimeoutRequests: timedOut, RequestsPerSecond: parsed.Summary.RequestsPerSec,
		BytesSent: 0, BytesReceived: parsed.Summary.TotalData,
		ThroughputBytesPerSecond: parsed.Summary.SizePerSec,
		LatencyMeanMS:            parsed.Summary.Average * 1000,
		LatencyP50MS:             parsed.LatencyPercentiles["p50"] * 1000,
		LatencyP75MS:             parsed.LatencyPercentiles["p75"] * 1000,
		LatencyP90MS:             parsed.LatencyPercentiles["p90"] * 1000,
		LatencyP95MS:             parsed.LatencyPercentiles["p95"] * 1000,
		LatencyP99MS:             parsed.LatencyPercentiles["p99"] * 1000,
		TimeToFirstByteMeanMS:    parsed.Details.FirstByte.Average * 1000,
		StatusCodeCounts:         parsed.StatusCodeDistribution,
	}
}

func findExpectation(scenarioID string) (scenarioExpectation, error) {
	for _, expectation := range defaultExpectations() {
		if expectation.ScenarioID == scenarioID {
			return expectation, nil
		}
	}
	return scenarioExpectation{}, fmt.Errorf("scenario %q is unsupported by %s", scenarioID, executorID)
}

func verifyOha(path string) error {
	expected, err := expectedOhaSHA256()
	if err != nil {
		return err
	}
	actual, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("oha SHA-256 mismatch: expected %s, observed %s", expected, actual)
	}
	return nil
}

func expectedOhaSHA256() (string, error) {
	if runtime.GOARCH != "amd64" {
		return "", fmt.Errorf("oha %s is not packaged for %s/%s", ohaVersion, runtime.GOOS, runtime.GOARCH)
	}
	switch runtime.GOOS {
	case "windows":
		return ohaWindowsAMD64PGOSHA256, nil
	case "linux":
		return ohaLinuxAMD64PGOSHA256, nil
	default:
		return "", fmt.Errorf("oha %s is not packaged for %s/%s", ohaVersion, runtime.GOOS, runtime.GOARCH)
	}
}

func resolvePackagedOhaPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	rid := "linux-x64"
	name := "oha"
	if runtime.GOOS == "windows" {
		rid = "win-x64"
		name = "oha.exe"
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(executable), "..", "..", "tools", rid, name))
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("packaged oha was unavailable at %s: %w", path, err)
	}
	return path, nil
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func formatDuration(value time.Duration) string {
	if value%time.Second == 0 {
		return strconv.FormatInt(int64(value/time.Second), 10) + "s"
	}
	return value.String()
}

func formatCommand(executable string, arguments []string) string {
	parts := []string{executable}
	for _, argument := range arguments {
		if strings.ContainsAny(argument, " \t\"") {
			parts = append(parts, strconv.Quote(argument))
		} else {
			parts = append(parts, argument)
		}
	}
	return strings.Join(parts, " ")
}

func writeExecutorResult(outputDir string, result executorResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(outputDir, "http-executor-result.json"), data, 0o644); err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

func sortedKeys(values map[string]int64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
