package main

import "testing"

func TestConfiguredListenAddressPrefersExactEnvironment(t *testing.T) {
	t.Setenv("PLAB_LISTEN_ADDRESS", "127.0.0.1:19443")
	t.Setenv("PLAB_TARGET_PORT", "20443")
	if actual := configuredListenAddress(); actual != "127.0.0.1:19443" {
		t.Fatalf("unexpected address %q", actual)
	}
}

func TestTargetIdentityIsStable(t *testing.T) {
	if implementationID != "go-tls13" || implementationVersion != "0.1.0" || alpn != "protocol-lab-tls" {
		t.Fatal("target identity changed")
	}
}
