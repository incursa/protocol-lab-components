package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	port := os.Getenv("PLAB_HTTP_PORT")
	if port == "" {
		port = "8082"
	}
	server := &http.Server{Addr: ":" + port, Handler: newHandler()}
	log.Printf("go-nethttp-http2 listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok","implementationId":"go-nethttp-http2","protocol":"h2"}`)
	})
	mux.HandleFunc("/protocol-lab/metadata", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"implementationId":"go-nethttp-http2","packageId":"org.protocol-lab.components.implementation.go-nethttp-http2","protocol":"h2","protocolVersion":"h2c","supportedScenarios":["http2.core.plaintext","http2.core.json"]}`)
	})
	mux.HandleFunc("/plaintext", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Hello, World!")
	})
	mux.HandleFunc("/json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"Hello, World!"}`)
	})
	return h2c.NewHandler(mux, &http2.Server{})
}
