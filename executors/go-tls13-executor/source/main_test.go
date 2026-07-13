package main

import (
	"crypto/tls"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTLSHandshakeSmokeConfigIsExact(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", fullScenarioID)
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

func TestTLSResumedHandshakeSmokeConfigIsExact(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", resumedScenarioID)
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
	if config.ScenarioID != resumedScenarioID {
		t.Fatalf("unexpected scenario: %+v", config)
	}
}

func TestUnsupportedTLSScenarioFailsClosed(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", "tls.early-data.accepted")
	if !isKnownUnsupportedScenario(os.Getenv("PLAB_SCENARIO_ID")) {
		t.Fatal("expected committed scenario to be recognized as unsupported")
	}
	t.Setenv("PLAB_SCENARIO_ID", "tls.unknown")
	if _, _, err := requestedScenario(); err == nil || !strings.Contains(err.Error(), "unknown or invalid") {
		t.Fatalf("expected invalid scenario error, got %v", err)
	}
}

func TestSessionTicketCacheConsumesOneTicketOnce(t *testing.T) {
	cache := &singleUseSessionCache{}
	session := &tls.ClientSessionState{}
	cache.Put("tls.plab.test", session)
	actual, ok := cache.Get("tls.plab.test")
	if !ok || actual != session {
		t.Fatal("expected the priming ticket")
	}
	if _, ok := cache.Get("tls.plab.test"); ok {
		t.Fatal("ticket was reused")
	}
	puts, gets, hits, available := cache.counts()
	if puts != 1 || gets != 2 || hits != 1 || available {
		t.Fatalf("unexpected cache proof: puts=%d gets=%d hits=%d available=%t", puts, gets, hits, available)
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
