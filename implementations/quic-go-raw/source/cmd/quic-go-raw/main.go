package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
)

const (
	implementationID   = "quic-go-raw"
	packageID          = "org.protocol-lab.components.implementation.quic-go-raw"
	defaultALPN        = "plab-raw-quic"
	defaultPort        = "5447"
	defaultEchoMaxSize = 64 * 1024
	maxReadBytes       = 64 * 1024 * 1024
)

var quicGoVersion = "v0.60.0"
var supportedScenarios = []string{
	"quic.transport.stream-throughput.1mb",
	"quic.transport.multiplex.100x64kb",
	"quic.transport.stream-churn",
	"quic.transport.duplex-streams",
	"quic.transport.cold-handshake",
}

type options struct {
	listen        string
	advertiseHost string
	alpn          string
	echoMaxBytes  int64
}

type metadata struct {
	Status             string   `json:"status"`
	ImplementationID   string   `json:"implementationId"`
	PackageID          string   `json:"packageId"`
	Protocol           string   `json:"protocol"`
	ALPN               string   `json:"alpn"`
	Listen             string   `json:"listen"`
	AdvertiseHost      string   `json:"advertiseHost,omitempty"`
	QuicGoVersion      string   `json:"quicGoVersion"`
	ProcessID          int      `json:"processId"`
	SupportedScenarios []string `json:"supportedScenarios"`
}

type streamReadWriteCloser interface {
	io.Reader
	io.Writer
	Close() error
}

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runServer(ctx, opts); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

func parseOptions(args []string) (options, error) {
	opts := options{
		listen:        defaultListenAddress(),
		advertiseHost: strings.TrimSpace(os.Getenv("PROTOCOL_LAB_TARGET_ADVERTISE_HOST")),
		alpn:          defaultALPN,
		echoMaxBytes:  defaultEchoMaxSize,
	}

	fs := flag.NewFlagSet("quic-go-raw", flag.ContinueOnError)
	fs.StringVar(&opts.listen, "listen", opts.listen, "UDP listen address")
	fs.StringVar(&opts.advertiseHost, "advertise-host", opts.advertiseHost, "host advertised by external orchestration")
	fs.StringVar(&opts.alpn, "alpn", opts.alpn, "QUIC ALPN value")
	fs.Int64Var(&opts.echoMaxBytes, "echo-max-bytes", opts.echoMaxBytes, "maximum stream payload size to echo")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if fs.NArg() != 0 {
		return opts, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	if opts.echoMaxBytes < 0 {
		return opts, errors.New("echo-max-bytes cannot be negative")
	}
	if opts.alpn == "" {
		return opts, errors.New("alpn cannot be empty")
	}
	if _, err := net.ResolveUDPAddr("udp", opts.listen); err != nil {
		return opts, fmt.Errorf("invalid listen address %q: %w", opts.listen, err)
	}

	return opts, nil
}

func defaultListenAddress() string {
	port := strings.TrimSpace(os.Getenv("PLAB_QUIC_PORT"))
	if port == "" {
		port = defaultPort
	}

	bind := strings.TrimSpace(os.Getenv("PROTOCOL_LAB_TARGET_BIND_ADDRESS"))
	if bind == "" {
		bind = "127.0.0.1"
	}

	if _, _, err := net.SplitHostPort(bind); err == nil {
		return bind
	}

	return net.JoinHostPort(bind, port)
}

func runServer(ctx context.Context, opts options) error {
	listener, err := quic.ListenAddr(opts.listen, mustTLSConfig(opts.alpn), &quic.Config{
		MaxIdleTimeout:                 30 * time.Second,
		KeepAlivePeriod:                5 * time.Second,
		MaxIncomingStreams:             1024,
		MaxIncomingUniStreams:          16,
		InitialStreamReceiveWindow:     16 * 1024 * 1024,
		MaxStreamReceiveWindow:         64 * 1024 * 1024,
		InitialConnectionReceiveWindow: 32 * 1024 * 1024,
		MaxConnectionReceiveWindow:     128 * 1024 * 1024,
	})
	if err != nil {
		return err
	}
	defer listener.Close()

	writeMetadata(os.Stdout, opts, listener.Addr().String())
	log.Printf("quic-go raw QUIC target listening on %s", listener.Addr())

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	return serveListener(ctx, listener, opts)
}

func serveListener(ctx context.Context, listener *quic.Listener, opts options) error {
	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		go handleConnection(ctx, conn, opts)
	}
}

func handleConnection(ctx context.Context, conn *quic.Conn, opts options) {
	defer conn.CloseWithError(0, "")
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return
		}
		go handleStream(stream, opts)
	}
}

func handleStream(stream streamReadWriteCloser, opts options) {
	defer stream.Close()

	payload, err := io.ReadAll(io.LimitReader(stream, maxReadBytes+1))
	if err != nil {
		log.Printf("read stream failed: %v", err)
		return
	}
	if len(payload) > maxReadBytes {
		log.Printf("stream payload exceeded limit: %d", len(payload))
		return
	}
	if int64(len(payload)) > opts.echoMaxBytes {
		return
	}
	if len(payload) == 0 {
		return
	}

	if _, err := stream.Write(payload); err != nil {
		log.Printf("write stream echo failed: %v", err)
	}
}

func writeMetadata(writer io.Writer, opts options, listen string) {
	value := metadata{
		Status:             "ready",
		ImplementationID:   implementationID,
		PackageID:          packageID,
		Protocol:           "quic",
		ALPN:               opts.alpn,
		Listen:             listen,
		AdvertiseHost:      opts.advertiseHost,
		QuicGoVersion:      quicGoVersion,
		ProcessID:          os.Getpid(),
		SupportedScenarios: supportedScenarios,
	}
	_ = json.NewEncoder(writer).Encode(value)
}

func mustTLSConfig(alpn string) *tls.Config {
	cert, err := generateCertificate()
	if err != nil {
		log.Fatalf("failed to generate TLS certificate: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{alpn},
		MinVersion:   tls.VersionTLS13,
	}
}

func generateCertificate() (tls.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "ProtocolLab quic-go raw QUIC local certificate",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "host.docker.internal"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return tls.X509KeyPair(certPEM, keyPEM)
}
