package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponseHeadersHandlerEmitsCanonicalFixture(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/headers/response?count=50&size=32", nil)
	recorder := httptest.NewRecorder()

	responseHeadersHandler(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := recorder.Body.String(), "headers"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	for index := 0; index < 50; index++ {
		name := fmt.Sprintf("x-protocol-bench-header-%02d", index)
		if got, want := recorder.Header().Get(name), strings.Repeat("h", 32); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestStreamBytesHandlerStreamsCanonicalQuery(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/stream/bytes?chunks=100&size=16384&delayMs=0", nil)

	routes().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status code = %d, want %d", got, want)
	}

	if got, want := recorder.Header().Get("Content-Type"), "application/octet-stream"; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}

	if got, want := recorder.Body.Len(), streamBytesCanonicalRows*streamBytesChunkSize; got != want {
		t.Fatalf("body length = %d, want %d", got, want)
	}

	if got, want := recorder.Body.Bytes()[:16], deterministicChunk(16); !bytes.Equal(got, want) {
		t.Fatalf("body prefix = %v, want %v", got, want)
	}
}

func TestStreamBytesHandlerRejectsNonCanonicalQuery(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/stream/bytes?chunks=99&size=16384&delayMs=0", nil)

	routes().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status code = %d, want %d", got, want)
	}
}

func TestStreamBytesStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	writer := &trackingResponseWriter{}
	err := streamBytes(ctx, writer, streamBytesParams{chunks: streamBytesCanonicalRows, size: streamBytesChunkSize})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("streamBytes error = %v, want context.Canceled", err)
	}

	if got, want := writer.writeCalls, 0; got != want {
		t.Fatalf("write calls = %d, want %d", got, want)
	}

	if got, want := writer.flushCalls, 0; got != want {
		t.Fatalf("flush calls = %d, want %d", got, want)
	}
}

func TestStreamBytesWritesChunkByChunk(t *testing.T) {
	writer := &trackingResponseWriter{}
	err := streamBytes(context.Background(), writer, streamBytesParams{chunks: streamBytesCanonicalRows, size: streamBytesChunkSize})
	if err != nil {
		t.Fatalf("streamBytes returned error: %v", err)
	}

	if got, want := writer.writeCalls, streamBytesCanonicalRows; got != want {
		t.Fatalf("write calls = %d, want %d", got, want)
	}

	if got, want := writer.flushCalls, streamBytesCanonicalRows; got != want {
		t.Fatalf("flush calls = %d, want %d", got, want)
	}

	if got, want := writer.body.Len(), streamBytesCanonicalRows*streamBytesChunkSize; got != want {
		t.Fatalf("body length = %d, want %d", got, want)
	}
}

type trackingResponseWriter struct {
	header     http.Header
	body       bytes.Buffer
	writeCalls int
	flushCalls int
}

func (w *trackingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}

	return w.header
}

func (w *trackingResponseWriter) WriteHeader(statusCode int) {}

func (w *trackingResponseWriter) Write(p []byte) (int, error) {
	w.writeCalls++
	return w.body.Write(p)
}

func (w *trackingResponseWriter) Flush() {
	w.flushCalls++
}
