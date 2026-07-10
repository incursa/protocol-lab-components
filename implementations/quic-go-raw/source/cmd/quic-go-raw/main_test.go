package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
)

func TestParseOptionsUsesProtocolLabBindEnvironment(t *testing.T) {
	t.Setenv("PROTOCOL_LAB_TARGET_BIND_ADDRESS", "0.0.0.0")
	t.Setenv("PROTOCOL_LAB_TARGET_ADVERTISE_HOST", "10.50.0.11")
	t.Setenv("PLAB_QUIC_PORT", "5547")

	opts, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("parseOptions returned error: %v", err)
	}

	if opts.listen != "0.0.0.0:5547" {
		t.Fatalf("listen = %q, want 0.0.0.0:5547", opts.listen)
	}
	if opts.advertiseHost != "10.50.0.11" {
		t.Fatalf("advertiseHost = %q, want 10.50.0.11", opts.advertiseHost)
	}
	if opts.alpn != defaultALPN {
		t.Fatalf("alpn = %q, want %q", opts.alpn, defaultALPN)
	}
}

func TestWriteMetadataIncludesSupportedScenarios(t *testing.T) {
	var buf bytes.Buffer

	writeMetadata(&buf, options{alpn: defaultALPN}, "127.0.0.1:5447")

	var got metadata
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	want := []string{
		"quic.transport.stream-throughput.1mb",
		"quic.transport.multiplex.100x64kb",
		"quic.transport.duplex-streams",
	}
	if !reflect.DeepEqual(got.SupportedScenarios, want) {
		t.Fatalf("supportedScenarios = %v, want %v", got.SupportedScenarios, want)
	}
}

func TestServerEchoesDuplexPayload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := quic.ListenAddr("127.0.0.1:0", mustTLSConfig(defaultALPN), &quic.Config{
		MaxIncomingStreams: 128,
	})
	if err != nil {
		t.Fatalf("ListenAddr failed: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = serveListener(ctx, listener, options{alpn: defaultALPN, echoMaxBytes: defaultEchoMaxSize})
	}()

	conn, err := quic.DialAddr(
		context.Background(),
		listener.Addr().String(),
		&tls.Config{InsecureSkipVerify: true, NextProtos: []string{defaultALPN}},
		&quic.Config{MaxIdleTimeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("DialAddr failed: %v", err)
	}
	defer conn.CloseWithError(0, "")

	payload := bytes.Repeat([]byte{0x5a}, 64*1024)
	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		t.Fatalf("OpenStreamSync failed: %v", err)
	}
	if _, err := stream.Write(payload); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close write side failed: %v", err)
	}

	echo, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(echo, payload) {
		t.Fatalf("echo payload mismatch: got %d bytes", len(echo))
	}
}

func TestServerAcceptsLargeClientToServerPayloadWithoutEcho(t *testing.T) {
	stream := &fakeStream{reader: bytes.NewReader(bytes.Repeat([]byte{0x7}, 1024*1024))}
	handleStream(stream, options{echoMaxBytes: defaultEchoMaxSize})

	if stream.write.Len() != 0 {
		t.Fatalf("large client-to-server payload was echoed: %d bytes", stream.write.Len())
	}
	if !stream.closed {
		t.Fatal("stream was not closed")
	}
}

type fakeStream struct {
	reader *bytes.Reader
	write  bytes.Buffer
	closed bool
}

func (s *fakeStream) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *fakeStream) Write(p []byte) (int, error) {
	return s.write.Write(p)
}

func (s *fakeStream) Close() error {
	s.closed = true
	return nil
}

func (s *fakeStream) CancelRead(quic.StreamErrorCode) {}

func (s *fakeStream) CancelWrite(quic.StreamErrorCode) {}

func (s *fakeStream) Context() context.Context {
	return context.Background()
}

func (s *fakeStream) SetDeadline(time.Time) error {
	return nil
}

func (s *fakeStream) SetReadDeadline(time.Time) error {
	return nil
}

func (s *fakeStream) SetWriteDeadline(time.Time) error {
	return nil
}

func (s *fakeStream) StreamID() quic.StreamID {
	return 0
}

var _ interface {
	io.Reader
	io.Writer
	Close() error
} = (*fakeStream)(nil)
