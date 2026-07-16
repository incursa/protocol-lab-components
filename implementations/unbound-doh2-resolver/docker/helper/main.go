package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"
)

var canonicalQuestion = []byte{4, 'p', 'l', 'a', 'b', 4, 't', 'e', 's', 't', 0, 0, 1, 0, 1}

func main() {
	authority := flag.String("authority", "127.0.0.1:5353", "fixture authority UDP address")
	listen := flag.String("listen", "0.0.0.0:444", "cache-control HTTP address")
	config := flag.String("unbound-config", "/opt/protocol-lab/unbound-resolver/unbound.conf", "Unbound configuration")
	flag.Parse()
	go serveAuthority(*authority)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/flush", func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		command := exec.Command("/opt/unbound/sbin/unbound-control", "-c", *config, "flush_zone", "plab.test.")
		if output, err := command.CombinedOutput(); err != nil {
			http.Error(w, fmt.Sprintf("flush failed: %v: %s", err, output), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := &http.Server{Addr: *listen, Handler: mux, ReadHeaderTimeout: 2 * time.Second}
	log.Fatal(server.ListenAndServe())
}

func serveAuthority(address string) {
	connection, err := net.ListenPacket("udp4", address)
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Close()
	buffer := make([]byte, 512)
	for {
		length, peer, err := connection.ReadFrom(buffer)
		if err != nil {
			log.Print(err)
			continue
		}
		request := append([]byte(nil), buffer[:length]...)
		response, ok := fixtureResponse(request)
		if !ok {
			continue
		}
		if _, err := connection.WriteTo(response, peer); err != nil {
			log.Print(err)
		}
	}
}

func fixtureResponse(request []byte) ([]byte, bool) {
	if len(request) < 12+len(canonicalQuestion) || binary.BigEndian.Uint16(request[4:6]) != 1 {
		return nil, false
	}
	for index := range canonicalQuestion {
		observed := request[12+index]
		expected := canonicalQuestion[index]
		if observed >= 'A' && observed <= 'Z' {
			observed += 'a' - 'A'
		}
		if observed != expected {
			return nil, false
		}
	}
	response := make([]byte, 43)
	copy(response[:2], request[:2])
	binary.BigEndian.PutUint16(response[2:4], 0x8400|(binary.BigEndian.Uint16(request[2:4])&0x0100))
	binary.BigEndian.PutUint16(response[4:6], 1)
	binary.BigEndian.PutUint16(response[6:8], 1)
	copy(response[12:27], request[12:27])
	copy(response[27:], []byte{0xc0, 0x0c, 0, 1, 0, 1, 0, 0, 0, 0, 0, 4, 192, 0, 2, 1})
	return response, true
}
