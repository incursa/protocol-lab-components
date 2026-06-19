package main

import (
	"sort"
	"testing"
	"time"
)

func TestParseOptionsMapsProtocolLabArguments(t *testing.T) {
	opts, err := parseOptions([]string{
		"--sni", "localhost",
		"--alpn", "plab-raw-quic",
		"--behavior", "multiplex-streams",
		"--stream-type", "bidirectional",
		"--payload-size-bytes", "65536",
		"--payload-direction", "bidirectional",
		"--open-pattern", "concurrent",
		"--connections", "1",
		"--streams-per-connection", "100",
		"--duration", "30s",
		"--warmup", "5s",
		"--fail-on-errors",
		"quic://127.0.0.1:4433/",
	})
	if err != nil {
		t.Fatalf("parseOptions returned error: %v", err)
	}

	if opts.connections != 1 {
		t.Fatalf("connections = %d, want 1", opts.connections)
	}
	if opts.streamsPerConnection != 100 {
		t.Fatalf("streamsPerConnection = %d, want 100", opts.streamsPerConnection)
	}
	if opts.duration != 30*time.Second {
		t.Fatalf("duration = %s, want 30s", opts.duration)
	}
	if opts.warmup != 5*time.Second {
		t.Fatalf("warmup = %s, want 5s", opts.warmup)
	}
	if !opts.failOnErrors {
		t.Fatal("failOnErrors = false, want true")
	}
	if opts.target != "quic://127.0.0.1:4433/" {
		t.Fatalf("target = %q", opts.target)
	}
}

func TestBuildMetricsComputesRawQuicFields(t *testing.T) {
	stats := runStats{
		totalRequests:      4,
		successfulRequests: 3,
		failedRequests:     1,
		bytesSent:          12,
		bytesReceived:      9,
		latencies:          []float64{10, 20, 30},
	}
	sort.Float64s(stats.latencies)

	metrics := buildMetrics(stats, 3*time.Second)

	if metrics.RequestsPerSecond != 1 {
		t.Fatalf("requestsPerSecond = %f, want 1", metrics.RequestsPerSecond)
	}
	if metrics.ThroughputBytesPerSecond != 4 {
		t.Fatalf("throughputBytesPerSecond = %f, want 4", metrics.ThroughputBytesPerSecond)
	}
	if metrics.ErrorRate != 0.25 {
		t.Fatalf("errorRate = %f, want 0.25", metrics.ErrorRate)
	}
	if metrics.LatencyP50Ms != 20 {
		t.Fatalf("LatencyP50Ms = %f, want 20", metrics.LatencyP50Ms)
	}
	if metrics.LatencyP95Ms != 29 {
		t.Fatalf("LatencyP95Ms = %f, want 29", metrics.LatencyP95Ms)
	}
}
