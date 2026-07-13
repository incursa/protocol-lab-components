package main

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

var canonicalQuery = mustHex("00000000000100000000000004706c616204746573740000010001")

func validRequest() *http.Request {
	request := httptest.NewRequest(http.MethodPost, "https://dns.plab.test/dns-query", bytes.NewReader(canonicalQuery))
	request.Host = "dns.plab.test"
	request.ProtoMajor = 2
	request.ProtoMinor = 0
	request.Header.Set("Content-Type", mediaType)
	request.Header.Set("Accept", mediaType)
	request.Header.Set("Cache-Control", "no-cache")
	request.TLS = &tls.ConnectionState{Version: tls.VersionTLS13, NegotiatedProtocol: "h2", HandshakeComplete: true}
	return request
}

func TestCanonicalFixture(t *testing.T) {
	if len(canonicalQuery) != 27 || sha256Hex(canonicalQuery) != queryHash {
		t.Fatal("canonical query mismatch")
	}
	if len(canonicalResponse) != 43 || sha256Hex(canonicalResponse) != "9d488461675ad5ab9f74c7b203861e1ad17521e413a407a25e6611012a595620" {
		t.Fatal("canonical response mismatch")
	}
}

func TestHandlerAcceptsOnlyExactDoH2Contract(t *testing.T) {
	recorder := httptest.NewRecorder()
	handle(recorder, validRequest())
	if recorder.Code != 200 || recorder.Header().Get("Content-Type") != mediaType || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("response=%d headers=%v", recorder.Code, recorder.Header())
	}
	if !bytes.Equal(recorder.Body.Bytes(), canonicalResponse) {
		t.Fatal("response body mismatch")
	}
}

func TestHandlerRejectsFallbackAndSemanticSubstitution(t *testing.T) {
	mutators := []func(*http.Request){
		func(r *http.Request) { r.ProtoMajor = 1 },
		func(r *http.Request) { r.TLS.NegotiatedProtocol = "http/1.1" },
		func(r *http.Request) { r.Method = http.MethodGet },
		func(r *http.Request) { r.URL.Path = "/" },
		func(r *http.Request) { r.Header.Set("Content-Type", "application/json") },
	}
	for index, mutate := range mutators {
		request := validRequest()
		mutate(request)
		recorder := httptest.NewRecorder()
		handle(recorder, request)
		if recorder.Code == 200 {
			t.Fatalf("mutator %d was accepted", index)
		}
	}
}
