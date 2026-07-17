package main

import (
	"log"
	"net"
)

func main() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 4433})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Printf("role=udp-target authority=masque-echo.plab.test:4433 bind=:4433 behavior=exact-echo ready=true")
	buffer := make([]byte, 1500)
	for {
		n, peer, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal(err)
		}
		if n != 256 {
			log.Printf("role=udp-target peer=%s rejected_bytes=%d", peer, n)
			continue
		}
		if _, err = conn.WriteToUDP(buffer[:n], peer); err != nil {
			log.Printf("role=udp-target peer=%s echo_error=%q", peer, err)
			continue
		}
		log.Printf("role=udp-target peer=%s echoed_bytes=%d", peer, n)
	}
}
