package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
)

const toolName = "quic-go-raw-load"
const outputSchemaVersion = "protocol-lab.raw-quic-executor-result.v1"
const slowReaderResponseDelay = 100 * time.Millisecond
const downloadRequestMagic = "PLAB-DL1"
const downloadRequestLength = len(downloadRequestMagic) + 8
const maximumDownloadPayloadLength = 64 * 1024 * 1024
const sustainedStreamChunkSize = 64 * 1024

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
	connectLatencies   []float64
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

type identityOutput struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type loadShapeOutput struct {
	Connections          int `json:"connections"`
	Concurrency          int `json:"concurrency"`
	StreamsPerConnection int `json:"streamsPerConnection"`
	DurationSeconds      int `json:"durationSeconds"`
	WarmupSeconds        int `json:"warmupSeconds"`
	Repetitions          int `json:"repetitions"`
}

type outputDocument struct {
	SchemaVersion string          `json:"schemaVersion"`
	Executor      identityOutput  `json:"executor"`
	LoadGenerator identityOutput  `json:"loadGenerator"`
	RequestedLoad loadShapeOutput `json:"requestedLoad"`
	EffectiveLoad loadShapeOutput `json:"effectiveLoad"`
	Tool          string          `json:"tool"`
	Target        string          `json:"target"`
	Behavior      string          `json:"behavior"`
	Metrics       metricsOutput   `json:"metrics"`
	Warnings      []string        `json:"warnings,omitempty"`
	Errors        []string        `json:"errors,omitempty"`
}

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("%s %s\n", executorIdentity().ID, executorIdentity().Version)
		return
	}

	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	warnings := []string{}
	if opts.warmup > 0 {
		if _, warmupErr := runLoad(context.Background(), opts, opts.warmup, true); warmupErr != nil {
			document := newOutputDocument(opts, buildMetrics(runStats{}, time.Nanosecond))
			document.Errors = []string{fmt.Sprintf("warmup failed: %v", warmupErr)}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if encodeErr := encoder.Encode(document); encodeErr != nil {
				fmt.Fprintln(os.Stderr, encodeErr)
			}
			os.Exit(1)
		}
	}

	start := time.Now()
	stats, err := runLoad(context.Background(), opts, opts.duration, false)
	elapsed := time.Since(start)
	if elapsed <= 0 {
		elapsed = time.Nanosecond
	}

	document := newOutputDocument(opts, buildMetrics(stats, elapsed))
	document.Warnings = warnings
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

func newOutputDocument(opts options, metrics metricsOutput) outputDocument {
	shape := requestedLoadShape(opts)
	return outputDocument{
		SchemaVersion: outputSchemaVersion,
		Executor:      executorIdentity(),
		LoadGenerator: loadGeneratorIdentity(),
		RequestedLoad: shape,
		EffectiveLoad: shape,
		Tool:          toolName,
		Target:        opts.target,
		Behavior:      opts.behavior,
		Metrics:       metrics,
	}
}

func executorIdentity() identityOutput {
	return identityOutput{
		ID:      environmentOrDefault("PLAB_EXECUTOR_ID", toolName),
		Version: environmentOrDefault("PLAB_EXECUTOR_VERSION", "development"),
	}
}

func loadGeneratorIdentity() identityOutput {
	return identityOutput{
		ID:      environmentOrDefault("PLAB_LOAD_GENERATOR_ID", toolName),
		Version: environmentOrDefault("PLAB_LOAD_GENERATOR_VERSION", executorIdentity().Version),
	}
}

func requestedLoadShape(opts options) loadShapeOutput {
	concurrency := opts.connections * max(1, opts.streamsPerConnection)
	if value, err := strconv.Atoi(os.Getenv("PLAB_CONCURRENCY")); err == nil && value > 0 {
		concurrency = value
	}

	return loadShapeOutput{
		Connections:          opts.connections,
		Concurrency:          concurrency,
		StreamsPerConnection: opts.streamsPerConnection,
		DurationSeconds:      int(opts.duration / time.Second),
		WarmupSeconds:        int(opts.warmup / time.Second),
		Repetitions:          1,
	}
}

func environmentOrDefault(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
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
	if opts.payloadSizeBytes < 0 {
		return opts, errors.New("payload-size-bytes cannot be negative")
	}
	if opts.duration <= 0 {
		return opts, errors.New("duration must be greater than zero")
	}
	if opts.warmup < 0 {
		return opts, errors.New("warmup cannot be negative")
	}

	if err := validateOptions(opts); err != nil {
		return opts, err
	}

	return opts, nil
}

func validateOptions(opts options) error {
	behavior := strings.ToLower(opts.behavior)
	openPattern := strings.ToLower(opts.openPattern)
	payloadDirection := strings.ToLower(opts.payloadDirection)

	switch payloadDirection {
	case "", "none", "client-to-server", "server-to-client", "bidirectional":
	default:
		return fmt.Errorf("unsupported payload direction %q", opts.payloadDirection)
	}
	if payloadDirection == "server-to-client" && opts.payloadSizeBytes > maximumDownloadPayloadLength {
		return fmt.Errorf("server-to-client payload size exceeds %d-byte limit", maximumDownloadPayloadLength)
	}

	switch behavior {
	case "stream-throughput", "latency-echo", "large-payload", "sustained-stream-256x64kb":
		if openPattern != "sequential" {
			return fmt.Errorf("behavior %q requires open-pattern %q", opts.behavior, "sequential")
		}
		if opts.streamsPerConnection < 1 {
			return errors.New("streams-per-connection must be greater than zero")
		}
	case "multiplex-streams", "duplex-streams", "stream-limit-pressure", "flow-control-slow-reader-100ms":
		if openPattern != "concurrent" {
			return fmt.Errorf("behavior %q requires open-pattern %q", opts.behavior, "concurrent")
		}
		if opts.streamsPerConnection < 1 {
			return errors.New("streams-per-connection must be greater than zero")
		}
	case "handshake-cold":
		if openPattern != "sequential" {
			return fmt.Errorf("behavior %q requires open-pattern %q", opts.behavior, "sequential")
		}
		if opts.streamsPerConnection != 0 {
			return errors.New("handshake-cold requires streams-per-connection to be zero")
		}
	case "connection-churn", "stream-churn":
		if openPattern != "churn" {
			return fmt.Errorf("behavior %q requires open-pattern %q", opts.behavior, "churn")
		}
		if opts.streamsPerConnection < 1 {
			return fmt.Errorf("%s requires streams-per-connection to be greater than zero", behavior)
		}
	default:
		return fmt.Errorf("unsupported behavior %q", opts.behavior)
	}

	return nil
}

func runLoad(ctx context.Context, opts options, duration time.Duration, discard bool) (runStats, error) {
	return runLoadWithQUICConfig(ctx, opts, duration, discard, defaultQUICConfig())
}

func runLoadWithQUICConfig(ctx context.Context, opts options, duration time.Duration, discard bool, quicConfig *quic.Config) (runStats, error) {
	if quicConfig == nil {
		quicConfig = defaultQUICConfig()
	}

	switch strings.ToLower(opts.behavior) {
	case "stream-throughput", "latency-echo", "large-payload", "sustained-stream-256x64kb":
		return runStreamLoad(ctx, opts, duration, discard, streamOpenSequential, quicConfig)
	case "multiplex-streams", "duplex-streams", "stream-limit-pressure", "flow-control-slow-reader-100ms":
		return runStreamLoad(ctx, opts, duration, discard, streamOpenConcurrent, quicConfig)
	case "handshake-cold":
		return runHandshakeColdLoad(ctx, opts, duration, discard, quicConfig)
	case "connection-churn":
		return runConnectionChurnLoad(ctx, opts, duration, discard, quicConfig)
	case "stream-churn":
		return runStreamLoad(ctx, opts, duration, discard, streamOpenSequential, quicConfig)
	default:
		return runStats{}, fmt.Errorf("unsupported behavior %q", opts.behavior)
	}
}

type streamOpenMode int

const (
	streamOpenSequential streamOpenMode = iota
	streamOpenConcurrent
)

func runStreamLoad(ctx context.Context, opts options, duration time.Duration, discard bool, mode streamOpenMode, quicConfig *quic.Config) (runStats, error) {
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
		batch := runStreamBatch(ctx, connections, payload, opts, mode)
		if !discard {
			stats.merge(batch)
		}
		if len(batch.errors) > 0 {
			return stats, errors.New(batch.errors[0])
		}
		if duration <= 0 {
			break
		}
	}

	return stats, nil
}

func runStreamBatch(ctx context.Context, connections []*quic.Conn, payload []byte, opts options, mode streamOpenMode) runStats {
	if mode == streamOpenSequential {
		return runSequentialStreamBatch(ctx, connections, payload, opts)
	}
	return runConcurrentStreamBatch(ctx, connections, payload, opts)
}

func runConcurrentStreamBatch(ctx context.Context, connections []*quic.Conn, payload []byte, opts options) runStats {
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
					if isTimeoutError(err) {
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

func runSequentialStreamBatch(ctx context.Context, connections []*quic.Conn, payload []byte, opts options) runStats {
	var stats runStats
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, conn := range connections {
		conn := conn
		wg.Add(1)
		go func() {
			defer wg.Done()
			batch := runSequentialStreams(ctx, conn, payload, opts)
			mutex.Lock()
			stats.merge(batch)
			mutex.Unlock()
		}()
	}

	wg.Wait()
	return stats
}

func runSequentialStreams(ctx context.Context, conn *quic.Conn, payload []byte, opts options) runStats {
	var stats runStats
	for streamIndex := 0; streamIndex < opts.streamsPerConnection; streamIndex++ {
		latency, bytesSent, bytesReceived, err := runStream(ctx, conn, payload, opts)
		stats.totalRequests++
		stats.bytesSent += bytesSent
		stats.bytesReceived += bytesReceived
		if err != nil {
			stats.failedRequests++
			if isTimeoutError(err) {
				stats.timeoutRequests++
			}
			stats.errors = append(stats.errors, err.Error())
			continue
		}
		stats.successfulRequests++
		stats.latencies = append(stats.latencies, latency)
	}

	return stats
}

func runStream(ctx context.Context, conn *quic.Conn, payload []byte, opts options) (float64, int64, int64, error) {
	if strings.EqualFold(opts.behavior, "duplex-streams") {
		return runDuplexStream(ctx, conn, payload, opts)
	}
	return runRequestResponseStream(ctx, conn, payload, opts)
}

func runRequestResponseStream(ctx context.Context, conn *quic.Conn, payload []byte, opts options) (float64, int64, int64, error) {
	streamTimeout := maxDuration(opts.duration+30*time.Second, 30*time.Second)
	streamCtx, cancel := context.WithTimeout(ctx, streamTimeout)
	defer cancel()

	start := time.Now()
	stream, err := conn.OpenStreamSync(streamCtx)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("open stream: %w", err)
	}
	if err := stream.SetDeadline(time.Now().Add(streamTimeout)); err != nil {
		return 0, 0, 0, fmt.Errorf("set stream deadline: %w", err)
	}
	closeStream := true
	defer func() {
		if closeStream {
			_ = stream.Close()
		}
	}()

	requestPayload := payload
	accountedBytesSent := int64(len(payload))
	if strings.EqualFold(opts.payloadDirection, "server-to-client") {
		requestPayload = buildDownloadRequest(len(payload))
		accountedBytesSent = 0
	}

	var written int
	if strings.EqualFold(opts.behavior, "sustained-stream-256x64kb") {
		written, err = writePayloadInChunks(stream, requestPayload, sustainedStreamChunkSize)
	} else {
		written, err = stream.Write(requestPayload)
	}
	if err != nil {
		return 0, accountedBytesSent, 0, fmt.Errorf("write request payload: %w", err)
	}
	if written != len(requestPayload) {
		return 0, accountedBytesSent, 0, io.ErrShortWrite
	}
	if err := stream.Close(); err != nil {
		return 0, accountedBytesSent, 0, fmt.Errorf("close request writes: %w", err)
	}
	closeStream = false

	if delay := responseReadDelay(opts.behavior); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-streamCtx.Done():
			return 0, accountedBytesSent, 0, fmt.Errorf("wait before reading response: %w", streamCtx.Err())
		case <-timer.C:
		}
	}

	var received int64
	buffer := make([]byte, 64*1024)
	for {
		read, readErr := stream.Read(buffer)
		if read > 0 {
			if !strings.EqualFold(opts.payloadDirection, "client-to-server") {
				if err := validateDeterministicPayloadChunk(payload, received, buffer[:read]); err != nil {
					return 0, accountedBytesSent, received + int64(read), fmt.Errorf("validate response payload: %w", err)
				}
			}
			received += int64(read)
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		return 0, accountedBytesSent, received, fmt.Errorf("read response payload or EOF: %w", readErr)
	}

	if strings.EqualFold(opts.payloadDirection, "client-to-server") {
		if received != 0 {
			return 0, accountedBytesSent, received, fmt.Errorf("received %d response bytes, expected 0", received)
		}
	} else if received != int64(len(payload)) {
		return 0, accountedBytesSent, received, fmt.Errorf("received %d response bytes, expected %d", received, len(payload))
	}

	return float64(time.Since(start).Microseconds()) / 1000.0, accountedBytesSent, received, nil
}

func writePayloadInChunks(writer io.Writer, payload []byte, chunkSize int) (int, error) {
	if chunkSize <= 0 {
		return 0, errors.New("chunk size must be greater than zero")
	}

	totalWritten := 0
	for totalWritten < len(payload) {
		chunkEnd := min(totalWritten+chunkSize, len(payload))
		chunk := payload[totalWritten:chunkEnd]
		written, err := writer.Write(chunk)
		totalWritten += written
		if err != nil {
			return totalWritten, err
		}
		if written != len(chunk) {
			return totalWritten, io.ErrShortWrite
		}
	}

	return totalWritten, nil
}

func buildDownloadRequest(payloadLength int) []byte {
	request := make([]byte, downloadRequestLength)
	copy(request, downloadRequestMagic)
	binary.BigEndian.PutUint64(request[len(downloadRequestMagic):], uint64(payloadLength))
	return request
}

func parseDownloadRequest(request []byte, maximumPayloadLength int) (int, bool) {
	if len(request) != downloadRequestLength || string(request[:len(downloadRequestMagic)]) != downloadRequestMagic {
		return 0, false
	}

	payloadLength := binary.BigEndian.Uint64(request[len(downloadRequestMagic):])
	if payloadLength == 0 || payloadLength > uint64(maximumPayloadLength) {
		return 0, false
	}

	return int(payloadLength), true
}

func validateDeterministicPayloadChunk(expected []byte, offset int64, chunk []byte) error {
	if offset < 0 || offset+int64(len(chunk)) > int64(len(expected)) {
		return fmt.Errorf("response exceeded expected %d-byte payload at offset %d", len(expected), offset)
	}

	for index, actual := range chunk {
		if wanted := expected[int(offset)+index]; actual != wanted {
			return fmt.Errorf("byte at offset %d was %d, expected %d", int(offset)+index, actual, wanted)
		}
	}

	return nil
}

func responseReadDelay(behavior string) time.Duration {
	if strings.EqualFold(behavior, "flow-control-slow-reader-100ms") {
		return slowReaderResponseDelay
	}
	return 0
}

func runDuplexStream(ctx context.Context, conn *quic.Conn, payload []byte, opts options) (float64, int64, int64, error) {
	streamTimeout := maxDuration(opts.duration+30*time.Second, 30*time.Second)
	streamCtx, cancel := context.WithTimeout(ctx, streamTimeout)
	defer cancel()

	start := time.Now()
	stream, err := conn.OpenStreamSync(streamCtx)
	if err != nil {
		return 0, 0, 0, err
	}
	if err := stream.SetDeadline(time.Now().Add(streamTimeout)); err != nil {
		return 0, 0, 0, err
	}
	closeStream := true
	type readResult struct {
		received int64
		err      error
	}
	readResults := make(chan readResult, 1)
	go func() {
		var result readResult
		buffer := make([]byte, 64*1024)
		for {
			read, err := stream.Read(buffer)
			if read > 0 {
				result.received += int64(read)
			}
			if err == nil {
				continue
			}
			if errors.Is(err, io.EOF) {
				readResults <- result
				return
			}
			result.err = err
			readResults <- result
			return
		}
	}()
	defer func() {
		if closeStream {
			_ = stream.Close()
		}
	}()

	written, err := stream.Write(payload)
	if err != nil {
		_ = stream.Close()
		closeStream = false
		result := <-readResults
		return 0, int64(written), result.received, err
	}
	if written != len(payload) {
		_ = stream.Close()
		closeStream = false
		result := <-readResults
		return 0, int64(written), result.received, io.ErrShortWrite
	}
	if err := stream.Close(); err != nil {
		closeStream = false
		result := <-readResults
		return 0, int64(written), result.received, err
	}
	closeStream = false

	result := <-readResults
	received := result.received
	if strings.EqualFold(opts.payloadDirection, "client-to-server") {
		if received != 0 {
			return 0, int64(written), received, fmt.Errorf("received %d response bytes, expected 0", received)
		}
	} else if received != int64(len(payload)) {
		return 0, int64(written), received, fmt.Errorf("received %d echoed bytes, expected %d", received, len(payload))
	}
	if result.err != nil {
		return 0, int64(written), received, result.err
	}

	return float64(time.Since(start).Microseconds()) / 1000.0, int64(written), received, nil
}

func runHandshakeColdLoad(ctx context.Context, opts options, duration time.Duration, discard bool, quicConfig *quic.Config) (runStats, error) {
	targetURL, err := url.Parse(opts.target)
	if err != nil {
		return runStats{}, err
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Local ProtocolLab raw QUIC endpoints use generated self-signed certificates.
		NextProtos:         []string{opts.alpn},
		ServerName:         opts.sni,
	}

	stats := runStats{
		connectLatencies: make([]float64, 0, opts.connections),
	}
	deadline := time.Now().Add(duration)
	firstBatch := true
	for firstBatch || time.Now().Before(deadline) {
		firstBatch = false
		batch := runHandshakeColdBatch(ctx, targetURL.Host, tlsConfig, quicConfig, opts)
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

func runHandshakeColdBatch(ctx context.Context, targetHost string, tlsConfig *tls.Config, quicConfig *quic.Config, opts options) runStats {
	var stats runStats
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < opts.connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, connectTimeMs, err := dialQUICConn(ctx, targetHost, tlsConfig, quicConfig)
			mutex.Lock()
			stats.totalRequests++
			if err != nil {
				stats.failedRequests++
				if isTimeoutError(err) {
					stats.timeoutRequests++
				}
				stats.errors = append(stats.errors, err.Error())
				mutex.Unlock()
				return
			}
			stats.successfulRequests++
			stats.connectLatencies = append(stats.connectLatencies, connectTimeMs)
			mutex.Unlock()
			_ = conn.CloseWithError(0, "")
		}()
	}

	wg.Wait()
	return stats
}

func runConnectionChurnLoad(ctx context.Context, opts options, duration time.Duration, discard bool, quicConfig *quic.Config) (runStats, error) {
	targetURL, err := url.Parse(opts.target)
	if err != nil {
		return runStats{}, err
	}

	payload := make([]byte, opts.payloadSizeBytes)
	for i := range payload {
		payload[i] = byte(i % 251)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Local ProtocolLab raw QUIC endpoints use generated self-signed certificates.
		NextProtos:         []string{opts.alpn},
		ServerName:         opts.sni,
	}

	stats := runStats{
		latencies:        make([]float64, 0, opts.connections*opts.streamsPerConnection),
		connectLatencies: make([]float64, 0, opts.connections),
	}
	deadline := time.Now().Add(duration)
	firstBatch := true
	for firstBatch || time.Now().Before(deadline) {
		firstBatch = false
		batch := runConnectionChurnBatch(ctx, targetURL.Host, tlsConfig, quicConfig, payload, opts)
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

func runConnectionChurnBatch(ctx context.Context, targetHost string, tlsConfig *tls.Config, quicConfig *quic.Config, payload []byte, opts options) runStats {
	var stats runStats
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < opts.connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, connectTimeMs, err := dialQUICConn(ctx, targetHost, tlsConfig, quicConfig)
			if err != nil {
				failedRequests := int64(opts.streamsPerConnection)
				if failedRequests < 1 {
					failedRequests = 1
				}
				batch := runStats{
					totalRequests:   failedRequests,
					failedRequests:  failedRequests,
					timeoutRequests: boolToInt64(isTimeoutError(err)) * failedRequests,
					errors:          []string{err.Error()},
				}
				mutex.Lock()
				stats.merge(batch)
				mutex.Unlock()
				return
			}
			defer func() {
				_ = conn.CloseWithError(0, "")
			}()

			batch := runSequentialStreams(ctx, conn, payload, opts)
			batch.connectLatencies = append(batch.connectLatencies, connectTimeMs)
			mutex.Lock()
			stats.merge(batch)
			mutex.Unlock()
		}()
	}

	wg.Wait()
	return stats
}

func dialQUICConn(ctx context.Context, targetHost string, tlsConfig *tls.Config, quicConfig *quic.Config) (*quic.Conn, float64, error) {
	start := time.Now()
	conn, err := quic.DialAddr(ctx, targetHost, tlsConfig, quicConfig)
	if err != nil {
		return nil, 0, err
	}
	return conn, float64(time.Since(start).Microseconds()) / 1000.0, nil
}

func defaultQUICConfig() *quic.Config {
	return &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 5 * time.Second,
	}
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func (s *runStats) merge(other runStats) {
	s.totalRequests += other.totalRequests
	s.successfulRequests += other.successfulRequests
	s.failedRequests += other.failedRequests
	s.timeoutRequests += other.timeoutRequests
	s.bytesSent += other.bytesSent
	s.bytesReceived += other.bytesReceived
	s.latencies = append(s.latencies, other.latencies...)
	s.connectLatencies = append(s.connectLatencies, other.connectLatencies...)
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
		ConnectTimeMeanMs:        mean(stats.connectLatencies),
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

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
