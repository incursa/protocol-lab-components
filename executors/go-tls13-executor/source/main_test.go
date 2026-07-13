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
	t.Setenv("PLAB_LOAD_PROFILE_ID", tlsSmokeProfileID)
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
	t.Setenv("PLAB_LOAD_PROFILE_ID", tlsSmokeProfileID)
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

func TestTLSRecordThroughputSmokeConfigIsExact(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", recordThroughputScenarioID)
	t.Setenv("PLAB_LOAD_PROFILE_ID", tlsSmokeProfileID)
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
	if config.ApplicationDataBytes != 1048576 {
		t.Fatalf("unexpected config: %+v", config)
	}
}

func TestTLSRecordCoverageDiagnosticConfigIsExact(t *testing.T) {
	t.Setenv("PLAB_SCENARIO_ID", recordCoverageScenarioID)
	t.Setenv("PLAB_LOAD_PROFILE_ID", tlsDiagnosticProfileID)
	t.Setenv("PLAB_CONNECTIONS", "1")
	t.Setenv("PLAB_CONCURRENCY", "1")
	t.Setenv("PLAB_TOTAL_OPERATIONS", "1")
	t.Setenv("PLAB_DURATION_SECONDS", "10")
	t.Setenv("PLAB_WARMUP_SECONDS", "0")
	t.Setenv("PLAB_REPETITION", "1")
	t.Setenv("PLAB_REQUEST_TIMEOUT_SECONDS", "15")
	config, err := loadConfigFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if config.TotalOperations != 1 {
		t.Fatalf("unexpected config: %+v", config)
	}
}

func TestCanonicalPayloadHashes(t *testing.T) {
	for size, expected := range map[int]string{
		1024:    "e8fb68ce4d4d002dba40c0a459d96807c96ded1c2fdefae3f56f8a0c06a4fecf",
		65536:   "944044fe482bc4e91085c15c5a923a1b9e02eac98d3bce04997d6dbecd2a5b8d",
		1048576: "bf63d8a95fcc2e64619813aae35fdcbe871fdd9264caa3f365eb3aed0f679129",
	} {
		if actual := payloadHash(size); actual != expected {
			t.Fatalf("size=%d actual=%s", size, actual)
		}
	}
}

func TestRecordParserHandlesSplitRecords(t *testing.T) {
	record := []byte{23, 3, 3, 0, 3, 1, 2, 3}
	var parser tlsRecordParser
	parser.feed(record[:4])
	parser.feed(record[4:])
	if parser.count != 1 || parser.bytes != len(record) {
		t.Fatalf("parser=%+v", parser)
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
