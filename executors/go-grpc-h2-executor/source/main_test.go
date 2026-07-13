package main

import (
	"bytes"
	"testing"
	"time"
)

func TestCanonicalScenarioByteScopes(t *testing.T) {
	expected := map[string]struct{ payload, protobuf, frame string }{
		"grpc.h2.unary.echo":             {"394394b5f0e91a21d1e932f9ed55e098c8b05f3668f77134eeee843fef1d1758", "c2046bce7238a2a9cae159ba85a75e93f07c74073fe7b83d2be713caef717cb4", "98f1fce70c79cf7da3649fc3ecfc77a975c1427c9f8afcd1a41d85559164525a"},
		"grpc.h2.unary.empty":            {"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "8855508aade16ec573d21e6a485dfd0a7624085c1a14b5ecdd6485de0c6839a4"},
		"grpc.h2.unary.fixed-metadata":   {"394394b5f0e91a21d1e932f9ed55e098c8b05f3668f77134eeee843fef1d1758", "c2046bce7238a2a9cae159ba85a75e93f07c74073fe7b83d2be713caef717cb4", "98f1fce70c79cf7da3649fc3ecfc77a975c1427c9f8afcd1a41d85559164525a"},
		"grpc.h2.unary.gzip":             {"9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a", "0a685639b341882d59f7b7da8308517b2b7862047846d3294b4ba9e8d48a4322", ""},
		"grpc.h2.unary.large":            {"b8824ab1d764167b60ec900ed95085d72dc8768660469a74effe79a0c22154e6", "18a46662a9a9321dc00af0b2f75b603a5a21416e6ea321b033b94d5c14769640", "f67e1ff2509e0ae9a59ee55eeed9d72ff6dd170798bcff37cce6c3d2bd6a9f2a"},
		"grpc.h2.server-streaming.echo":  {"9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a", "0a685639b341882d59f7b7da8308517b2b7862047846d3294b4ba9e8d48a4322", "5027e498c899a73e723e4abc3a12416e375826f4de8a7dbb38d066193ad4e93e"},
		"grpc.h2.client-streaming.echo":  {"9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a", "0a685639b341882d59f7b7da8308517b2b7862047846d3294b4ba9e8d48a4322", "5027e498c899a73e723e4abc3a12416e375826f4de8a7dbb38d066193ad4e93e"},
		"grpc.h2.bidi-streaming.echo":    {"9b6ce55f379e9771551de6939556a7e6b949814ae27c2f5cfd5dbeb378ce7c2a", "0a685639b341882d59f7b7da8308517b2b7862047846d3294b4ba9e8d48a4322", "5027e498c899a73e723e4abc3a12416e375826f4de8a7dbb38d066193ad4e93e"},
		"grpc.h2.trailers-only-status":   {"394394b5f0e91a21d1e932f9ed55e098c8b05f3668f77134eeee843fef1d1758", "c2046bce7238a2a9cae159ba85a75e93f07c74073fe7b83d2be713caef717cb4", "98f1fce70c79cf7da3649fc3ecfc77a975c1427c9f8afcd1a41d85559164525a"},
		"grpc.h2.deadline-exceeded":      {"394394b5f0e91a21d1e932f9ed55e098c8b05f3668f77134eeee843fef1d1758", "c2046bce7238a2a9cae159ba85a75e93f07c74073fe7b83d2be713caef717cb4", "98f1fce70c79cf7da3649fc3ecfc77a975c1427c9f8afcd1a41d85559164525a"},
		"grpc.h2.client-cancellation":    {"394394b5f0e91a21d1e932f9ed55e098c8b05f3668f77134eeee843fef1d1758", "c2046bce7238a2a9cae159ba85a75e93f07c74073fe7b83d2be713caef717cb4", "98f1fce70c79cf7da3649fc3ecfc77a975c1427c9f8afcd1a41d85559164525a"},
		"grpc.h2.unary.echo-new-channel": {"394394b5f0e91a21d1e932f9ed55e098c8b05f3668f77134eeee843fef1d1758", "c2046bce7238a2a9cae159ba85a75e93f07c74073fe7b83d2be713caef717cb4", "98f1fce70c79cf7da3649fc3ecfc77a975c1427c9f8afcd1a41d85559164525a"},
	}
	for _, id := range supportedScenarioIDs {
		t.Run(id, func(t *testing.T) {
			s, ok := specFor(id)
			if !ok {
				t.Fatal("missing spec")
			}
			frame, err := encodeFrame(encodeProtobuf(s.payload), s.compression)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateResponseFrame(frame, s); err != nil {
				t.Fatal(err)
			}
			e := expected[id]
			if s.payloadHash != e.payload || s.protobufHash != e.protobuf || (e.frame != "" && s.frameHash != e.frame) {
				t.Fatalf("public byte-scope hash drift for %s", id)
			}
		})
	}
}

func TestStreamingSpecsMatchPublicCardinalityAndLifecycle(t *testing.T) {
	expected := map[string]struct {
		requestCount, responseCount int
		rpcType                     string
	}{
		"grpc.h2.server-streaming.echo": {1, 100, "server-streaming"},
		"grpc.h2.client-streaming.echo": {100, 1, "client-streaming"},
		"grpc.h2.bidi-streaming.echo":   {100, 100, "bidirectional-streaming"},
	}
	for id, want := range expected {
		spec, ok := specFor(id)
		if !ok || spec.requestCount != want.requestCount || spec.responseCount != want.responseCount || spec.rpcType != want.rpcType {
			t.Fatalf("%s cardinality/lifecycle mismatch: %#v", id, spec)
		}
		frames := bytes.Repeat(mustFrame(t, spec), spec.responseCount)
		body, arrivals, count, err := readAndValidateResponseFrames(bytes.NewReader(frames), spec, time.Now())
		if err != nil || count != spec.responseCount || len(body) != spec.responseCount*spec.frameBytes {
			t.Fatalf("%s response sequence rejected: count=%d bytes=%d err=%v", id, count, len(body), err)
		}
		if len(arrivals) != spec.responseCount {
			t.Fatalf("%s message-arrival evidence count mismatch", id)
		}
	}
}

func mustFrame(t *testing.T, spec scenarioSpec) []byte {
	t.Helper()
	frame, err := encodeFrame(encodeProtobuf(spec.payload), spec.compression)
	if err != nil {
		t.Fatal(err)
	}
	return frame
}
func TestSelectionAcceptsExactIdentities(t *testing.T) {
	s, _ := specFor("grpc.h2.unary.empty")
	setSelection(t, s.id, loadProfileDiagnostic)
	if _, err := validateSelection(s); err != nil {
		t.Fatal(err)
	}
}

func TestFixedMetadataResultCarriesExactRequestProof(t *testing.T) {
	s, _ := specFor("grpc.h2.unary.fixed-metadata")
	result := baseResult(s, loadProfileDiagnostic)
	if result.Request["requestInitialTextMetadata"] != "protocol-lab" || result.Request["requestInitialBinaryMetadata"] != "AAECAw==" || result.Request["requestInitialBinaryMetadataDecodedSha256"] != sha256Hex([]byte{0, 1, 2, 3}) {
		t.Fatal("fixed request metadata proof is incomplete")
	}
}
func TestSelectionRejectsScenarioSubstitution(t *testing.T) {
	s, _ := specFor("grpc.h2.unary.empty")
	setSelection(t, "grpc.h2.unary.echo", loadProfileDiagnostic)
	if _, err := validateSelection(s); err == nil {
		t.Fatal("scenario substitution accepted")
	}
}
func TestEveryCommittedGRPCIdentityIsImplemented(t *testing.T) {
	if len(knownUnsupportedScenarioIDs) != 0 {
		t.Fatalf("remaining unsupported identities: %v", knownUnsupportedScenarioIDs)
	}
	for _, id := range []string{"grpc.h2.trailers-only-status", "grpc.h2.deadline-exceeded", "grpc.h2.client-cancellation", "grpc.h2.unary.echo-new-channel"} {
		if _, ok := specFor(id); !ok {
			t.Fatalf("missing spec for %s", id)
		}
	}
	if contains(knownUnsupportedScenarioIDs, "grpc.h2.not-real") {
		t.Fatal("unknown classified as known")
	}
}
func setSelection(t *testing.T, scenario, profile string) {
	t.Helper()
	duration, warmup, repetition, variant := "5", "1", "1", protocolVariant
	if profile == loadProfileDiagnostic {
		duration, warmup = "10", "0"
	}
	if profile == loadProfileChannelChurn {
		duration, warmup, repetition, variant = "30", "0", "3", protocolVariantNewChannel
	}
	values := map[string]string{"PLAB_EXECUTOR_ID": executorID, "PLAB_EXECUTOR_VERSION": executorVersion, "PLAB_LOAD_GENERATOR_ID": loadGeneratorID, "PLAB_LOAD_GENERATOR_VERSION": loadGeneratorVersion, "PLAB_SCENARIO_ID": scenario, "PLAB_LOAD_PROFILE_ID": profile, "PLAB_PROTOCOL": "h2", "PLAB_PROTOCOL_VARIANT": variant, "PLAB_CONNECTIONS": "1", "PLAB_CONCURRENCY": "1", "PLAB_STREAMS_PER_CONNECTION": "1", "PLAB_DURATION_SECONDS": duration, "PLAB_WARMUP_SECONDS": warmup, "PLAB_REPETITION": repetition}
	for k, v := range values {
		t.Setenv(k, v)
	}
}
