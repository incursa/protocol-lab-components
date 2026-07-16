package main

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/http2"
)

func TestH2cOriginRows(t *testing.T) {
	server := httptest.NewServer(newHandler())
	defer server.Close()
	transport := &http2.Transport{AllowHTTP: true, DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) { return (&net.Dialer{}).DialContext(ctx, network, addr) }}
	client := &http.Client{Transport: transport}
	for _, test := range []struct{ path, contentType, body string }{{"/plaintext", "text/plain", "Hello, World!"}, {"/json", "application/json", `{"message":"Hello, World!"}`}} {
		response, err := client.Get(server.URL + test.path)
		if err != nil { t.Fatal(err) }
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.ProtoMajor != 2 || response.Header.Get("Content-Type") != test.contentType || string(body) != test.body { t.Fatalf("%s: proto=%s content-type=%q body=%q", test.path, response.Proto, response.Header.Get("Content-Type"), body) }
	}
}
