package main

import "testing"

func TestCanonicalFrameByteScopes(t *testing.T) {
	frame := canonicalFrame()
	if err := validateCanonicalFrame(frame); err != nil {
		t.Fatal(err)
	}
	if len(frame) != 136 || len(frame[5:]) != 131 || len(frame[8:]) != 128 {
		t.Fatalf("unexpected byte scopes: %d/%d/%d", len(frame), len(frame[5:]), len(frame[8:]))
	}
	if sha256Hex(frame) != expectedFrameHash || sha256Hex(frame[5:]) != expectedProtobufHash || sha256Hex(frame[8:]) != expectedPayloadHash {
		t.Fatal("canonical hash constants drifted")
	}
}

func TestSelectionRejectsScenarioSubstitution(t *testing.T) {
	setExactSelection(t)
	t.Setenv("PLAB_SCENARIO_ID", "grpc.h2.server-streaming.echo")
	if err := validateSelection(); err == nil {
		t.Fatal("unsupported scenario was accepted")
	}
}

func TestSelectionAcceptsExactIdentities(t *testing.T) {
	setExactSelection(t)
	if err := validateSelection(); err != nil {
		t.Fatal(err)
	}
}

func TestSelectionRejectsMissingIdentity(t *testing.T) {
	setExactSelection(t)
	t.Setenv("PLAB_LOAD_GENERATOR_ID", "")
	if err := validateSelection(); err == nil {
		t.Fatal("missing load-generator identity was accepted")
	}
}

func TestSelectionRejectsLoadSubstitution(t *testing.T) {
	setExactSelection(t)
	t.Setenv("PLAB_CONCURRENCY", "2")
	if err := validateSelection(); err == nil {
		t.Fatal("unsupported load shape was accepted")
	}
}

func setExactSelection(t *testing.T) {
	t.Helper()
	values := map[string]string{"PLAB_EXECUTOR_ID": executorID, "PLAB_EXECUTOR_VERSION": executorVersion, "PLAB_LOAD_GENERATOR_ID": loadGeneratorID, "PLAB_LOAD_GENERATOR_VERSION": loadGeneratorVersion, "PLAB_SCENARIO_ID": scenarioID, "PLAB_LOAD_PROFILE_ID": loadProfileID, "PLAB_PROTOCOL": "h2", "PLAB_PROTOCOL_VARIANT": protocolVariant, "PLAB_CONNECTIONS": "1", "PLAB_CONCURRENCY": "1", "PLAB_STREAMS_PER_CONNECTION": "1", "PLAB_DURATION_SECONDS": "5", "PLAB_WARMUP_SECONDS": "1", "PLAB_REPETITION": "1"}
	for name, value := range values {
		t.Setenv(name, value)
	}
}
