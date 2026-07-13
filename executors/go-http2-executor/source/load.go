package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http/httptrace"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

type h2CConnection struct {
	client *http2.ClientConn
	raw    net.Conn
}

func (connection *h2CConnection) Close() {
	if connection == nil {
		return
	}
	if connection.client != nil {
		_ = connection.client.Close()
	}
	if connection.raw != nil {
		_ = connection.raw.Close()
	}
}

func dialH2C(ctx context.Context, baseURL string, timeout time.Duration) (*h2CConnection, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(parsed.Scheme, "http") || parsed.Hostname() == "" {
		return nil, errors.New("h2c prior-knowledge targets must use an http URL with a host")
	}
	port := parsed.Port()
	if port == "" {
		port = "80"
	}
	address := net.JoinHostPort(parsed.Hostname(), port)
	raw, err := (&net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial h2c target %s: %w", address, err)
	}
	transport := &http2.Transport{AllowHTTP: true}
	client, err := transport.NewClientConn(raw)
	if err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("create HTTP/2 prior-knowledge client connection: %w", err)
	}
	return &h2CConnection{client: client, raw: raw}, nil
}

type loadConfig struct {
	ScenarioID            string
	LoadProfileID         string
	Connections           int
	Concurrency           int
	StreamsPerConnection  int
	OperationDistribution string
	Duration              time.Duration
	Warmup                time.Duration
	Repetition            int
	RequestTimeout        time.Duration
	ExecutionTimeout      time.Duration
}

type componentIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type loadGeneratorIdentity struct {
	ID                  string `json:"id"`
	Version             string `json:"version"`
	SHA256              string `json:"sha256"`
	Command             string `json:"command"`
	EngineModule        string `json:"engineModule"`
	EngineModuleVersion string `json:"engineModuleVersion"`
}

type validitySummary struct {
	Status string `json:"status"`
}

type normalizedProtocolProof struct {
	RequestedProtocol                         string `json:"requestedProtocol"`
	ObservedProtocol                          string `json:"observedProtocol"`
	ExecutionVariant                          string `json:"executionVariant"`
	ExactProtocolMatched                      bool   `json:"exactProtocolMatched"`
	FallbackDetected                          bool   `json:"fallbackDetected"`
	ObservedDials                             int    `json:"observedDials"`
	ConfiguredStreamsPerConnection            int    `json:"configuredStreamsPerConnection"`
	MaximumActiveOperations                   int    `json:"maximumActiveOperations"`
	MaximumObservedActiveStreams              int    `json:"maximumObservedActiveStreams"`
	MinimumPeerAdvertisedMaxConcurrentStreams uint32 `json:"minimumPeerAdvertisedMaxConcurrentStreams"`
}

type normalizedLoadShape struct {
	Connections                                    int      `json:"connections"`
	Concurrency                                    int      `json:"concurrency"`
	StreamsPerConnection                           int      `json:"streamsPerConnection"`
	DurationSeconds                                float64  `json:"durationSeconds"`
	WarmupSeconds                                  float64  `json:"warmupSeconds"`
	RequestTimeoutSeconds                          float64  `json:"requestTimeoutSeconds"`
	Repetition                                     int      `json:"repetition"`
	OperationDistribution                          string   `json:"operationDistribution"`
	MaximumActiveStreamsByConnection               []int    `json:"maximumActiveStreamsByConnection,omitempty"`
	PeerAdvertisedMaxConcurrentStreamsByConnection []uint32 `json:"peerAdvertisedMaxConcurrentStreamsByConnection,omitempty"`
	Clamped                                        bool     `json:"clamped"`
	Redistributed                                  bool     `json:"redistributed"`
}

type committedLoadProfile struct {
	Connections           int
	Concurrency           int
	StreamsPerConnection  int
	DurationSeconds       int
	WarmupSeconds         int
	RequestTimeoutSeconds int
	OperationDistribution string
	MaximumRepetition     int
}

type http2TopologyArtifact struct {
	SchemaVersion string              `json:"schemaVersion"`
	LoadProfileID string              `json:"loadProfileId"`
	Requested     normalizedLoadShape `json:"requested"`
	Effective     normalizedLoadShape `json:"effective"`
}

type connectionScheduler struct {
	mu     sync.Mutex
	next   int
	slots  []chan struct{}
	notify chan struct{}
}

func newConnectionScheduler(connectionCount, streamsPerConnection int) *connectionScheduler {
	slots := make([]chan struct{}, connectionCount)
	for index := range slots {
		slots[index] = make(chan struct{}, streamsPerConnection)
	}
	return &connectionScheduler{slots: slots, notify: make(chan struct{}, 1)}
}

func (scheduler *connectionScheduler) acquire(ctx context.Context) (int, error) {
	for {
		scheduler.mu.Lock()
		for offset := 0; offset < len(scheduler.slots); offset++ {
			index := (scheduler.next + offset) % len(scheduler.slots)
			select {
			case scheduler.slots[index] <- struct{}{}:
				scheduler.next = (index + 1) % len(scheduler.slots)
				scheduler.mu.Unlock()
				return index, nil
			default:
			}
		}
		scheduler.mu.Unlock()
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-scheduler.notify:
		}
	}
}

func (scheduler *connectionScheduler) release(index int) {
	<-scheduler.slots[index]
	select {
	case scheduler.notify <- struct{}{}:
	default:
	}
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
	TimeToFirstByteMeanMS    float64          `json:"timeToFirstByteMeanMs"`
	StatusCodeCounts         map[string]int64 `json:"statusCodeCounts"`
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

type phaseSummary struct {
	Phase                                    string           `json:"phase"`
	StartedAtUTC                             string           `json:"startedAtUtc"`
	DurationSeconds                          float64          `json:"durationSeconds"`
	ObservedDials                            int              `json:"observedDials"`
	MaximumActiveOperations                  int              `json:"maximumActiveOperations"`
	MaximumActiveOperationsByDial            []int            `json:"maximumActiveOperationsByDial"`
	MaximumObservedActiveStreamsByDial       []int            `json:"maximumObservedActiveStreamsByDial"`
	MaximumObservedPendingStreamsByDial      []int            `json:"maximumObservedPendingStreamsByDial"`
	PeerAdvertisedMaxConcurrentStreamsByDial []uint32         `json:"peerAdvertisedMaxConcurrentStreamsByDial"`
	TotalRequests                            int64            `json:"totalRequests"`
	SuccessfulRequests                       int64            `json:"successfulRequests"`
	FailedRequests                           int64            `json:"failedRequests"`
	TimeoutRequests                          int64            `json:"timeoutRequests"`
	BytesReceived                            int64            `json:"bytesReceived"`
	StatusCodeCounts                         map[string]int64 `json:"statusCodeCounts"`
	LatencySamplesMilliseconds               []float64        `json:"latencySamplesMilliseconds"`
	FirstByteSamplesMilliseconds             []float64        `json:"firstByteSamplesMilliseconds"`
	Errors                                   map[string]int64 `json:"errors"`
}

type phaseRecorder struct {
	mu                     sync.Mutex
	summary                phaseSummary
	activeOperations       int
	activeOperationsByDial []int
}

func loadConfigFromEnvironment() (loadConfig, error) {
	config := loadConfig{
		ScenarioID:            strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")),
		LoadProfileID:         strings.TrimSpace(os.Getenv("PLAB_LOAD_PROFILE_ID")),
		Connections:           envInt("PLAB_CONNECTIONS", 0),
		Concurrency:           envInt("PLAB_CONCURRENCY", 0),
		StreamsPerConnection:  envInt("PLAB_STREAMS_PER_CONNECTION", 0),
		OperationDistribution: strings.TrimSpace(os.Getenv("PLAB_OPERATION_DISTRIBUTION")),
		Duration:              time.Duration(envInt("PLAB_DURATION_SECONDS", 0)) * time.Second,
		Warmup:                time.Duration(envInt("PLAB_WARMUP_SECONDS", 0)) * time.Second,
		Repetition:            envInt("PLAB_REPETITION", 1),
		RequestTimeout:        time.Duration(envInt("PLAB_REQUEST_TIMEOUT_SECONDS", 0)) * time.Second,
		ExecutionTimeout:      time.Duration(envInt("PLAB_TIMEOUT_SECONDS", 0)) * time.Second,
	}
	if config.ScenarioID == "" {
		return config, errors.New("PLAB_SCENARIO_ID is required for performance execution")
	}
	expectedProfiles := map[string]committedLoadProfile{
		"http2-smoke": {
			Connections: 1, Concurrency: 1, StreamsPerConnection: 1,
			DurationSeconds: 5, WarmupSeconds: 1, RequestTimeoutSeconds: 5,
			OperationDistribution: "balanced-round-robin", MaximumRepetition: 1,
		},
		"http2-diagnostic": {
			Connections: 1, Concurrency: 8, StreamsPerConnection: 8,
			DurationSeconds: 10, WarmupSeconds: 1, RequestTimeoutSeconds: 10,
			OperationDistribution: "balanced-round-robin", MaximumRepetition: 1,
		},
		"http2-comparison": {
			Connections: 16, Concurrency: 128, StreamsPerConnection: 8,
			DurationSeconds: 30, WarmupSeconds: 10, RequestTimeoutSeconds: 10,
			OperationDistribution: "balanced-round-robin", MaximumRepetition: 3,
		},
	}
	expectedProfile, supportedProfile := expectedProfiles[config.LoadProfileID]
	if !supportedProfile {
		return config, fmt.Errorf("load profile %q is unsupported; go-http2-executor 0.3.0 accepts http2-smoke, http2-diagnostic, and http2-comparison only", config.LoadProfileID)
	}
	if config.Connections != expectedProfile.Connections || config.Concurrency != expectedProfile.Concurrency || config.StreamsPerConnection != expectedProfile.StreamsPerConnection {
		return config, fmt.Errorf("HTTP/2 profile %q requires connections=%d, concurrency=%d, streamsPerConnection=%d; observed %d/%d/%d", config.LoadProfileID, expectedProfile.Connections, expectedProfile.Concurrency, expectedProfile.StreamsPerConnection, config.Connections, config.Concurrency, config.StreamsPerConnection)
	}
	if config.Duration != time.Duration(expectedProfile.DurationSeconds)*time.Second || config.Warmup != time.Duration(expectedProfile.WarmupSeconds)*time.Second || config.RequestTimeout != time.Duration(expectedProfile.RequestTimeoutSeconds)*time.Second {
		return config, fmt.Errorf("HTTP/2 profile %q requires duration=%ds, warmup=%ds, requestTimeout=%ds; observed %s/%s/%s", config.LoadProfileID, expectedProfile.DurationSeconds, expectedProfile.WarmupSeconds, expectedProfile.RequestTimeoutSeconds, config.Duration, config.Warmup, config.RequestTimeout)
	}
	if config.OperationDistribution == "" && config.LoadProfileID != "http2-comparison" {
		config.OperationDistribution = expectedProfile.OperationDistribution
	}
	if config.OperationDistribution != expectedProfile.OperationDistribution {
		return config, fmt.Errorf("HTTP/2 profile %q requires operationDistribution=%s; observed %q", config.LoadProfileID, expectedProfile.OperationDistribution, config.OperationDistribution)
	}
	if config.Repetition < 1 || config.Repetition > expectedProfile.MaximumRepetition {
		return config, fmt.Errorf("HTTP/2 profile %q requires repetition in [1,%d]; observed %d", config.LoadProfileID, expectedProfile.MaximumRepetition, config.Repetition)
	}
	if config.ExecutionTimeout <= 0 {
		config.ExecutionTimeout = config.Duration + config.Warmup + 30*time.Second
	}
	return config, nil
}

func runH2CLoad(targetBaseURL, outputDir string, expectation scenarioExpectation, config loadConfig) (executorResult, error) {
	if config.Warmup > 0 {
		warmupContext, cancel := context.WithTimeout(context.Background(), config.Warmup+30*time.Second)
		warmup, err := runLoadPhase(warmupContext, "warmup", targetBaseURL, expectation, config, config.Warmup)
		cancel()
		_ = writeJSON(filepath.Join(outputDir, "http2-warmup-summary.json"), warmup)
		if err != nil {
			return executorResult{}, fmt.Errorf("HTTP/2 warmup failed: %w", err)
		}
	}

	measuredContext, cancel := context.WithTimeout(context.Background(), config.ExecutionTimeout)
	measured, err := runLoadPhase(measuredContext, "measured", targetBaseURL, expectation, config, config.Duration)
	cancel()
	if writeErr := writeJSON(filepath.Join(outputDir, "http2-load-summary.json"), measured); writeErr != nil {
		return executorResult{}, writeErr
	}
	if err != nil {
		return executorResult{}, fmt.Errorf("HTTP/2 measured phase failed: %w", err)
	}
	if measured.FailedRequests != 0 || measured.TimeoutRequests != 0 || measured.SuccessfulRequests == 0 {
		return executorResult{}, fmt.Errorf("HTTP/2 measured phase rejected: successful=%d failed=%d timedOut=%d", measured.SuccessfulRequests, measured.FailedRequests, measured.TimeoutRequests)
	}
	if err := validateMeasuredTopology(measured, config); err != nil {
		return executorResult{}, err
	}

	executableHash, err := currentExecutableSHA256()
	if err != nil {
		return executorResult{}, err
	}
	metrics := normalizePhaseMetrics(measured)
	requestedShape := normalizedLoadShape{
		Connections: config.Connections, Concurrency: config.Concurrency, StreamsPerConnection: config.StreamsPerConnection,
		DurationSeconds: config.Duration.Seconds(), WarmupSeconds: config.Warmup.Seconds(), RequestTimeoutSeconds: config.RequestTimeout.Seconds(), Repetition: config.Repetition,
		OperationDistribution: config.OperationDistribution,
	}
	effectiveShape := normalizedLoadShape{
		Connections: measured.ObservedDials, Concurrency: measured.MaximumActiveOperations,
		StreamsPerConnection: config.StreamsPerConnection,
		DurationSeconds:      config.Duration.Seconds(), WarmupSeconds: config.Warmup.Seconds(), RequestTimeoutSeconds: config.RequestTimeout.Seconds(), Repetition: config.Repetition,
		OperationDistribution:                          config.OperationDistribution,
		MaximumActiveStreamsByConnection:               append([]int(nil), measured.MaximumActiveOperationsByDial...),
		PeerAdvertisedMaxConcurrentStreamsByConnection: append([]uint32(nil), measured.PeerAdvertisedMaxConcurrentStreamsByDial...),
		Clamped: false, Redistributed: false,
	}
	if err := writeJSON(filepath.Join(outputDir, "http2-topology.json"), http2TopologyArtifact{
		SchemaVersion: "protocol-lab.http2-topology.v1", LoadProfileID: config.LoadProfileID,
		Requested: requestedShape, Effective: effectiveShape,
	}); err != nil {
		return executorResult{}, err
	}
	warnings := []string{"HTTP/2 h2c execution is local smoke or diagnostic evidence only; TLS/ALPN and ranking use are unsupported."}
	if config.LoadProfileID == "http2-comparison" {
		warnings = []string{"HTTP/2 h2c execution satisfies the comparison load contract, but same-host or single-implementation evidence is not publishable or ranking eligible."}
	}
	return executorResult{
		SchemaVersion: "protocol-lab.http-executor-result.v1",
		Executor:      componentIdentity{ID: executorID, Version: executorVersion},
		LoadGenerator: loadGeneratorIdentity{
			ID: loadGeneratorID, Version: loadGeneratorVersion, SHA256: executableHash,
			Command: "built-in h2c prior-knowledge engine", EngineModule: http2EngineModule,
			EngineModuleVersion: http2EngineModuleVersion,
		},
		Validation: validitySummary{Status: "passed"},
		ProtocolProof: normalizedProtocolProof{
			RequestedProtocol: requestedFamily, ObservedProtocol: requestedFamily,
			ExecutionVariant: requestedExecutionVariant, ExactProtocolMatched: true,
			FallbackDetected: false, ObservedDials: measured.ObservedDials,
			ConfiguredStreamsPerConnection:            config.StreamsPerConnection,
			MaximumActiveOperations:                   measured.MaximumActiveOperations,
			MaximumObservedActiveStreams:              maxInt(measured.MaximumObservedActiveStreamsByDial),
			MinimumPeerAdvertisedMaxConcurrentStreams: minPositiveUint32(measured.PeerAdvertisedMaxConcurrentStreamsByDial),
		},
		RequestedLoad: requestedShape,
		EffectiveLoad: effectiveShape,
		Metrics:       metrics,
		Warnings:      warnings,
	}, nil
}

func runLoadPhase(ctx context.Context, phase, targetBaseURL string, expectation scenarioExpectation, config loadConfig, duration time.Duration) (phaseSummary, error) {
	started := time.Now().UTC()
	recorder := &phaseRecorder{summary: phaseSummary{
		Phase: phase, StartedAtUTC: started.Format(time.RFC3339Nano),
		StatusCodeCounts: map[string]int64{}, Errors: map[string]int64{},
		MaximumActiveOperationsByDial:            make([]int, config.Connections),
		MaximumObservedActiveStreamsByDial:       make([]int, config.Connections),
		MaximumObservedPendingStreamsByDial:      make([]int, config.Connections),
		PeerAdvertisedMaxConcurrentStreamsByDial: make([]uint32, config.Connections),
	}, activeOperationsByDial: make([]int, config.Connections)}
	connections := make([]*h2CConnection, 0, config.Connections)
	for range config.Connections {
		connection, err := dialH2C(ctx, targetBaseURL, config.RequestTimeout)
		if err != nil {
			closeConnections(connections)
			recorder.summary.DurationSeconds = time.Since(started).Seconds()
			return recorder.summary, err
		}
		connections = append(connections, connection)
	}
	recorder.summary.ObservedDials = len(connections)
	defer closeConnections(connections)

	samplerContext, stopSampler := context.WithCancel(ctx)
	var sampler sync.WaitGroup
	sampler.Add(1)
	go func() {
		defer sampler.Done()
		sampleConnectionStates(samplerContext, connections, recorder)
	}()
	defer func() {
		stopSampler()
		sampler.Wait()
	}()

	deadline := time.Now().Add(duration)
	scheduler := newConnectionScheduler(len(connections), config.StreamsPerConnection)
	var workers sync.WaitGroup
	var initialOperationsReady sync.WaitGroup
	initialOperationsReady.Add(config.Concurrency)
	releaseInitialOperations := make(chan struct{})
	for range config.Concurrency {
		workers.Add(1)
		go func() {
			defer workers.Done()
			firstOperation := true
			for time.Now().Before(deadline) && ctx.Err() == nil {
				connectionIndex, acquireErr := scheduler.acquire(ctx)
				if acquireErr != nil {
					return
				}
				recorder.begin(connectionIndex)
				if firstOperation {
					initialOperationsReady.Done()
					<-releaseInitialOperations
					firstOperation = false
				}
				latency, firstByte, check := executeLoadOperation(ctx, connections[connectionIndex].client, targetBaseURL, expectation, config.RequestTimeout)
				recorder.finish(connectionIndex, latency, firstByte, check)
				scheduler.release(connectionIndex)
			}
		}()
	}
	initialOperationsReady.Wait()
	close(releaseInitialOperations)
	workers.Wait()
	stopSampler()
	sampler.Wait()
	recorder.captureConnectionStates(connections)
	recorder.mu.Lock()
	recorder.summary.DurationSeconds = time.Since(started).Seconds()
	summary := recorder.summary
	recorder.mu.Unlock()
	if ctx.Err() != nil {
		return summary, ctx.Err()
	}
	return summary, nil
}

func executeLoadOperation(ctx context.Context, client roundTripper, targetBaseURL string, expectation scenarioExpectation, timeout time.Duration) (time.Duration, time.Duration, checkResult) {
	var firstByteAt time.Time
	operationStart := time.Now()
	trace := &httptrace.ClientTrace{GotFirstResponseByte: func() { firstByteAt = time.Now() }}
	check := runCheck(httptrace.WithClientTrace(ctx, trace), client, targetBaseURL, expectation, timeout)
	finished := time.Now()
	firstByte := time.Duration(0)
	if !firstByteAt.IsZero() {
		firstByte = firstByteAt.Sub(operationStart)
	}
	return finished.Sub(operationStart), firstByte, check
}

func (recorder *phaseRecorder) begin(connectionIndex int) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.activeOperations++
	recorder.activeOperationsByDial[connectionIndex]++
	if recorder.activeOperations > recorder.summary.MaximumActiveOperations {
		recorder.summary.MaximumActiveOperations = recorder.activeOperations
	}
	if recorder.activeOperationsByDial[connectionIndex] > recorder.summary.MaximumActiveOperationsByDial[connectionIndex] {
		recorder.summary.MaximumActiveOperationsByDial[connectionIndex] = recorder.activeOperationsByDial[connectionIndex]
	}
}

func (recorder *phaseRecorder) finish(connectionIndex int, latency, firstByte time.Duration, check checkResult) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.activeOperations--
	recorder.activeOperationsByDial[connectionIndex]--
	recorder.summary.TotalRequests++
	recorder.summary.LatencySamplesMilliseconds = append(recorder.summary.LatencySamplesMilliseconds, durationMilliseconds(latency))
	if firstByte > 0 {
		recorder.summary.FirstByteSamplesMilliseconds = append(recorder.summary.FirstByteSamplesMilliseconds, durationMilliseconds(firstByte))
	}
	if check.ObservedStatus > 0 {
		recorder.summary.StatusCodeCounts[strconv.Itoa(check.ObservedStatus)]++
	}
	if check.TimedOut {
		recorder.summary.TimeoutRequests++
		recorder.summary.Errors[check.Error]++
		return
	}
	if !check.Passed {
		recorder.summary.FailedRequests++
		recorder.summary.Errors[check.Error]++
		return
	}
	recorder.summary.SuccessfulRequests++
	recorder.summary.BytesReceived += int64(check.ObservedPayloadLength)
}

func sampleConnectionStates(ctx context.Context, connections []*h2CConnection, recorder *phaseRecorder) {
	ticker := time.NewTicker(250 * time.Microsecond)
	defer ticker.Stop()
	for {
		recorder.captureConnectionStates(connections)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (recorder *phaseRecorder) captureConnectionStates(connections []*h2CConnection) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	for index, connection := range connections {
		state := connection.client.State()
		if state.StreamsActive > recorder.summary.MaximumObservedActiveStreamsByDial[index] {
			recorder.summary.MaximumObservedActiveStreamsByDial[index] = state.StreamsActive
		}
		if state.StreamsPending > recorder.summary.MaximumObservedPendingStreamsByDial[index] {
			recorder.summary.MaximumObservedPendingStreamsByDial[index] = state.StreamsPending
		}
		if state.MaxConcurrentStreams > recorder.summary.PeerAdvertisedMaxConcurrentStreamsByDial[index] {
			recorder.summary.PeerAdvertisedMaxConcurrentStreamsByDial[index] = state.MaxConcurrentStreams
		}
	}
}

func validateMeasuredTopology(summary phaseSummary, config loadConfig) error {
	if summary.ObservedDials != config.Connections ||
		summary.MaximumActiveOperations != config.Concurrency ||
		len(summary.MaximumActiveOperationsByDial) != config.Connections ||
		len(summary.PeerAdvertisedMaxConcurrentStreamsByDial) != config.Connections {
		return fmt.Errorf(
			"HTTP/2 topology proof rejected: requested=%d/%d/%d observedDials=%d maxActiveOperations=%d perDial=%v peerLimits=%v",
			config.Connections, config.Concurrency, config.StreamsPerConnection,
			summary.ObservedDials, summary.MaximumActiveOperations,
			summary.MaximumActiveOperationsByDial, summary.PeerAdvertisedMaxConcurrentStreamsByDial)
	}
	for index, activeOperations := range summary.MaximumActiveOperationsByDial {
		if activeOperations > config.StreamsPerConnection {
			return fmt.Errorf(
				"HTTP/2 topology proof rejected: connection %d used %d concurrent operations above configured streamsPerConnection=%d",
				index, activeOperations, config.StreamsPerConnection)
		}
		peerLimit := summary.PeerAdvertisedMaxConcurrentStreamsByDial[index]
		if peerLimit == 0 || peerLimit < uint32(config.StreamsPerConnection) {
			return fmt.Errorf(
				"HTTP/2 topology proof rejected: connection %d peer advertised maxConcurrentStreams=%d below requested=%d",
				index, peerLimit, config.StreamsPerConnection)
		}
	}
	return nil
}

func maxInt(values []int) int {
	maximum := 0
	for _, value := range values {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func minPositiveUint32(values []uint32) uint32 {
	var minimum uint32
	for _, value := range values {
		if value > 0 && (minimum == 0 || value < minimum) {
			minimum = value
		}
	}
	return minimum
}

func normalizePhaseMetrics(summary phaseSummary) normalizedHTTPMetrics {
	duration := summary.DurationSeconds
	if duration <= 0 {
		duration = 1
	}
	return normalizedHTTPMetrics{
		TotalRequests: summary.TotalRequests, SuccessfulRequests: summary.SuccessfulRequests,
		FailedRequests: summary.FailedRequests, TimeoutRequests: summary.TimeoutRequests,
		RequestsPerSecond: float64(summary.TotalRequests) / duration,
		BytesSent:         0, BytesReceived: summary.BytesReceived,
		ThroughputBytesPerSecond: float64(summary.BytesReceived) / duration,
		LatencyMeanMS:            mean(summary.LatencySamplesMilliseconds),
		LatencyP50MS:             percentile(summary.LatencySamplesMilliseconds, 0.50),
		LatencyP75MS:             percentile(summary.LatencySamplesMilliseconds, 0.75),
		LatencyP90MS:             percentile(summary.LatencySamplesMilliseconds, 0.90),
		LatencyP95MS:             percentile(summary.LatencySamplesMilliseconds, 0.95),
		LatencyP99MS:             percentile(summary.LatencySamplesMilliseconds, 0.99),
		TimeToFirstByteMeanMS:    mean(summary.FirstByteSamplesMilliseconds),
		StatusCodeCounts:         summary.StatusCodeCounts,
	}
}

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

func percentile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	copyOfValues := append([]float64(nil), values...)
	sort.Float64s(copyOfValues)
	index := int(math.Ceil(quantile*float64(len(copyOfValues)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(copyOfValues) {
		index = len(copyOfValues) - 1
	}
	return copyOfValues[index]
}

func closeConnections(connections []*h2CConnection) {
	for _, connection := range connections {
		connection.Close()
	}
}

func currentExecutableSHA256() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
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

func durationMilliseconds(value time.Duration) float64 {
	return float64(value) / float64(time.Millisecond)
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
