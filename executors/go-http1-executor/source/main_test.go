package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestRunCheckProvesExactHTTP11AndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Proto != requestedVersion {
			t.Fatalf("request protocol = %q, want %q", r.Proto, requestedVersion)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	result := runCheck(context.Background(), newHTTP1Client(time.Second), server.URL, scenarioExpectation{
		ScenarioID: "http1.core.plaintext", Path: "/plaintext", ExpectedStatus: 200,
		ExpectedContentType: "text/plain", ExpectedBody: []byte("Hello, World!"),
	})

	if !result.Passed {
		t.Fatalf("validation failed: %s", result.Error)
	}
	if result.ObservedVersion != requestedVersion || result.FallbackDetected {
		t.Fatalf("observed protocol = %q, fallback = %t", result.ObservedVersion, result.FallbackDetected)
	}
	if result.ExpectedPayloadSHA256 != result.ObservedPayloadSHA256 {
		t.Fatalf("payload hashes differ: expected %s, observed %s", result.ExpectedPayloadSHA256, result.ObservedPayloadSHA256)
	}
}

func TestRunCheckRejectsWrongStatusTypeAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("wrong"))
	}))
	defer server.Close()

	result := runCheck(context.Background(), newHTTP1Client(time.Second), server.URL, scenarioExpectation{
		ScenarioID: "http1.core.plaintext", Path: "/plaintext", ExpectedStatus: 200,
		ExpectedContentType: "text/plain", ExpectedBody: []byte("Hello, World!"),
	})

	if result.Passed {
		t.Fatal("validation unexpectedly passed")
	}
	for _, expected := range []string{"expected status", "expected content type", "expected payload length", "expected payload SHA-256"} {
		if !strings.Contains(result.Error, expected) {
			t.Errorf("error %q does not contain %q", result.Error, expected)
		}
	}
}

func TestRunCheckRejectsHTTP2ResponseAsFallback(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Proto:      "HTTP/2.0",
			ProtoMajor: 2,
			ProtoMinor: 0,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       io.NopCloser(strings.NewReader("Hello, World!")),
			Request:    request,
		}, nil
	})}

	result := runCheck(context.Background(), client, "http://example.test", scenarioExpectation{
		ScenarioID: "http1.core.plaintext", Path: "/plaintext", ExpectedStatus: 200,
		ExpectedContentType: "text/plain", ExpectedBody: []byte("Hello, World!"),
	})

	if result.Passed || !result.FallbackDetected || result.ObservedVersion != "HTTP/2.0" {
		t.Fatalf("expected HTTP/2 fallback rejection, got passed=%t fallback=%t observed=%q", result.Passed, result.FallbackDetected, result.ObservedVersion)
	}
	if !strings.Contains(result.Error, "expected exact HTTP/1.1") {
		t.Fatalf("unexpected error: %s", result.Error)
	}
}

func TestRunCheckClassifiesTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	result := runCheck(context.Background(), newHTTP1Client(50*time.Millisecond), server.URL, scenarioExpectation{
		ScenarioID: "http1.core.plaintext", Path: "/plaintext", ExpectedStatus: 200,
		ExpectedContentType: "text/plain", ExpectedBody: []byte("Hello, World!"),
	})

	if result.Passed || !result.TimedOut {
		t.Fatalf("expected timeout failure, got passed=%t timedOut=%t error=%q", result.Passed, result.TimedOut, result.Error)
	}
}

func TestRunCheckRejectsUnreachableTarget(t *testing.T) {
	result := runCheck(context.Background(), newHTTP1Client(250*time.Millisecond), "http://127.0.0.1:1", scenarioExpectation{
		ScenarioID: "http1.core.plaintext", Path: "/plaintext", ExpectedStatus: 200,
		ExpectedContentType: "text/plain", ExpectedBody: []byte("Hello, World!"),
	})

	if result.Passed || result.Error == "" || result.ObservedVersion != "" {
		t.Fatalf("expected unreachable-target failure, got passed=%t observed=%q error=%q", result.Passed, result.ObservedVersion, result.Error)
	}
}

func TestJoinURLRejectsTLSUntilSeparateExecutionProfileExists(t *testing.T) {
	_, err := joinURL("https://example.test", "/plaintext")
	if err == nil || !strings.Contains(err.Error(), "cleartext HTTP/1.1") {
		t.Fatalf("expected cleartext-only error, got %v", err)
	}
}

func TestJoinURLReplacesScenarioPathInsteadOfAppendingIt(t *testing.T) {
	actual, err := joinURL("http://example.test/json", "/plaintext")
	if err != nil {
		t.Fatal(err)
	}
	if actual != "http://example.test/plaintext" {
		t.Fatalf("joined URL = %q", actual)
	}
}

func TestBuildOhaArgumentsForcesExactHTTP11AndWaitsForOngoingRequests(t *testing.T) {
	arguments := buildOhaArguments("http://example.test/plaintext", 16, 30*time.Second, 10*time.Second)
	joined := strings.Join(arguments, " ")
	for _, required := range []string{
		"--http-version 1.1",
		"--redirect 0",
		"--wait-ongoing-requests-after-deadline",
		"-z 30s",
		"-c 16",
		"-t 10s",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("arguments %q do not contain %q", joined, required)
		}
	}
}

func TestParseAndNormalizeOhaJSON(t *testing.T) {
	parsed, err := parseOhaJSON([]byte(validOhaJSON), http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
	metrics := normalizeOhaMetrics(parsed, http.StatusOK)
	if metrics.TotalRequests != 102 || metrics.SuccessfulRequests != 100 || metrics.FailedRequests != 1 || metrics.TimeoutRequests != 1 {
		t.Fatalf("unexpected request outcomes: %+v", metrics)
	}
	if metrics.RequestsPerSecond != 50.5 || metrics.LatencyP75MS != 2.5 || metrics.TimeToFirstByteMeanMS != 1.25 {
		t.Fatalf("unexpected normalized metrics: %+v", metrics)
	}
}

func TestParseOhaJSONRequiresRankingPercentiles(t *testing.T) {
	_, err := parseOhaJSON([]byte(strings.Replace(validOhaJSON, `"p75": 0.0025,`, "", 1)), http.StatusOK)
	if err == nil || !strings.Contains(err.Error(), "p75") {
		t.Fatalf("expected missing-p75 error, got %v", err)
	}
}

func TestLoadConfigRejectsHTTP1StreamsGreaterThanOne(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", "http1.core.plaintext")
	t.Setenv("PLAB_CONNECTIONS", "4")
	t.Setenv("PLAB_STREAMS_PER_CONNECTION", "2")
	t.Setenv("PLAB_DURATION_SECONDS", "10")
	_, err := loadConfigFromEnvironment()
	if err == nil || !strings.Contains(err.Error(), "STREAMS_PER_CONNECTION=1") {
		t.Fatalf("expected streams rejection, got %v", err)
	}
}

func TestMediaTypeMatchingIgnoresParametersButNotType(t *testing.T) {
	if !mediaTypeMatches("application/json; charset=utf-8", "application/json") {
		t.Fatal("expected media type with charset to match")
	}
	if mediaTypeMatches("text/plain", "application/json") {
		t.Fatal("different media types unexpectedly matched")
	}
}

func TestExactHTTP11VersionCheck(t *testing.T) {
	if !isExactHTTP11(1, 1) {
		t.Fatal("HTTP/1.1 should be accepted")
	}
	for _, version := range [][2]int{{1, 0}, {2, 0}, {3, 0}} {
		if isExactHTTP11(version[0], version[1]) {
			t.Fatalf("version %d.%d unexpectedly accepted", version[0], version[1])
		}
	}
}

const validOhaJSON = `{
  "summary": {
    "total": 2.0,
    "average": 0.002,
    "requestsPerSec": 50.5,
    "totalData": 1300,
    "sizePerSec": 650.0
  },
  "latencyPercentiles": {
    "p50": 0.001,
    "p75": 0.0025,
    "p90": 0.003,
    "p95": 0.004,
    "p99": 0.005
  },
  "details": { "firstByte": { "average": 0.00125 } },
  "statusCodeDistribution": { "200": 100, "500": 1 },
  "errorDistribution": { "request timeout": 1 }
}`
