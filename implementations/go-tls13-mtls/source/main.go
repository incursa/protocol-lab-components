package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	implementationID      = "go-tls13-mtls"
	implementationVersion = "0.1.0"
	scenarioID            = "tls.handshake.mutual-auth"
	profileID             = "plab-tls13-aes128gcm-p256-mutual-auth-v2"
	serverCertProfileID   = "plab-single-leaf-p256-server-v2"
	clientCertProfileID   = "plab-single-leaf-p256-client-v2"
	alpn                  = "protocol-lab-tls"
	serverLeafDERHash     = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
	clientLeafDERHash     = "ca2e4f661e7b29cfc516c48f53c05be0ef59fb6cc410cb205f5759e07a5deb20"
	clientLeafSPKIHash    = "4b3a176400147e50a4efc3a7a26f66a9dec74a11042b7565eadd85b1ee27c0fb"
)

func main() {
	listen := flag.String("listen", configuredListenAddress(), "TLS listen address")
	certPath := flag.String("cert", envOrDefault("PLAB_TLS_CERT_FILE", materialPath("certs/leaf.pem")), "server certificate")
	keyPath := flag.String("key", envOrDefault("PLAB_TLS_KEY_FILE", materialPath("certs/leaf-key.pem")), "server private key")
	clientRootPath := flag.String("client-root", envOrDefault("PLAB_TLS_CLIENT_ROOT_CERTIFICATE_PATH", materialPath("certs/client-root.pem")), "client trust anchor")
	flag.Parse()
	if requested := strings.TrimSpace(os.Getenv("PLAB_SCENARIO_ID")); requested != "" && requested != scenarioID {
		fatal(fmt.Errorf("scenario %q is explicitly unsupported by %s", requested, implementationID))
	}

	certificate, err := tls.LoadX509KeyPair(*certPath, *keyPath)
	if err != nil {
		fatal(err)
	}
	if len(certificate.Certificate) != 1 || hash(certificate.Certificate[0]) != serverLeafDERHash {
		fatal(errors.New("server certificate substitution or chain expansion detected"))
	}
	clientRoots, err := loadRoots(*clientRootPath)
	if err != nil {
		fatal(err)
	}
	config := serverConfig(certificate, clientRoots, clientLeafDERHash, clientLeafSPKIHash)
	listener, err := tls.Listen("tcp", *listen, config)
	if err != nil {
		fatal(err)
	}
	defer listener.Close()
	writeReady(*listen)
	for {
		connection, err := listener.Accept()
		if err != nil {
			fatal(err)
		}
		go handle(connection)
	}
}

func serverConfig(certificate tls.Certificate, clientRoots *x509.CertPool, expectedDER, expectedSPKI string) *tls.Config {
	return &tls.Config{
		Certificates:           []tls.Certificate{certificate},
		MinVersion:             tls.VersionTLS13,
		MaxVersion:             tls.VersionTLS13,
		NextProtos:             []string{alpn},
		CurvePreferences:       []tls.CurveID{tls.X25519},
		ClientAuth:             tls.RequireAndVerifyClientCert,
		ClientCAs:              clientRoots,
		SessionTicketsDisabled: true,
		VerifyConnection:       peerValidator(expectedDER, expectedSPKI),
	}
}

func peerValidator(expectedDER, expectedSPKI string) func(tls.ConnectionState) error {
	return func(state tls.ConnectionState) error {
		var failures []string
		if state.Version != tls.VersionTLS13 {
			failures = append(failures, "exact TLS 1.3 was not negotiated")
		}
		if state.CipherSuite != tls.TLS_AES_128_GCM_SHA256 {
			failures = append(failures, "cipher suite mismatch")
		}
		if state.CurveID != tls.X25519 {
			failures = append(failures, "key exchange group mismatch")
		}
		if state.NegotiatedProtocol != alpn {
			failures = append(failures, "ALPN mismatch")
		}
		if state.DidResume {
			failures = append(failures, "session resumption is forbidden")
		}
		if len(state.PeerCertificates) != 1 {
			failures = append(failures, "exactly one client leaf certificate must be sent")
		}
		if len(state.VerifiedChains) != 1 || len(state.VerifiedChains[0]) != 2 {
			failures = append(failures, "client chain must verify to exactly one out-of-band trust anchor")
		}
		if len(state.PeerCertificates) > 0 {
			leaf := state.PeerCertificates[0]
			if hash(leaf.Raw) != expectedDER || hash(leaf.RawSubjectPublicKeyInfo) != expectedSPKI {
				failures = append(failures, "client certificate identity mismatch")
			}
			if leaf.SignatureAlgorithm != x509.ECDSAWithSHA256 || leaf.PublicKeyAlgorithm != x509.ECDSA {
				failures = append(failures, "client certificate algorithm mismatch")
			}
			key, ok := leaf.PublicKey.(*ecdsa.PublicKey)
			if !ok || key.Curve.Params().Name != "P-256" {
				failures = append(failures, "client certificate curve mismatch")
			}
			if !hasClientAuthEKU(leaf.ExtKeyUsage) {
				failures = append(failures, "client certificate lacks client-auth EKU")
			}
		}
		if len(failures) > 0 {
			return errors.New(strings.Join(failures, "; "))
		}
		return nil
	}
}

func handle(connection net.Conn) {
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(10 * time.Second))
	tlsConnection, ok := connection.(*tls.Conn)
	if !ok {
		fmt.Fprintln(os.Stderr, "accepted connection was not TLS")
		return
	}
	if err := tlsConnection.Handshake(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	one := make([]byte, 1)
	n, err := tlsConnection.Read(one)
	if n != 0 {
		fmt.Fprintln(os.Stderr, "application data is forbidden for the handshake-only scenario")
		return
	}
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(os.Stderr, err)
	}
}

func writeReady(listen string) {
	ready := map[string]any{
		"eventName": "ready", "implementationId": implementationID, "implementationVersion": implementationVersion,
		"listenAddress": listen, "protocol": "tls", "tlsVersion": "TLS1.3", "alpn": alpn,
		"cipherSuite": tls.CipherSuiteName(tls.TLS_AES_128_GCM_SHA256), "keyExchangeGroup": "X25519",
		"tlsProfileId": profileID, "certificateProfileId": serverCertProfileID,
		"clientCertificateProfileId": clientCertProfileID, "certificateDerSha256": serverLeafDERHash,
		"clientCertificateDerSha256": clientLeafDERHash, "clientCertificateSpkiSha256": clientLeafSPKIHash,
		"clientCertificateRequired": true, "sessionTicketsEnabled": false, "supportedScenarios": []string{scenarioID},
	}
	encoded, _ := json.Marshal(ready)
	fmt.Println(string(encoded))
}

func hasClientAuthEKU(usages []x509.ExtKeyUsage) bool {
	for _, usage := range usages {
		if usage == x509.ExtKeyUsageClientAuth {
			return true
		}
	}
	return false
}

func loadRoots(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(data) {
		return nil, errors.New("client root PEM contained no certificate")
	}
	return roots, nil
}

func configuredListenAddress() string {
	if explicit := strings.TrimSpace(os.Getenv("PLAB_LISTEN_ADDRESS")); explicit != "" {
		return explicit
	}
	if port := strings.TrimSpace(os.Getenv("PLAB_TARGET_PORT")); port != "" {
		return "127.0.0.1:" + port
	}
	return "127.0.0.1:18443"
}
func materialPath(relative string) string {
	if executable, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(executable), "..", "..", relative)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", relative)
}
func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func fatal(err error)          { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
