package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PLAB_HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: newHandler()}
	log.Printf("go-nethttp-http1 listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok","implementationId":"go-nethttp-http1","protocol":"h1"}`)
	})
	mux.HandleFunc("/protocol-lab/metadata", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"implementationId":"go-nethttp-http1","packageId":"org.protocol-lab.components.implementation.go-nethttp-http1","protocol":"h1","protocolVersion":"http/1.1","supportedScenarios":["http1.core.plaintext","http1.core.json"]}`)
	})
	mux.HandleFunc("/plaintext", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Hello, World!")
	})
	mux.HandleFunc("/json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"Hello, World!"}`)
	})
	return mux
}
