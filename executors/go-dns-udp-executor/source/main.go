package main

import dnsclassic "github.com/incursa/protocol-lab-components/executors/dns-classic-fixture"

func main() {
	dnsclassic.Run(dnsclassic.Config{ExecutorID: "go-dns-udp-executor", LoadGeneratorID: "go-dns-udp-load", Mode: "udp", Supported: []string{"dns.classic.udp.query.a", "dns.classic.udp-truncated-tcp-retry"}})
}
