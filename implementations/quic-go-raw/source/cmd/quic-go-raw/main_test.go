package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
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
		"quic.transport.stream-churn",
		"quic.transport.duplex-streams",
		"quic.transport.duplex-streams-peer-matrix",
		"quic.transport.handshake-cold",
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

func TestServerAcceptsColdHandshakeWithoutOpeningStream(t *testing.T) {
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
	if err := conn.CloseWithError(0, ""); err != nil {
		t.Fatalf("CloseWithError failed: %v", err)
	}
}

func TestServerHandlesConnectionChurnWithFreshConnections(t *testing.T) {
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

	for i := 0; i < 3; i++ {
		conn, err := quic.DialAddr(
			context.Background(),
			listener.Addr().String(),
			&tls.Config{InsecureSkipVerify: true, NextProtos: []string{defaultALPN}},
			&quic.Config{MaxIdleTimeout: 10 * time.Second})
		if err != nil {
			t.Fatalf("DialAddr iteration %d failed: %v", i, err)
		}

		payload := bytes.Repeat([]byte{byte(0x40 + i)}, 1024)
		stream, err := conn.OpenStreamSync(context.Background())
		if err != nil {
			t.Fatalf("OpenStreamSync iteration %d failed: %v", i, err)
		}
		if _, err := stream.Write(payload); err != nil {
			t.Fatalf("Write iteration %d failed: %v", i, err)
		}
		if err := stream.Close(); err != nil {
			t.Fatalf("Close write side iteration %d failed: %v", i, err)
		}

		echo, err := io.ReadAll(stream)
		if err != nil {
			t.Fatalf("ReadAll iteration %d failed: %v", i, err)
		}
		if !bytes.Equal(echo, payload) {
			t.Fatalf("echo payload mismatch on iteration %d: got %d bytes", i, len(echo))
		}
		if err := conn.CloseWithError(0, ""); err != nil {
			t.Fatalf("CloseWithError iteration %d failed: %v", i, err)
		}
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

func TestPackageManifestsStayDualRidAndCanonical(t *testing.T) {
	packageManifestPath := filepath.Join("..", "..", "..", "protocol-lab-package.json")
	internalManifestPath := filepath.Join("..", "..", "..", "protocol-lab.internal.json")
	implementationManifestPath := filepath.Join("..", "..", "..", "implementations", "quic-go-raw.yaml")

	packageManifestBytes, err := os.ReadFile(packageManifestPath)
	if err != nil {
		t.Fatalf("read package manifest: %v", err)
	}

	var packageManifest struct {
		PackageVersion          string `json:"packageVersion"`
		ProvidedImplementations []struct {
			Scenarios []string `json:"scenarios"`
		} `json:"providedImplementations"`
	}
	if err := json.Unmarshal(packageManifestBytes, &packageManifest); err != nil {
		t.Fatalf("unmarshal package manifest: %v", err)
	}
	if packageManifest.PackageVersion != "0.1.8" {
		t.Fatalf("packageVersion = %q, want 0.1.8", packageManifest.PackageVersion)
	}
	if len(packageManifest.ProvidedImplementations) != 1 {
		t.Fatalf("providedImplementations length = %d, want 1", len(packageManifest.ProvidedImplementations))
	}
	wantPackageScenarios := []string{
		"quic.transport.stream-throughput.1mb",
		"quic.transport.multiplex.100x64kb",
		"quic.transport.stream-churn",
		"quic.transport.duplex-streams",
		"quic.transport.duplex-streams-peer-matrix",
		"quic.transport.handshake-cold",
	}
	if !reflect.DeepEqual(packageManifest.ProvidedImplementations[0].Scenarios, wantPackageScenarios) {
		t.Fatalf("providedImplementations[0].scenarios = %v, want %v", packageManifest.ProvidedImplementations[0].Scenarios, wantPackageScenarios)
	}

	internalManifestBytes, err := os.ReadFile(internalManifestPath)
	if err != nil {
		t.Fatalf("read internal manifest: %v", err)
	}

	var internalManifest struct {
		Environments []struct {
			OS         string `json:"os"`
			Arch       string `json:"arch"`
			Entrypoint struct {
				Kind             string   `json:"kind"`
				Path             string   `json:"path"`
				Arguments        []string `json:"arguments"`
				WorkingDirectory string   `json:"workingDirectory"`
			} `json:"entrypoint"`
		} `json:"environments"`
		Commands struct {
			BuildTemplate  string `json:"buildTemplate"`
			ServerTemplate string `json:"serverTemplate"`
			PlanOnly       string `json:"planOnly"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(internalManifestBytes, &internalManifest); err != nil {
		t.Fatalf("unmarshal internal manifest: %v", err)
	}

	if got, want := len(internalManifest.Environments), 2; got != want {
		t.Fatalf("environments length = %d, want %d", got, want)
	}

	wantEnvironments := map[string]string{
		"linux/x64":   "bin/linux-x64/quic-go-raw",
		"windows/x64": "bin/windows-x64/quic-go-raw.exe",
	}
	for _, environment := range internalManifest.Environments {
		key := environment.OS + "/" + environment.Arch
		wantPath, ok := wantEnvironments[key]
		if !ok {
			t.Fatalf("unexpected environment %s", key)
		}
		if environment.Entrypoint.Kind != "process" {
			t.Fatalf("environment %s kind = %q, want process", key, environment.Entrypoint.Kind)
		}
		if environment.Entrypoint.Path != wantPath {
			t.Fatalf("environment %s path = %q, want %q", key, environment.Entrypoint.Path, wantPath)
		}
		if len(environment.Entrypoint.Arguments) != 0 {
			t.Fatalf("environment %s arguments = %v, want none", key, environment.Entrypoint.Arguments)
		}
		if environment.Entrypoint.WorkingDirectory != "." {
			t.Fatalf("environment %s workingDirectory = %q, want .", key, environment.Entrypoint.WorkingDirectory)
		}
		delete(wantEnvironments, key)
	}
	if len(wantEnvironments) != 0 {
		t.Fatalf("missing environments: %v", wantEnvironments)
	}

	if internalManifest.Commands.BuildTemplate != "pwsh ../../scripts/package/Build-QuicGoRawPackage.ps1" {
		t.Fatalf("buildTemplate = %q, want repo package builder", internalManifest.Commands.BuildTemplate)
	}
	if internalManifest.Commands.ServerTemplate != "pwsh ./run.ps1" {
		t.Fatalf("serverTemplate = %q, want pwsh ./run.ps1", internalManifest.Commands.ServerTemplate)
	}
	if internalManifest.Commands.PlanOnly != "pwsh ./run.ps1 -PlanOnly" {
		t.Fatalf("planOnly = %q, want pwsh ./run.ps1 -PlanOnly", internalManifest.Commands.PlanOnly)
	}

	implementationManifestBytes, err := os.ReadFile(implementationManifestPath)
	if err != nil {
		t.Fatalf("read implementation manifest: %v", err)
	}
	if !bytes.Contains(implementationManifestBytes, []byte("executable: bin/linux-x64/quic-go-raw")) {
		t.Fatal("implementation YAML does not retain the canonical Linux executable")
	}
	if bytes.Contains(implementationManifestBytes, []byte("bin/windows-x64/quic-go-raw.exe")) {
		t.Fatal("implementation YAML should not advertise a Windows executable")
	}
	if !bytes.Contains(implementationManifestBytes, []byte("quic.transport.stream-churn")) {
		t.Fatal("implementation YAML does not advertise quic.transport.stream-churn")
	}
	if !bytes.Contains(implementationManifestBytes, []byte("quic.transport.handshake-cold")) {
		t.Fatal("implementation YAML does not advertise quic.transport.handshake-cold")
	}
	if !bytes.Contains(implementationManifestBytes, []byte("quic.transport.resumption-rejected")) {
		t.Fatal("implementation YAML does not mark quic.transport.resumption-rejected unsupported")
	}
	if !bytes.Contains(implementationManifestBytes, []byte("quic.transport.resumed-handshake")) {
		t.Fatal("implementation YAML does not mark quic.transport.resumed-handshake unsupported")
	}
	if !bytes.Contains(implementationManifestBytes, []byte("quic.transport.zero-rtt-accepted")) {
		t.Fatal("implementation YAML does not mark quic.transport.zero-rtt-accepted unsupported")
	}
	if !bytes.Contains(implementationManifestBytes, []byte("quic.transport.zero-rtt-rejected")) {
		t.Fatal("implementation YAML does not mark quic.transport.zero-rtt-rejected unsupported")
	}
	if !bytes.Contains(implementationManifestBytes, []byte("  - quicHandshake")) {
		t.Fatal("implementation YAML does not advertise quicHandshake capability")
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
