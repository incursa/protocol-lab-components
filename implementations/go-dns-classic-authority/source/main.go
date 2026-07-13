package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	implementationID      = "go-dns-classic-authority"
	implementationVersion = "0.1.0"
	aQueryHash            = "c46b9fb76019b5a644d0884b17e816cb7c3076275d9468c27d180f70488eb8ec"
	largeQueryHash        = "7445dfa148164c2ee02186b962f735847e9d62f46414287ebf5cf6a3dfce9e4f"
)

var (
	aResponse         = mustHex("00008400000100010000000004706c616204746573740000010001c00c00010001000000000004c0000201")
	truncatedResponse = mustHex("00008600000100000000000106646e736b657904706c6162047465737400003000010000290200000080000000")
	largeResponse     = mustHex("00008400000100070000000106646e736b657904706c616204746573740000300001c00c003000010000000000440101030d01010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101c00c003000010000000000440101030d02020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202c00c003000010000000000440101030d03030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303c00c003000010000000000440101030d04040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404040404c00c003000010000000000440101030d05050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505050505c00c003000010000000000440101030d06060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606060606c00c002e000100000000005d00300d030000000077359400713fb300303904706c6162047465737400a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a50000290200000080000000")
)

func main() {
	listen := flag.String("listen", defaultListen(), "shared UDP/TCP listen address")
	flag.Parse()
	udpAddress, err := net.ResolveUDPAddr("udp", *listen)
	if err != nil {
		fatal(err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddress)
	if err != nil {
		fatal(err)
	}
	defer udpConn.Close()
	tcpListener, err := net.Listen("tcp", *listen)
	if err != nil {
		fatal(err)
	}
	defer tcpListener.Close()
	_, port, _ := net.SplitHostPort(tcpListener.Addr().String())
	ready := map[string]any{"status": "ready", "implementationId": implementationID, "version": implementationVersion, "host": "127.0.0.1", "port": port, "protocols": []string{"dns-udp", "dns-tcp", "dns-udp-tcp-retry"}, "fixtureIds": []string{"dns.plab-test-a-v2.canonical", "dns.dnskey-plab-test-large-edns-dnssec-shaped.canonical"}, "authorityMode": "local-fixture-authoritative", "externalUpstream": "prohibited", "cacheState": "disabled", "recursionAvailable": false}
	data, _ := json.Marshal(ready)
	fmt.Println(string(data))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); serveUDP(ctx, udpConn) }()
	go func() { defer wg.Done(); serveTCP(ctx, tcpListener) }()
	<-ctx.Done()
	_ = udpConn.Close()
	_ = tcpListener.Close()
	wg.Wait()
}

func serveUDP(ctx context.Context, conn *net.UDPConn) {
	buffer := make([]byte, 65535)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, remote, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				continue
			}
			fmt.Fprintln(os.Stderr, "UDP read failed:", err)
			continue
		}
		response, err := responseFor(buffer[:n], true)
		if err != nil {
			fmt.Fprintln(os.Stderr, "UDP query rejected:", err)
			continue
		}
		if _, err := conn.WriteToUDP(response, remote); err != nil {
			fmt.Fprintln(os.Stderr, "UDP response failed:", err)
		}
	}
}

func serveTCP(ctx context.Context, listener net.Listener) {
	for {
		if tcp, ok := listener.(*net.TCPListener); ok {
			_ = tcp.SetDeadline(time.Now().Add(time.Second))
		}
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				continue
			}
			fmt.Fprintln(os.Stderr, "TCP accept failed:", err)
			continue
		}
		go handleTCP(conn)
	}
}

func handleTCP(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	for {
		prefix := make([]byte, 2)
		if _, err := io.ReadFull(conn, prefix); err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintln(os.Stderr, "TCP prefix read failed:", err)
			}
			return
		}
		length := int(binary.BigEndian.Uint16(prefix))
		if length != 27 && length != 45 {
			fmt.Fprintln(os.Stderr, "TCP query length rejected:", length)
			return
		}
		query := make([]byte, length)
		if _, err := io.ReadFull(conn, query); err != nil {
			return
		}
		response, err := responseFor(query, false)
		if err != nil {
			fmt.Fprintln(os.Stderr, "TCP query rejected:", err)
			return
		}
		framed := frame(response)
		if _, err := conn.Write(framed); err != nil {
			return
		}
	}
}

func responseFor(query []byte, udp bool) ([]byte, error) {
	if len(query) < 12 {
		return nil, errors.New("short DNS message")
	}
	id := binary.BigEndian.Uint16(query[:2])
	if id == 0 {
		return nil, errors.New("zero runtime message ID")
	}
	normalized := append([]byte(nil), query...)
	binary.BigEndian.PutUint16(normalized[:2], 0)
	var template []byte
	switch hash(normalized) {
	case aQueryHash:
		template = aResponse
	case largeQueryHash:
		if udp {
			template = truncatedResponse
		} else {
			template = largeResponse
		}
	default:
		return nil, errors.New("query outside package fixtures")
	}
	response := append([]byte(nil), template...)
	binary.BigEndian.PutUint16(response[:2], id)
	return response, nil
}

func frame(message []byte) []byte {
	framed := make([]byte, 2+len(message))
	binary.BigEndian.PutUint16(framed[:2], uint16(len(message)))
	copy(framed[2:], message)
	return framed
}
func hash(value []byte) string { sum := sha256.Sum256(value); return fmt.Sprintf("%x", sum) }
func mustHex(value string) []byte {
	b, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return b
}
func defaultListen() string {
	port := strings.TrimSpace(os.Getenv("PLAB_DNS_CLASSIC_PORT"))
	if port == "" {
		port = "15353"
	}
	if _, err := strconv.Atoi(port); err != nil {
		fatal(err)
	}
	return net.JoinHostPort("127.0.0.1", port)
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
