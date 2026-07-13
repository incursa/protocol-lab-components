package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestExactH2CPriorKnowledgeValidation(t *testing.T) {
	serverURL := startH2CServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.ProtoMajor != 2 {
			t.Errorf("request protocol = %s, want HTTP/2", request.Proto)
		}
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = writer.Write([]byte("Hello, World!"))
	}))
	connection, err := dialH2C(context.Background(), serverURL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	result := runCheck(context.Background(), connection.client, serverURL, defaultExpectations()[0], time.Second)
	if !result.Passed || result.FallbackDetected || result.ObservedVersion != requestedVersion {
		t.Fatalf("unexpected validation result: %+v", result)
	}
}

func TestRunCheckRejectsHTTP1Fallback(t *testing.T) {
	client := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1,
			Header: http.Header{"Content-Type": []string{"text/plain"}},
			Body:   io.NopCloser(strings.NewReader("Hello, World!")), Request: request,
		}, nil
	})
	result := runCheck(context.Background(), client, "http://example.test", defaultExpectations()[0], time.Second)
	if result.Passed || !result.FallbackDetected || !strings.Contains(result.Error, "expected exact HTTP/2") {
		t.Fatalf("expected fail-closed HTTP/1 rejection, got %+v", result)
	}
}

func TestRunCheckRejectsRedirectAndWrongPayload(t *testing.T) {
	client := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusFound, Proto: requestedVersion, ProtoMajor: 2,
			Header: http.Header{"Content-Type": []string{"application/octet-stream"}},
			Body:   io.NopCloser(strings.NewReader("wrong")), Request: request,
		}, nil
	})
	result := runCheck(context.Background(), client, "http://example.test", defaultExpectations()[0], time.Second)
	if result.Passed || result.FallbackDetected {
		t.Fatalf("expected semantic rejection without protocol fallback, got %+v", result)
	}
	for _, required := range []string{"expected status", "expected content type", "expected payload length", "expected payload SHA-256"} {
		if !strings.Contains(result.Error, required) {
			t.Errorf("error %q does not contain %q", result.Error, required)
		}
	}
}

func TestSmokeLoadProvesOneDialAndOneActiveOperation(t *testing.T) {
	serverURL := startH2CServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write([]byte("Hello, World!"))
	}))
	config := loadConfig{
		ScenarioID: "http2.core.plaintext", LoadProfileID: "http2-smoke",
		Connections: 1, Concurrency: 1, StreamsPerConnection: 1,
		Duration: 100 * time.Millisecond, Repetition: 1,
		RequestTimeout: time.Second, ExecutionTimeout: 5 * time.Second,
		OperationDistribution: "balanced-round-robin",
	}
	result, err := runH2CLoad(serverURL, t.TempDir(), defaultExpectations()[0], config)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProtocolProof.ObservedDials != 1 || result.ProtocolProof.MaximumActiveOperations != 1 {
		t.Fatalf("unexpected topology proof: %+v", result.ProtocolProof)
	}
	if result.EffectiveLoad.Connections != 1 || result.EffectiveLoad.Concurrency != 1 || result.EffectiveLoad.StreamsPerConnection != 1 {
		t.Fatalf("unexpected effective load: %+v", result.EffectiveLoad)
	}
	if result.Metrics.SuccessfulRequests == 0 || result.Metrics.FailedRequests != 0 || result.Metrics.TimeoutRequests != 0 {
		t.Fatalf("unexpected metrics: %+v", result.Metrics)
	}
	if result.LoadGenerator.ID == result.Executor.ID || result.LoadGenerator.ID != loadGeneratorID {
		t.Fatalf("executor/generator identity was not distinct: executor=%q generator=%q", result.Executor.ID, result.LoadGenerator.ID)
	}
}

func TestDiagnosticLoadProvesCommittedOneByEightByEightShape(t *testing.T) {
	serverURL := startH2CServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write([]byte("Hello, World!"))
	}))
	config := loadConfig{
		ScenarioID: "http2.core.plaintext", LoadProfileID: "http2-diagnostic",
		Connections: 1, Concurrency: 8, StreamsPerConnection: 8,
		Duration: 100 * time.Millisecond, Repetition: 1,
		RequestTimeout: time.Second, ExecutionTimeout: 5 * time.Second,
		OperationDistribution: "balanced-round-robin",
	}
	result, err := runH2CLoad(serverURL, t.TempDir(), defaultExpectations()[0], config)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProtocolProof.ObservedDials != 1 || result.ProtocolProof.MaximumActiveOperations != 8 {
		t.Fatalf("unexpected diagnostic topology proof: %+v", result.ProtocolProof)
	}
	if result.ProtocolProof.MinimumPeerAdvertisedMaxConcurrentStreams < 8 {
		t.Fatalf("peer HTTP/2 stream limit was not proven: %+v", result.ProtocolProof)
	}
	if result.EffectiveLoad.Connections != 1 || result.EffectiveLoad.Concurrency != 8 || result.EffectiveLoad.StreamsPerConnection != 8 {
		t.Fatalf("unexpected effective diagnostic load: %+v", result.EffectiveLoad)
	}
	if result.RequestedLoad.RequestTimeoutSeconds != 1 || result.EffectiveLoad.RequestTimeoutSeconds != 1 {
		t.Fatalf("request timeout was not preserved in load evidence: requested=%+v effective=%+v", result.RequestedLoad, result.EffectiveLoad)
	}
	if result.Metrics.SuccessfulRequests == 0 || result.Metrics.FailedRequests != 0 || result.Metrics.TimeoutRequests != 0 {
		t.Fatalf("unexpected diagnostic metrics: %+v", result.Metrics)
	}
}

func TestComparisonLoadProvesSixteenByOneHundredTwentyEightByEightShape(t *testing.T) {
	serverURL := startH2CServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write([]byte("Hello, World!"))
	}))
	config := loadConfig{
		ScenarioID: "http2.core.plaintext", LoadProfileID: "http2-comparison",
		Connections: 16, Concurrency: 128, StreamsPerConnection: 8,
		OperationDistribution: "balanced-round-robin",
		Duration:              100 * time.Millisecond, Repetition: 1,
		RequestTimeout: time.Second, ExecutionTimeout: 5 * time.Second,
	}
	outputDir := t.TempDir()
	result, err := runH2CLoad(serverURL, outputDir, defaultExpectations()[0], config)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProtocolProof.ObservedDials != 16 || result.ProtocolProof.MaximumActiveOperations != 128 {
		t.Fatalf("unexpected comparison topology proof: %+v", result.ProtocolProof)
	}
	if result.EffectiveLoad.OperationDistribution != "balanced-round-robin" ||
		len(result.EffectiveLoad.MaximumActiveStreamsByConnection) != 16 {
		t.Fatalf("comparison distribution evidence was incomplete: %+v", result.EffectiveLoad)
	}
	for index, active := range result.EffectiveLoad.MaximumActiveStreamsByConnection {
		if active != 8 {
			t.Fatalf("connection %d maximum active operations = %d, want 8", index, active)
		}
	}
	if _, err := os.Stat(filepath.Join(outputDir, "http2-topology.json")); err != nil {
		t.Fatalf("http2-topology.json was not produced: %v", err)
	}
	if result.Metrics.SuccessfulRequests == 0 || result.Metrics.FailedRequests != 0 || result.Metrics.TimeoutRequests != 0 {
		t.Fatalf("unexpected comparison metrics: %+v", result.Metrics)
	}
}

func TestLoadConfigAcceptsOnlyCommittedSmokeDiagnosticAndComparisonShapes(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", "http2.core.plaintext")
	t.Setenv("PLAB_LOAD_PROFILE_ID", "http2-smoke")
	t.Setenv("PLAB_CONNECTIONS", "1")
	t.Setenv("PLAB_CONCURRENCY", "1")
	t.Setenv("PLAB_STREAMS_PER_CONNECTION", "1")
	t.Setenv("PLAB_DURATION_SECONDS", "5")
	t.Setenv("PLAB_WARMUP_SECONDS", "1")
	t.Setenv("PLAB_REQUEST_TIMEOUT_SECONDS", "5")
	t.Setenv("PLAB_OPERATION_DISTRIBUTION", "")
	config, err := loadConfigFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if config.Connections != 1 || config.Concurrency != 1 || config.StreamsPerConnection != 1 {
		t.Fatalf("unexpected config: %+v", config)
	}

	t.Setenv("PLAB_CONCURRENCY", "8")
	if _, err := loadConfigFromEnvironment(); err == nil || !strings.Contains(err.Error(), "requires connections=1, concurrency=1") {
		t.Fatalf("expected concurrency rejection, got %v", err)
	}
	t.Setenv("PLAB_STREAMS_PER_CONNECTION", "8")
	t.Setenv("PLAB_LOAD_PROFILE_ID", "http2-diagnostic")
	t.Setenv("PLAB_DURATION_SECONDS", "10")
	t.Setenv("PLAB_REQUEST_TIMEOUT_SECONDS", "10")
	config, err = loadConfigFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if config.Connections != 1 || config.Concurrency != 8 || config.StreamsPerConnection != 8 {
		t.Fatalf("unexpected diagnostic config: %+v", config)
	}
	if config.RequestTimeout != 10*time.Second {
		t.Fatalf("request timeout = %s, want 10s", config.RequestTimeout)
	}

	t.Setenv("PLAB_REQUEST_TIMEOUT_SECONDS", "9")
	if _, err := loadConfigFromEnvironment(); err == nil || !strings.Contains(err.Error(), "requestTimeout=10s") {
		t.Fatalf("expected request-timeout rejection, got %v", err)
	}
	t.Setenv("PLAB_REQUEST_TIMEOUT_SECONDS", "10")

	t.Setenv("PLAB_LOAD_PROFILE_ID", "http2-comparison")
	t.Setenv("PLAB_CONNECTIONS", "16")
	t.Setenv("PLAB_CONCURRENCY", "128")
	t.Setenv("PLAB_STREAMS_PER_CONNECTION", "8")
	t.Setenv("PLAB_DURATION_SECONDS", "30")
	t.Setenv("PLAB_WARMUP_SECONDS", "10")
	t.Setenv("PLAB_REQUEST_TIMEOUT_SECONDS", "10")
	if _, err := loadConfigFromEnvironment(); err == nil || !strings.Contains(err.Error(), "operationDistribution=balanced-round-robin") {
		t.Fatalf("expected missing distribution rejection, got %v", err)
	}
	t.Setenv("PLAB_OPERATION_DISTRIBUTION", "balanced-round-robin")
	config, err = loadConfigFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if config.Connections != 16 || config.Concurrency != 128 || config.StreamsPerConnection != 8 {
		t.Fatalf("unexpected comparison config: %+v", config)
	}
	t.Setenv("PLAB_REPETITION", "4")
	if _, err := loadConfigFromEnvironment(); err == nil || !strings.Contains(err.Error(), "repetition in [1,3]") {
		t.Fatalf("expected comparison repetition rejection, got %v", err)
	}
}

func TestJoinURLRejectsTLSAndReplacesPath(t *testing.T) {
	if _, err := joinURL("https://example.test", "/plaintext"); err == nil || !strings.Contains(err.Error(), "h2c") {
		t.Fatalf("expected TLS rejection, got %v", err)
	}
	actual, err := joinURL("http://example.test/json", "/plaintext")
	if err != nil || actual != "http://example.test/plaintext" {
		t.Fatalf("joined URL = %q, err = %v", actual, err)
	}
}

func TestPercentilesUseNearestRank(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	if percentile(values, 0.50) != 3 || percentile(values, 0.75) != 4 || percentile(values, 0.99) != 5 {
		t.Fatalf("unexpected percentiles: p50=%v p75=%v p99=%v", percentile(values, 0.50), percentile(values, 0.75), percentile(values, 0.99))
	}
}

func startH2CServer(t *testing.T, handler http.Handler) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http2.Server{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go server.ServeConn(connection, &http2.ServeConnOpts{Handler: handler})
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("h2c accept loop did not stop")
		}
	})
	return "http://" + listener.Addr().String()
}

func TestDirectH2CPrefaceDoesNotAcceptHTTP1Server(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write([]byte("Hello, World!"))
	}))
	defer server.Close()
	connection, err := dialH2C(context.Background(), server.URL, time.Second)
	if err == nil {
		defer connection.Close()
		result := runCheck(context.Background(), connection.client, server.URL, defaultExpectations()[0], 250*time.Millisecond)
		if result.Passed {
			t.Fatalf("HTTP/1 server unexpectedly satisfied h2c prior-knowledge validation: %+v", result)
		}
	}
}
