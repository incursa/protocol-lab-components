package main

import dnsclassic "github.com/incursa/protocol-lab-components/executors/dns-classic-fixture"

func main() {
	dnsclassic.Run(dnsclassic.Config{ExecutorID: "go-dns-tcp-executor", LoadGeneratorID: "go-dns-tcp-load", Mode: "tcp", Supported: []string{"dns.classic.tcp.query.a"}})
}
