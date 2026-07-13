package main

import (
	"strings"
	"testing"
	"time"
)

func TestTLSHandshakeSmokeConfigIsExact(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", scenarioID)
	t.Setenv("PLAB_LOAD_PROFILE_ID", loadProfileID)
	t.Setenv("PLAB_CONNECTIONS", "1")
	t.Setenv("PLAB_CONCURRENCY", "1")
	t.Setenv("PLAB_DURATION_SECONDS", "5")
	t.Setenv("PLAB_WARMUP_SECONDS", "1")
	t.Setenv("PLAB_REPETITION", "1")
	t.Setenv("PLAB_REQUEST_TIMEOUT_SECONDS", "5")
	config, err := loadConfigFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if config.Duration != 5*time.Second || config.ConnectionTimeout != 5*time.Second {
		t.Fatalf("unexpected config: %+v", config)
	}
	t.Setenv("PLAB_REQUEST_TIMEOUT_SECONDS", "6")
	if _, err := loadConfigFromEnvironment(); err == nil || !strings.Contains(err.Error(), "operationTimeout=5s") {
		t.Fatalf("expected operation-timeout rejection, got %v", err)
	}
}

func TestTargetNormalizationRejectsNonTLSURLs(t *testing.T) {
	if _, err := normalizeTarget("https://example.test:443"); err == nil {
		t.Fatal("expected HTTPS URL rejection")
	}
	if actual, err := normalizeTarget("tls://127.0.0.1:8443"); err != nil || actual != "127.0.0.1:8443" {
		t.Fatalf("actual=%q err=%v", actual, err)
	}
}

func TestPercentilesUseNearestRank(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	if percentile(values, .50) != 3 || percentile(values, .95) != 5 {
		t.Fatalf("unexpected percentiles")
	}
}
