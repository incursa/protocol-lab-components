package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
)

func TestValidateOptionsRejectsUnsupportedBehaviorOpenPatternCombos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options options
		wantErr string
	}{
		{
			name: "stream-throughput accepts sequential",
			options: options{
				behavior:             "stream-throughput",
				openPattern:          "sequential",
				streamsPerConnection: 1,
			},
		},
		{
			name: "latency echo accepts sequential",
			options: options{
				behavior:             "latency-echo",
				openPattern:          "sequential",
				streamsPerConnection: 1,
			},
		},
		{
			name: "large payload accepts sequential",
			options: options{
				behavior:             "large-payload",
				openPattern:          "sequential",
				streamsPerConnection: 1,
			},
		},
		{
			name: "sustained stream accepts sequential",
			options: options{
				behavior:             "sustained-stream-256x64kb",
				openPattern:          "sequential",
				streamsPerConnection: 1,
			},
		},
		{
			name: "sustained download accepts sequential",
			options: options{
				behavior:             "sustained-download-256x64kb",
				payloadDirection:     "server-to-client",
				openPattern:          "sequential",
				streamsPerConnection: 1,
			},
		},
		{
			name: "stream limit pressure accepts concurrent",
			options: options{
				behavior:             "stream-limit-pressure",
				openPattern:          "concurrent",
				streamsPerConnection: 100,
			},
		},
		{
			name: "flow control slow reader accepts concurrent",
			options: options{
				behavior:             "flow-control-slow-reader-100ms",
				openPattern:          "concurrent",
				streamsPerConnection: 16,
			},
		},
		{
			name: "mixed multiplex accepts exact round robin sequence",
			options: options{
				behavior:             "multiplex-streams-mixed-size",
				payloadDirection:     "bidirectional",
				payloadSizesBytes:    []int{1024, 16384, 65536, 1048576},
				openPattern:          "concurrent",
				streamsPerConnection: 16,
			},
		},
		{
			name: "mixed multiplex rejects missing size sequence",
			options: options{
				behavior:             "multiplex-streams-mixed-size",
				payloadDirection:     "bidirectional",
				openPattern:          "concurrent",
				streamsPerConnection: 16,
			},
			wantErr: "requires payload-sizes-bytes",
		},
		{
			name: "mixed multiplex rejects incomplete round robin",
			options: options{
				behavior:             "multiplex-streams-mixed-size",
				payloadDirection:     "bidirectional",
				payloadSizesBytes:    []int{1024, 16384, 65536},
				openPattern:          "concurrent",
				streamsPerConnection: 16,
			},
			wantErr: "must be divisible",
		},
		{
			name: "multiplex-streams rejects sequential",
			options: options{
				behavior:             "multiplex-streams",
				openPattern:          "sequential",
				streamsPerConnection: 1,
			},
			wantErr: "requires open-pattern",
		},
		{
			name: "handshake-cold accepts zero-stream sequential",
			options: options{
				behavior:             "handshake-cold",
				openPattern:          "sequential",
				streamsPerConnection: 0,
			},
		},
		{
			name: "handshake-cold rejects concurrent",
			options: options{
				behavior:             "handshake-cold",
				openPattern:          "concurrent",
				streamsPerConnection: 0,
			},
			wantErr: "requires open-pattern",
		},
		{
			name: "connection-churn accepts churn",
			options: options{
				behavior:             "connection-churn",
				openPattern:          "churn",
				streamsPerConnection: 8,
			},
		},
		{
			name: "connection-churn rejects concurrent",
			options: options{
				behavior:             "connection-churn",
				openPattern:          "concurrent",
				streamsPerConnection: 8,
			},
			wantErr: "requires open-pattern",
		},
		{
			name: "stream-churn accepts churn",
			options: options{
				behavior:             "stream-churn",
				openPattern:          "churn",
				streamsPerConnection: 1000,
			},
		},
		{
			name: "stream-churn rejects concurrent",
			options: options{
				behavior:             "stream-churn",
				openPattern:          "concurrent",
				streamsPerConnection: 1000,
			},
			wantErr: "requires open-pattern",
		},
		{
			name: "unknown behavior rejects",
			options: options{
				behavior:             "made-up",
				openPattern:          "sequential",
				streamsPerConnection: 1,
			},
			wantErr: "unsupported behavior",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateOptions(tt.options)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateOptions returned error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateOptions returned nil error, want substring %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateOptions error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParsePayloadSizesAndRoundRobinSelection(t *testing.T) {
	t.Parallel()

	sizes, err := parsePayloadSizes("1024,16384,65536,1048576")
	if err != nil {
		t.Fatalf("parsePayloadSizes returned error: %v", err)
	}
	opts := options{payloadSizesBytes: sizes}
	payloads := buildPayloads(opts)
	wantSizes := []int{1024, 16384, 65536, 1048576, 1024, 16384, 65536, 1048576}
	for streamIndex, wantSize := range wantSizes {
		payload := payloadForStream(payloads, streamIndex)
		if len(payload) != wantSize {
			t.Fatalf("stream %d payload size = %d, want %d", streamIndex, len(payload), wantSize)
		}
		for byteIndex, value := range payload {
			if value != byte(byteIndex%251) {
				t.Fatalf("stream %d payload byte %d = %d, want %d", streamIndex, byteIndex, value, byte(byteIndex%251))
			}
		}
	}
}

func TestParsePayloadSizesRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"1024,,65536", "1024,0", "1024,invalid"} {
		if _, err := parsePayloadSizes(value); err == nil {
			t.Fatalf("parsePayloadSizes(%q) returned nil error", value)
		}
	}
}

func TestResponseReadDelayIsExactAndBehaviorScoped(t *testing.T) {
	t.Parallel()

	if got := responseReadDelay("flow-control-slow-reader-100ms"); got != 100*time.Millisecond {
		t.Fatalf("responseReadDelay(slow reader) = %s, want 100ms", got)
	}
	if got := responseReadDelay("multiplex-streams"); got != 0 {
		t.Fatalf("responseReadDelay(multiplex) = %s, want 0", got)
	}
}

func TestWritePayloadInChunksPreservesPayloadAndBoundaries(t *testing.T) {
	t.Parallel()

	payload := make([]byte, 256*64*1024)
	for index := range payload {
		payload[index] = byte(index % 251)
	}
	writer := &recordingWriter{}

	written, err := writePayloadInChunks(writer, payload, 64*1024)
	if err != nil {
		t.Fatalf("writePayloadInChunks returned error: %v", err)
	}
	if written != len(payload) {
		t.Fatalf("written = %d, want %d", written, len(payload))
	}
	if len(writer.writeSizes) != 256 {
		t.Fatalf("write calls = %d, want 256", len(writer.writeSizes))
	}
	for index, size := range writer.writeSizes {
		if size != 64*1024 {
			t.Fatalf("write %d size = %d, want 65536", index, size)
		}
	}
	if !bytes.Equal(writer.bytes, payload) {
		t.Fatal("written payload did not preserve the source bytes")
	}
}

type recordingWriter struct {
	bytes      []byte
	writeSizes []int
}

func (writer *recordingWriter) Write(payload []byte) (int, error) {
	writer.writeSizes = append(writer.writeSizes, len(payload))
	writer.bytes = append(writer.bytes, payload...)
	return len(payload), nil
}

func TestBuildMetricsIncludesConnectTimeMean(t *testing.T) {
	t.Parallel()

	stats := runStats{
		totalRequests:      4,
		successfulRequests: 3,
		failedRequests:     1,
		bytesSent:          12,
		bytesReceived:      9,
		latencies:          []float64{30, 10, 20},
		connectLatencies:   []float64{9, 21, 30},
	}

	metrics := buildMetrics(stats, 3*time.Second)

	if metrics.RequestsPerSecond != 1 {
		t.Fatalf("requestsPerSecond = %f, want 1", metrics.RequestsPerSecond)
	}
	if metrics.ThroughputBytesPerSecond != 4 {
		t.Fatalf("throughputBytesPerSecond = %f, want 4", metrics.ThroughputBytesPerSecond)
	}
	if metrics.ConnectTimeMeanMs != 20 {
		t.Fatalf("connectTimeMeanMs = %f, want 20", metrics.ConnectTimeMeanMs)
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

func TestOutputDocumentEchoesPackagedExecutorIdentityAndRequestedLoad(t *testing.T) {
	t.Setenv("PLAB_EXECUTOR_ID", "quic-go-raw-load")
	t.Setenv("PLAB_EXECUTOR_VERSION", "package-v1")
	t.Setenv("PLAB_LOAD_GENERATOR_ID", "quic-go-raw-load")
	t.Setenv("PLAB_LOAD_GENERATOR_VERSION", "package-v1")
	t.Setenv("PLAB_CONCURRENCY", "8")

	opts := options{
		behavior:             "multiplex-streams",
		connections:          2,
		streamsPerConnection: 4,
		duration:             5 * time.Second,
		warmup:               time.Second,
		target:               "quic://127.0.0.1:4433/",
	}
	document := newOutputDocument(opts, metricsOutput{SuccessfulRequests: 1})

	if document.SchemaVersion != outputSchemaVersion {
		t.Fatalf("schemaVersion = %q, want %q", document.SchemaVersion, outputSchemaVersion)
	}
	if document.Executor.ID != "quic-go-raw-load" || document.Executor.Version != "package-v1" {
		t.Fatalf("executor = %#v, want package identity", document.Executor)
	}
	if document.LoadGenerator.ID != "quic-go-raw-load" || document.LoadGenerator.Version != "package-v1" {
		t.Fatalf("loadGenerator = %#v, want package identity", document.LoadGenerator)
	}
	if document.RequestedLoad.Connections != 2 || document.RequestedLoad.Concurrency != 8 ||
		document.RequestedLoad.StreamsPerConnection != 4 || document.RequestedLoad.DurationSeconds != 5 ||
		document.RequestedLoad.WarmupSeconds != 1 {
		t.Fatalf("requestedLoad = %#v, want injected load shape", document.RequestedLoad)
	}
	if document.EffectiveLoad != document.RequestedLoad {
		t.Fatalf("effectiveLoad = %#v, requestedLoad = %#v", document.EffectiveLoad, document.RequestedLoad)
	}
}

func TestRunLoadDispatchesRawQuicBehaviors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		opts              options
		echo              bool
		wantConnect       bool
		wantBytesSent     bool
		wantBytesReceived bool
	}{
		{
			name: "stream-throughput sequential",
			opts: options{
				sni:                  "localhost",
				alpn:                 "plab-raw-quic",
				behavior:             "stream-throughput",
				streamType:           "bidirectional",
				payloadSizeBytes:     32,
				payloadDirection:     "client-to-server",
				openPattern:          "sequential",
				connections:          1,
				streamsPerConnection: 1,
				duration:             25 * time.Millisecond,
				target:               "",
			},
			echo:          false,
			wantConnect:   false,
			wantBytesSent: true,
		},
		{
			name: "stream-download sequential",
			opts: options{
				sni:                  "localhost",
				alpn:                 "plab-raw-quic",
				behavior:             "stream-throughput",
				streamType:           "bidirectional",
				payloadSizeBytes:     32,
				payloadDirection:     "server-to-client",
				openPattern:          "sequential",
				connections:          1,
				streamsPerConnection: 1,
				duration:             25 * time.Millisecond,
				target:               "",
			},
			wantBytesReceived: true,
		},
		{
			name: "sustained-download sequential",
			opts: options{
				sni:                  "localhost",
				alpn:                 "plab-raw-quic",
				behavior:             "sustained-download-256x64kb",
				streamType:           "bidirectional",
				payloadSizeBytes:     32,
				payloadDirection:     "server-to-client",
				openPattern:          "sequential",
				connections:          1,
				streamsPerConnection: 1,
				duration:             25 * time.Millisecond,
				target:               "",
			},
			wantBytesReceived: true,
		},
		{
			name: "multiplex-streams concurrent",
			opts: options{
				sni:                  "localhost",
				alpn:                 "plab-raw-quic",
				behavior:             "multiplex-streams",
				streamType:           "bidirectional",
				payloadSizeBytes:     32,
				payloadDirection:     "bidirectional",
				openPattern:          "concurrent",
				connections:          1,
				streamsPerConnection: 4,
				duration:             25 * time.Millisecond,
				target:               "",
			},
			echo:              true,
			wantConnect:       false,
			wantBytesSent:     true,
			wantBytesReceived: true,
		},
		{
			name: "flow control slow reader concurrent",
			opts: options{
				sni:                  "localhost",
				alpn:                 "plab-raw-quic",
				behavior:             "flow-control-slow-reader-100ms",
				streamType:           "bidirectional",
				payloadSizeBytes:     32,
				payloadDirection:     "bidirectional",
				openPattern:          "concurrent",
				connections:          1,
				streamsPerConnection: 4,
				duration:             0,
				target:               "",
			},
			echo:              true,
			wantConnect:       false,
			wantBytesSent:     true,
			wantBytesReceived: true,
		},
		{
			name: "handshake-cold",
			opts: options{
				sni:                  "localhost",
				alpn:                 "plab-raw-quic",
				behavior:             "handshake-cold",
				streamType:           "none",
				payloadSizeBytes:     0,
				payloadDirection:     "none",
				openPattern:          "sequential",
				connections:          1,
				streamsPerConnection: 0,
				duration:             25 * time.Millisecond,
				target:               "",
			},
			echo:        false,
			wantConnect: true,
		},
		{
			name: "connection-churn",
			opts: options{
				sni:                  "localhost",
				alpn:                 "plab-raw-quic",
				behavior:             "connection-churn",
				streamType:           "bidirectional",
				payloadSizeBytes:     16,
				payloadDirection:     "bidirectional",
				openPattern:          "churn",
				connections:          1,
				streamsPerConnection: 2,
				duration:             25 * time.Millisecond,
				target:               "",
			},
			echo:              true,
			wantConnect:       true,
			wantBytesSent:     true,
			wantBytesReceived: true,
		},
		{
			name: "stream-churn",
			opts: options{
				sni:                  "localhost",
				alpn:                 "plab-raw-quic",
				behavior:             "stream-churn",
				streamType:           "bidirectional",
				payloadSizeBytes:     16,
				payloadDirection:     "bidirectional",
				openPattern:          "churn",
				connections:          1,
				streamsPerConnection: 8,
				duration:             25 * time.Millisecond,
				target:               "",
			},
			echo:              true,
			wantConnect:       false,
			wantBytesSent:     true,
			wantBytesReceived: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			serverTarget := startRawQUICTestServer(t, tt.echo)
			tt.opts.target = serverTarget

			if err := validateOptions(tt.opts); err != nil {
				t.Fatalf("validateOptions returned error: %v", err)
			}

			stats, err := runLoad(context.Background(), tt.opts, tt.opts.duration, false)
			if err != nil {
				t.Fatalf("runLoad returned error: %v", err)
			}
			if stats.successfulRequests == 0 {
				t.Fatal("successfulRequests = 0, want > 0")
			}
			if stats.failedRequests != 0 {
				t.Fatalf("failedRequests = %d, want 0", stats.failedRequests)
			}

			metrics := buildMetrics(stats, tt.opts.duration)
			if tt.wantConnect {
				if metrics.ConnectTimeMeanMs <= 0 {
					t.Fatalf("connectTimeMeanMs = %f, want > 0", metrics.ConnectTimeMeanMs)
				}
			} else if metrics.ConnectTimeMeanMs != 0 {
				t.Fatalf("connectTimeMeanMs = %f, want 0", metrics.ConnectTimeMeanMs)
			}

			if (metrics.BytesSent > 0) != tt.wantBytesSent {
				t.Fatalf("bytesSent = %d, want positive = %t", metrics.BytesSent, tt.wantBytesSent)
			}
			if (metrics.BytesReceived > 0) != tt.wantBytesReceived {
				t.Fatalf("bytesReceived = %d, want positive = %t", metrics.BytesReceived, tt.wantBytesReceived)
			}

			if tt.opts.behavior == "flow-control-slow-reader-100ms" && metrics.LatencyMinMs < 100 {
				t.Fatalf("latencyMinMs = %f, want >= 100ms", metrics.LatencyMinMs)
			}
		})
	}
}

func TestRunLoadDuplexStreamsWithTightWindowsDoesNotDeadlock(t *testing.T) {
	serverConfig := tightQUICConfig()
	clientConfig := tightQUICConfig()
	serverTarget := startRawQUICTestServerWithConfig(t, true, serverConfig, true)

	opts := options{
		sni:                  "localhost",
		alpn:                 "plab-raw-quic",
		behavior:             "duplex-streams",
		streamType:           "bidirectional",
		payloadSizeBytes:     64 * 1024,
		payloadDirection:     "bidirectional",
		openPattern:          "concurrent",
		connections:          1,
		streamsPerConnection: 1,
		duration:             100 * time.Millisecond,
		target:               serverTarget,
	}

	if err := validateOptions(opts); err != nil {
		t.Fatalf("validateOptions returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := runLoadWithQUICConfig(ctx, opts, 0, false, clientConfig)
	if err != nil {
		t.Fatalf("runLoadWithQUICConfig returned error: %v", err)
	}
	if stats.successfulRequests != 1 {
		t.Fatalf("successfulRequests = %d, want 1", stats.successfulRequests)
	}
	if stats.failedRequests != 0 {
		t.Fatalf("failedRequests = %d, want 0", stats.failedRequests)
	}

	metrics := buildMetrics(stats, opts.duration)
	if metrics.BytesSent == 0 || metrics.BytesReceived == 0 {
		t.Fatalf("bytesSent=%d bytesReceived=%d, want both > 0", metrics.BytesSent, metrics.BytesReceived)
	}
}

func startRawQUICTestServer(t *testing.T, echo bool) string {
	return startRawQUICTestServerWithConfig(t, echo, nil, false)
}

func startRawQUICTestServerWithConfig(t *testing.T, echo bool, serverConfig *quic.Config, chunkedEcho bool) string {
	t.Helper()

	tlsConf := testServerTLSConfig(t)
	if serverConfig == nil {
		serverConfig = defaultQUICConfig()
	}
	serverConfig = serverConfig.Clone()
	serverConfig.MaxIdleTimeout = time.Second
	serverConfig.KeepAlivePeriod = 250 * time.Millisecond

	listener, err := quic.ListenAddr("127.0.0.1:0", tlsConf, serverConfig)
	if err != nil {
		t.Fatalf("quic.ListenAddr returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = listener.Close()
	})

	go func() {
		for {
			conn, err := listener.Accept(ctx)
			if err != nil {
				return
			}
			if chunkedEcho {
				go serveChunkedEchoRawQUICTestConn(conn, echo)
				continue
			}
			go serveRawQUICTestConn(conn, echo)
		}
	}()

	return "quic://" + listener.Addr().String() + "/"
}

func serveRawQUICTestConn(conn *quic.Conn, echo bool) {
	defer conn.CloseWithError(0, "")

	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go func(stream *quic.Stream) {
			defer stream.Close()

			data, err := io.ReadAll(stream)
			if err != nil {
				return
			}
			if payloadLength, ok := parseDownloadRequest(data, maximumDownloadPayloadLength); ok {
				payload := make([]byte, payloadLength)
				for index := range payload {
					payload[index] = byte(index % 251)
				}
				_, _ = stream.Write(payload)
			} else if echo && len(data) > 0 {
				_, _ = stream.Write(data)
			}
			_ = stream.Close()
		}(stream)
	}
}

func serveChunkedEchoRawQUICTestConn(conn *quic.Conn, echo bool) {
	defer conn.CloseWithError(0, "")

	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go func(stream *quic.Stream) {
			defer stream.Close()

			buffer := make([]byte, 1024)
			for {
				read, readErr := stream.Read(buffer)
				if read > 0 && echo {
					_, _ = stream.Write(buffer[:read])
				}
				if readErr == nil {
					continue
				}
				if !errors.Is(readErr, io.EOF) {
					return
				}
				return
			}
		}(stream)
	}
}

func testServerTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey returned error: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "quic-go-raw-load-test",
			Organization: []string{"Incursa"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("x509.CreateCertificate returned error: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("tls.X509KeyPair returned error: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"plab-raw-quic"},
		MinVersion:   tls.VersionTLS13,
	}
}

func tightQUICConfig() *quic.Config {
	return &quic.Config{
		InitialStreamReceiveWindow:     1024,
		MaxStreamReceiveWindow:         1024,
		InitialConnectionReceiveWindow: 1024,
		MaxConnectionReceiveWindow:     1024,
		MaxIdleTimeout:                 time.Second,
		KeepAlivePeriod:                250 * time.Millisecond,
	}
}
