package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const canonicalServiceDigest = "b7b987814f8af5cd4f15c03989b9c309c1c0ec643972ae32668304d71502120f"

type serviceContract struct {
	ContractID string `json:"contractId"`
	Package    string `json:"package"`
	Service    string `json:"service"`
	Messages   []struct {
		Name   string `json:"name"`
		Fields []struct {
			Number int    `json:"number"`
			Name   string `json:"name"`
			Type   string `json:"type"`
		} `json:"fields"`
	} `json:"messages"`
	CompressionProfiles []struct {
		ID             string `json:"id"`
		GRPCEncoding   string `json:"grpcEncoding"`
		CompressedFlag int    `json:"compressedFlag"`
	} `json:"compressionProfiles"`
	MetadataProfiles []struct {
		ID string `json:"id"`
	} `json:"metadataProfiles"`
	Methods []struct {
		Name               string `json:"name"`
		Input              string `json:"input"`
		Output             string `json:"output"`
		ClientStreaming    bool   `json:"clientStreaming"`
		ServerStreaming    bool   `json:"serverStreaming"`
		CompressionProfile string `json:"compressionProfile"`
		MetadataProfile    string `json:"metadataProfile"`
		TerminalStatus     struct {
			Code int    `json:"code"`
			Name string `json:"name"`
		} `json:"terminalStatus"`
	} `json:"methods"`
}

func TestProtoAndImplementationDescriptorMatchPublicV2Contract(t *testing.T) {
	contractPath := filepath.Join("..", "..", "..", "scenarios", "grpc-h2-performance", "fixtures", "public-contracts", "grpc", "v2", "valid", "echo-service-v2.json")
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	var canonicalValue any
	if err := json.Unmarshal(raw, &canonicalValue); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(canonicalValue)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	if observed := hex.EncodeToString(digest[:]); observed != canonicalServiceDigest {
		t.Fatalf("canonical service digest mismatch: expected %s, observed %s", canonicalServiceDigest, observed)
	}
	var contract serviceContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Package != "protocollab.performance.v1" || contract.Service != "EchoService" || contract.ContractID != "protocollab.performance.v1.EchoService" {
		t.Fatalf("unexpected public identity: %#v", contract)
	}
	if len(contract.Messages) != 2 || len(contract.Methods) != 9 || len(contract.CompressionProfiles) != 2 || len(contract.MetadataProfiles) != 2 {
		t.Fatalf("unexpected public contract breadth: messages=%d methods=%d compression=%d metadata=%d", len(contract.Messages), len(contract.Methods), len(contract.CompressionProfiles), len(contract.MetadataProfiles))
	}
	for _, message := range contract.Messages {
		if len(message.Fields) != 1 || message.Fields[0].Number != 1 || message.Fields[0].Name != "payload" || message.Fields[0].Type != "bytes" {
			t.Fatalf("message %s does not match the one-field bytes contract", message.Name)
		}
	}

	protoPath := filepath.Join("..", "contract", "echo.proto")
	protoBytes, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatal(err)
	}
	proto := string(protoBytes)
	if !strings.Contains(proto, "package "+contract.Package+";") || !strings.Contains(proto, "service "+contract.Service) {
		t.Fatal("proto package or service does not match public contract")
	}
	for _, message := range contract.Messages {
		pattern := regexp.MustCompile(`message\s+` + regexp.QuoteMeta(message.Name) + `\s*\{\s*bytes\s+payload\s*=\s*1\s*;\s*\}`)
		if !pattern.MatchString(proto) {
			t.Fatalf("proto message %s does not match public contract", message.Name)
		}
	}
	for _, method := range contract.Methods {
		input := regexp.QuoteMeta(method.Input)
		if method.ClientStreaming {
			input = `stream\s+` + input
		}
		output := regexp.QuoteMeta(method.Output)
		if method.ServerStreaming {
			output = `stream\s+` + output
		}
		pattern := regexp.MustCompile(`rpc\s+` + regexp.QuoteMeta(method.Name) + `\s*\(\s*` + input + `\s*\)\s*returns\s*\(\s*` + output + `\s*\)\s*;`)
		if !pattern.MatchString(proto) {
			t.Fatalf("proto method %s or its streaming flags do not match public contract", method.Name)
		}
	}

	assertIDs(t, compressionIDs(contract), []string{"gzip-semantic-v1", "identity-v1"}, "compression profiles")
	assertIDs(t, metadataIDs(contract), []string{"fixed-ascii-and-binary-metadata-v1", "fixed-empty-user-metadata"}, "metadata profiles")
	type methodBinding struct {
		status                            int
		statusName, compression, metadata string
	}
	expectedBindings := map[string]methodBinding{
		"UnaryEcho":                  {0, "OK", "identity-v1", "fixed-empty-user-metadata"},
		"ServerStreamingEcho":        {0, "OK", "identity-v1", "fixed-empty-user-metadata"},
		"ClientStreamingEcho":        {0, "OK", "identity-v1", "fixed-empty-user-metadata"},
		"BidirectionalStreamingEcho": {0, "OK", "identity-v1", "fixed-empty-user-metadata"},
		"TrailersOnlyStatus":         {3, "INVALID_ARGUMENT", "identity-v1", "fixed-empty-user-metadata"},
		"DeadlineExceeded":           {4, "DEADLINE_EXCEEDED", "identity-v1", "fixed-empty-user-metadata"},
		"ClientCancellation":         {1, "CANCELLED", "identity-v1", "fixed-empty-user-metadata"},
		"UnaryGzip":                  {0, "OK", "gzip-semantic-v1", "fixed-empty-user-metadata"},
		"UnaryFixedMetadata":         {0, "OK", "identity-v1", "fixed-ascii-and-binary-metadata-v1"},
	}
	for _, method := range contract.Methods {
		expected, ok := expectedBindings[method.Name]
		if !ok || expected.status != method.TerminalStatus.Code || expected.statusName != method.TerminalStatus.Name || expected.compression != method.CompressionProfile || expected.metadata != method.MetadataProfile {
			t.Fatalf("method %s status/compression/metadata binding mismatch", method.Name)
		}
	}
}

func compressionIDs(contract serviceContract) []string {
	ids := make([]string, 0, len(contract.CompressionProfiles))
	for _, profile := range contract.CompressionProfiles {
		ids = append(ids, profile.ID)
	}
	return ids
}

func metadataIDs(contract serviceContract) []string {
	ids := make([]string, 0, len(contract.MetadataProfiles))
	for _, profile := range contract.MetadataProfiles {
		ids = append(ids, profile.ID)
	}
	return ids
}

func assertIDs(t *testing.T, actual, expected []string, label string) {
	t.Helper()
	sort.Strings(actual)
	if strings.Join(actual, ",") != strings.Join(expected, ",") {
		t.Fatalf("unexpected %s: %v", label, actual)
	}
}
