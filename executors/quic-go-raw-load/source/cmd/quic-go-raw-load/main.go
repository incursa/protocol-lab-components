package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
)

const toolName = "quic-go-raw-load"

type options struct {
	sni                  string
	alpn                 string
	behavior             string
	streamType           string
	payloadSizeBytes     int
	payloadDirection     string
	openPattern          string
	connections          int
	streamsPerConnection int
	duration             time.Duration
	warmup               time.Duration
	target               string
	failOnErrors         bool
}

type runStats struct {
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	timeoutRequests    int64
	bytesSent          int64
	bytesReceived      int64
	latencies          []float64
	errors             []string
}

type metricsOutput struct {
	RequestsPerSecond        float64 `json:"requestsPerSecond"`
	TotalRequests            int64   `json:"totalRequests"`
	SuccessfulRequests       int64   `json:"successfulRequests"`
	FailedRequests           int64   `json:"failedRequests"`
	TimeoutRequests          int64   `json:"timeoutRequests"`
	BytesSent                int64   `json:"bytesSent"`
	BytesReceived            int64   `json:"bytesReceived"`
	ThroughputBytesPerSecond float64 `json:"throughputBytesPerSecond"`
	LatencyMinMs             float64 `json:"latencyMinMs"`
	LatencyMeanMs            float64 `json:"latencyMeanMs"`
	LatencyP50Ms             float64 `json:"latencyP50Ms"`
	LatencyP75Ms             float64 `json:"latencyP75Ms"`
	LatencyP90Ms             float64 `json:"latencyP90Ms"`
	LatencyP95Ms             float64 `json:"latencyP95Ms"`
	LatencyP99Ms             float64 `json:"latencyP99Ms"`
	LatencyMaxMs             float64 `json:"latencyMaxMs"`
	ConnectTimeMeanMs        float64 `json:"connectTimeMeanMs"`
	TimeToFirstByteMeanMs    float64 `json:"timeToFirstByteMeanMs"`
	ErrorRate                float64 `json:"errorRate"`
}

type outputDocument struct {
	Tool     string        `json:"tool"`
	Target   string        `json:"target"`
	Behavior string        `json:"behavior"`
	Metrics  metricsOutput `json:"metrics"`
	Warnings []string      `json:"warnings,omitempty"`
	Errors   []string      `json:"errors,omitempty"`
}

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	warnings := validateOptions(opts)
	if opts.warmup > 0 {
		if _, err := runLoad(context.Background(), opts, opts.warmup, true); err != nil {
			warnings = append(warnings, fmt.Sprintf("warmup failed: %v", err))
		}
	}

	start := time.Now()
	stats, err := runLoad(context.Background(), opts, opts.duration, false)
	elapsed := time.Since(start)
	if elapsed <= 0 {
		elapsed = time.Nanosecond
	}

	document := outputDocument{
		Tool:     toolName,
		Target:   opts.target,
		Behavior: opts.behavior,
		Metrics:  buildMetrics(stats, elapsed),
		Warnings: warnings,
	}
	if len(stats.errors) > 0 {
		document.Errors = stats.errors
	}
	if err != nil {
		document.Errors = append(document.Errors, err.Error())
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if encodeErr := encoder.Encode(document); encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}

	if stats.successfulRequests == 0 || (opts.failOnErrors && (err != nil || stats.failedRequests > 0 || stats.timeoutRequests > 0)) {
		os.Exit(1)
	}
}

func parseOptions(args []string) (options, error) {
	opts := options{
		sni:                  "localhost",
		alpn:                 "plab-raw-quic",
		behavior:             "multiplex-streams",
		streamType:           "bidirectional",
		payloadSizeBytes:     65536,
		payloadDirection:     "bidirectional",
		openPattern:          "concurrent",
		connections:          1,
		streamsPerConnection: 1,
		duration:             time.Second,
	}

	fs := flag.NewFlagSet(toolName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.sni, "sni", opts.sni, "TLS SNI value")
	fs.StringVar(&opts.alpn, "alpn", opts.alpn, "QUIC ALPN value")
	fs.StringVar(&opts.behavior, "behavior", opts.behavior, "raw QUIC behavior")
	fs.StringVar(&opts.streamType, "stream-type", opts.streamType, "QUIC stream type")
	fs.IntVar(&opts.payloadSizeBytes, "payload-size-bytes", opts.payloadSizeBytes, "payload size per stream")
	fs.StringVar(&opts.payloadDirection, "payload-direction", opts.payloadDirection, "payload direction")
	fs.StringVar(&opts.openPattern, "open-pattern", opts.openPattern, "stream open pattern")
	fs.IntVar(&opts.connections, "connections", opts.connections, "QUIC connections")
	fs.IntVar(&opts.streamsPerConnection, "streams-per-connection", opts.streamsPerConnection, "streams per connection")
	fs.DurationVar(&opts.duration, "duration", opts.duration, "measurement duration")
	fs.DurationVar(&opts.warmup, "warmup", opts.warmup, "warmup duration")
	fs.BoolVar(&opts.failOnErrors, "fail-on-errors", opts.failOnErrors, "exit non-zero when measured requests include stream errors or timeouts")

	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if fs.NArg() != 1 {
		return opts, errors.New("expected exactly one quic:// target URL")
	}
	opts.target = fs.Arg(0)

	targetURL, err := url.Parse(opts.target)
	if err != nil {
		return opts, fmt.Errorf("invalid target URL: %w", err)
	}
	if !strings.EqualFold(targetURL.Scheme, "quic") {
		return opts, fmt.Errorf("unsupported target URL scheme %q", targetURL.Scheme)
	}
	if targetURL.Host == "" {
		return opts, errors.New("target URL must include host and port")
	}
	if opts.connections < 1 {
		return opts, errors.New("connections must be greater than zero")
	}
	if opts.streamsPerConnection < 1 {
		return opts, errors.New("streams-per-connection must be greater than zero")
	}
	if opts.payloadSizeBytes < 0 {
		return opts, errors.New("payload-size-bytes cannot be negative")
	}
	if opts.duration <= 0 {
		return opts, errors.New("duration must be greater than zero")
	}
	if opts.warmup < 0 {
		return opts, errors.New("warmup cannot be negative")
	}

	return opts, nil
}

func validateOptions(opts options) []string {
	warnings := make([]string, 0)
	if !strings.EqualFold(opts.streamType, "bidirectional") {
		warnings = append(warnings, "only bidirectional streams are implemented; using bidirectional streams")
	}
	if !strings.EqualFold(opts.openPattern, "concurrent") {
		warnings = append(warnings, "only concurrent stream opening is implemented; using concurrent batches")
	}
	if !strings.EqualFold(opts.payloadDirection, "bidirectional") {
		warnings = append(warnings, "non-bidirectional payload directions are measured, but response bytes may be zero")
	}
	return warnings
}

func runLoad(ctx context.Context, opts options, duration time.Duration, discard bool) (runStats, error) {
	targetURL, err := url.Parse(opts.target)
	if err != nil {
		return runStats{}, err
	}
	payload := make([]byte, opts.payloadSizeBytes)
	for i := range payload {
		payload[i] = byte(i % 251)
	}

	connections := make([]*quic.Conn, 0, opts.connections)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Local ProtocolLab raw QUIC endpoints use generated self-signed certificates.
		NextProtos:         []string{opts.alpn},
		ServerName:         opts.sni,
	}
	quicConfig := &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 5 * time.Second,
	}

	for i := 0; i < opts.connections; i++ {
		conn, dialErr := quic.DialAddr(ctx, targetURL.Host, tlsConfig, quicConfig)
		if dialErr != nil {
			return runStats{failedRequests: int64(opts.connections * opts.streamsPerConnection), errors: []string{dialErr.Error()}}, dialErr
		}
		connections = append(connections, conn)
	}
	defer func() {
		for _, conn := range connections {
			_ = conn.CloseWithError(0, "")
		}
	}()

	stats := runStats{
		latencies: make([]float64, 0, opts.connections*opts.streamsPerConnection),
	}
	deadline := time.Now().Add(duration)
	firstBatch := true
	for firstBatch || time.Now().Before(deadline) {
		firstBatch = false
		batch := runBatch(ctx, connections, payload, opts)
		if !discard {
			stats.merge(batch)
		}
		if len(batch.errors) > 0 && !discard {
			return stats, errors.New(batch.errors[0])
		}
		if duration <= 0 {
			break
		}
	}

	return stats, nil
}

func runBatch(ctx context.Context, connections []*quic.Conn, payload []byte, opts options) runStats {
	var stats runStats
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, conn := range connections {
		for streamIndex := 0; streamIndex < opts.streamsPerConnection; streamIndex++ {
			wg.Add(1)
			go func(conn *quic.Conn) {
				defer wg.Done()
				latency, bytesSent, bytesReceived, err := runStream(ctx, conn, payload, opts)
				atomic.AddInt64(&stats.totalRequests, 1)
				atomic.AddInt64(&stats.bytesSent, bytesSent)
				atomic.AddInt64(&stats.bytesReceived, bytesReceived)
				if err != nil {
					atomic.AddInt64(&stats.failedRequests, 1)
					if errors.Is(err, context.DeadlineExceeded) {
						atomic.AddInt64(&stats.timeoutRequests, 1)
					}
					mutex.Lock()
					stats.errors = append(stats.errors, err.Error())
					mutex.Unlock()
					return
				}
				atomic.AddInt64(&stats.successfulRequests, 1)
				mutex.Lock()
				stats.latencies = append(stats.latencies, latency)
				mutex.Unlock()
			}(conn)
		}
	}

	wg.Wait()
	return stats
}

func runStream(ctx context.Context, conn *quic.Conn, payload []byte, opts options) (float64, int64, int64, error) {
	streamCtx, cancel := context.WithTimeout(ctx, maxDuration(opts.duration+30*time.Second, 30*time.Second))
	defer cancel()

	start := time.Now()
	stream, err := conn.OpenStreamSync(streamCtx)
	if err != nil {
		return 0, 0, 0, err
	}
	defer stream.Close()

	written, err := stream.Write(payload)
	if err != nil {
		return 0, int64(written), 0, err
	}
	if written != len(payload) {
		return 0, int64(written), 0, io.ErrShortWrite
	}
	if err := stream.Close(); err != nil {
		return 0, int64(written), 0, err
	}

	var received int64
	if !strings.EqualFold(opts.payloadDirection, "client-to-server") {
		buffer := make([]byte, 64*1024)
		for {
			read, readErr := stream.Read(buffer)
			if read > 0 {
				received += int64(read)
			}
			if readErr == nil {
				continue
			}
			if errors.Is(readErr, io.EOF) {
				break
			}
			return 0, int64(written), received, readErr
		}
		if received != int64(len(payload)) {
			return 0, int64(written), received, fmt.Errorf("received %d echoed bytes, expected %d", received, len(payload))
		}
	}

	return float64(time.Since(start).Microseconds()) / 1000.0, int64(written), received, nil
}

func (s *runStats) merge(other runStats) {
	s.totalRequests += other.totalRequests
	s.successfulRequests += other.successfulRequests
	s.failedRequests += other.failedRequests
	s.timeoutRequests += other.timeoutRequests
	s.bytesSent += other.bytesSent
	s.bytesReceived += other.bytesReceived
	s.latencies = append(s.latencies, other.latencies...)
	s.errors = append(s.errors, other.errors...)
}

func buildMetrics(stats runStats, elapsed time.Duration) metricsOutput {
	elapsedSeconds := elapsed.Seconds()
	if elapsedSeconds <= 0 {
		elapsedSeconds = 1
	}
	errorRate := 0.0
	if stats.totalRequests > 0 {
		errorRate = float64(stats.failedRequests) / float64(stats.totalRequests)
	}

	sort.Float64s(stats.latencies)
	throughputBytes := stats.bytesReceived
	if stats.bytesSent > throughputBytes {
		throughputBytes = stats.bytesSent
	}

	return metricsOutput{
		RequestsPerSecond:        float64(stats.successfulRequests) / elapsedSeconds,
		TotalRequests:            stats.totalRequests,
		SuccessfulRequests:       stats.successfulRequests,
		FailedRequests:           stats.failedRequests,
		TimeoutRequests:          stats.timeoutRequests,
		BytesSent:                stats.bytesSent,
		BytesReceived:            stats.bytesReceived,
		ThroughputBytesPerSecond: float64(throughputBytes) / elapsedSeconds,
		LatencyMinMs:             percentile(stats.latencies, 0),
		LatencyMeanMs:            mean(stats.latencies),
		LatencyP50Ms:             percentile(stats.latencies, 50),
		LatencyP75Ms:             percentile(stats.latencies, 75),
		LatencyP90Ms:             percentile(stats.latencies, 90),
		LatencyP95Ms:             percentile(stats.latencies, 95),
		LatencyP99Ms:             percentile(stats.latencies, 99),
		LatencyMaxMs:             percentile(stats.latencies, 100),
		ConnectTimeMeanMs:        0,
		TimeToFirstByteMeanMs:    0,
		ErrorRate:                errorRate,
	}
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if p <= 0 {
		return values[0]
	}
	if p >= 100 {
		return values[len(values)-1]
	}
	position := (p / 100) * float64(len(values)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return values[lower]
	}
	weight := position - float64(lower)
	return values[lower]*(1-weight) + values[upper]*weight
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func maxDuration(a time.Duration, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
