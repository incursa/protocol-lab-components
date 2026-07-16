package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginRows(t *testing.T) {
	tests := []struct{ path, contentType, body string }{
		{"/plaintext", "text/plain", "Hello, World!"},
		{"/json", "application/json", `{"message":"Hello, World!"}`},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		newHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		body, _ := io.ReadAll(recorder.Result().Body)
		if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != test.contentType || string(body) != test.body {
			t.Fatalf("%s: status=%d content-type=%q body=%q", test.path, recorder.Code, recorder.Header().Get("Content-Type"), body)
		}
	}
}
