package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:854", "control HTTP address")
	config := flag.String("rndc-config", "/etc/bind/plab/rndc.conf", "rndc configuration")
	flag.Parse()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/flush", func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		command := exec.Command("rndc", "-c", *config, "flush")
		if output, err := command.CombinedOutput(); err != nil {
			http.Error(w, fmt.Sprintf("flush failed: %v: %s", err, output), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := &http.Server{Addr: *listen, Handler: mux, ReadHeaderTimeout: 2 * time.Second}
	log.Fatal(server.ListenAndServe())
}
