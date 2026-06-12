package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const executorID = "go-http1-executor"

type validationResult struct {
	ExecutorID   string        `json:"executorId"`
	TargetURL    string        `json:"targetUrl"`
	Protocol     string        `json:"protocol"`
	StartedAtUTC string        `json:"startedAtUtc"`
	DurationMS   int64         `json:"durationMs"`
	Passed       bool          `json:"passed"`
	Checks       []checkResult `json:"checks"`
}

type checkResult struct {
	TestCaseID string `json:"testCaseId"`
	Path       string `json:"path"`
	Passed     bool   `json:"passed"`
	StatusCode int    `json:"statusCode"`
	Error      string `json:"error,omitempty"`
}

func main() {
	targetURL := flag.String("target-url", os.Getenv("PLAB_TARGET_BASE_URL"), "Target base URL.")
	outputDir := flag.String("output-dir", os.Getenv("PLAB_ARTIFACT_DIR"), "Artifact output directory.")
	flag.Parse()

	if strings.TrimSpace(*targetURL) == "" {
		fatal(2, "target-url or PLAB_TARGET_BASE_URL is required")
	}

	if strings.TrimSpace(*outputDir) == "" {
		*outputDir = "artifacts"
	}

	started := time.Now().UTC()
	client := &http.Client{Timeout: 10 * time.Second}
	ctx := context.Background()

	checks := []checkResult{
		runCheck(ctx, client, *targetURL, "http.core.plaintext", "/plaintext", "Hello, World!", "text/plain"),
		runCheck(ctx, client, *targetURL, "http.core.json", "/json", `"message":"Hello, World!"`, "application/json"),
	}

	passed := true
	for _, check := range checks {
		if !check.Passed {
			passed = false
			break
		}
	}

	result := validationResult{
		ExecutorID:   executorID,
		TargetURL:    *targetURL,
		Protocol:     "h1",
		StartedAtUTC: started.Format(time.RFC3339Nano),
		DurationMS:   time.Since(started).Milliseconds(),
		Passed:       passed,
		Checks:       checks,
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fatal(1, err.Error())
	}

	if err := writeJSON(filepath.Join(*outputDir, "validation.json"), result); err != nil {
		fatal(1, err.Error())
	}
	if err := writeJSON(filepath.Join(*outputDir, "result.json"), result); err != nil {
		fatal(1, err.Error())
	}
	if err := writeJSON(filepath.Join(*outputDir, "load-tool-execution.json"), map[string]any{
		"executorId":   executorID,
		"targetUrl":    *targetURL,
		"protocol":     "h1",
		"completedUtc": time.Now().UTC().Format(time.RFC3339Nano),
		"passed":       passed,
	}); err != nil {
		fatal(1, err.Error())
	}

	if passed {
		fmt.Println("go-http1-executor validation passed")
		return
	}

	fmt.Fprintln(os.Stderr, "go-http1-executor validation failed")
	os.Exit(1)
}

func runCheck(ctx context.Context, client *http.Client, baseURL, testCaseID, requestPath, expectedBodyFragment, expectedContentType string) checkResult {
	requestURL, err := joinURL(baseURL, requestPath)
	if err != nil {
		return checkResult{TestCaseID: testCaseID, Path: requestPath, Error: err.Error()}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return checkResult{TestCaseID: testCaseID, Path: requestPath, Error: err.Error()}
	}
	req.Header.Set("User-Agent", executorID+"/0.1.0")

	resp, err := client.Do(req)
	if err != nil {
		return checkResult{TestCaseID: testCaseID, Path: requestPath, Error: err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return checkResult{TestCaseID: testCaseID, Path: requestPath, StatusCode: resp.StatusCode, Error: err.Error()}
	}

	contentType := resp.Header.Get("Content-Type")
	passed := resp.StatusCode == http.StatusOK &&
		strings.Contains(string(body), expectedBodyFragment) &&
		strings.Contains(contentType, expectedContentType)

	result := checkResult{
		TestCaseID: testCaseID,
		Path:       requestPath,
		Passed:     passed,
		StatusCode: resp.StatusCode,
	}
	if !passed {
		result.Error = fmt.Sprintf("expected status 200, body containing %q, and content type containing %q", expectedBodyFragment, expectedContentType)
	}
	return result
}

func joinURL(baseURL, requestPath string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("target-url must include scheme and host")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + requestPath
	return parsed.String(), nil
}

func writeJSON(filePath string, value any) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, append(bytes, '\n'), 0o644)
}

func fatal(code int, message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(code)
}
