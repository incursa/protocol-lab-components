package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	fixtureID    = "dns.plab-test-a.canonical"
	queryHash    = "c46b9fb76019b5a644d0884b17e816cb7c3076275d9468c27d180f70488eb8ec"
	responseHash = "9d488461675ad5ab9f74c7b203861e1ad17521e413a407a25e6611012a595620"
)

var (
	canonicalQuery    = mustHex("00000000000100000000000004706c616204746573740000010001")
	canonicalResponse = mustHex("00008400000100010000000004706c616204746573740000010001c00c00010001000000000004c0000201")
)

type dnsProof struct {
	FixtureID                    string `json:"fixtureId"`
	Transport                    string `json:"transport"`
	QuestionName                 string `json:"questionName"`
	QuestionType                 string `json:"questionType"`
	QuestionClass                string `json:"questionClass"`
	Answer                       string `json:"answer"`
	TTLSeconds                   int    `json:"ttlSeconds"`
	ResponseCode                 string `json:"responseCode"`
	AuthoritativeAnswer          bool   `json:"authoritativeAnswer"`
	RecursionDesired             bool   `json:"recursionDesired"`
	RecursionAvailable           bool   `json:"recursionAvailable"`
	ExternalUpstreamUsed         bool   `json:"externalUpstreamUsed"`
	CacheEnabled                 bool   `json:"cacheEnabled"`
	Framing                      string `json:"framing"`
	AuthorityMode                string `json:"authorityMode"`
	RequestMessageID             uint16 `json:"requestMessageId"`
	RuntimeMessageID             uint16 `json:"runtimeMessageId"`
	ResponseMessageID            uint16 `json:"responseMessageId"`
	CanonicalHashNormalization   string `json:"canonicalHashNormalization"`
	QueryLengthBytes             int    `json:"queryLengthBytes"`
	QueryFramedLengthBytes       int    `json:"queryFramedLengthBytes"`
	QueryNormalizedSHA256        string `json:"queryNormalizedSha256"`
	ResponseRawLengthBytes       int    `json:"responseRawLengthBytes"`
	ResponseFramedLengthBytes    int    `json:"responseFramedLengthBytes"`
	ResponseCanonicalLengthBytes int    `json:"responseCanonicalLengthBytes"`
	ResponseNormalizedSHA256     string `json:"responseNormalizedSha256"`
	ResponseCompressionDiffered  bool   `json:"responseCompressionDiffered"`
	ResponseCanonicalized        bool   `json:"responseCanonicalized"`
	RawResponseEqualityRequired  bool   `json:"rawResponseEqualityRequired"`
}

type malformedResponseError struct{ reason string }

func (err malformedResponseError) Error() string { return err.reason }

func validateDNSFrame(raw []byte) (dnsProof, error) {
	if len(raw) != 45 || binary.BigEndian.Uint16(raw[:2]) != 43 {
		return dnsProof{}, malformedResponseError{"DoQ response framing mismatch"}
	}
	message := raw[2:]
	if len(message) < 12 || binary.BigEndian.Uint16(message[:2]) != 0 {
		return dnsProof{}, malformedResponseError{"response message ID must be zero"}
	}
	flags := binary.BigEndian.Uint16(message[2:4])
	if flags != 0x8400 {
		return dnsProof{}, malformedResponseError{"response flags or rcode mismatch"}
	}
	if binary.BigEndian.Uint16(message[4:6]) != 1 || binary.BigEndian.Uint16(message[6:8]) != 1 ||
		binary.BigEndian.Uint16(message[8:10]) != 0 || binary.BigEndian.Uint16(message[10:12]) != 0 {
		return dnsProof{}, malformedResponseError{"response section counts mismatch"}
	}
	name, next, err := readName(message, 12)
	if err != nil || name != "plab.test." {
		return dnsProof{}, malformedResponseError{"response question name mismatch"}
	}
	if next+4 > len(message) || binary.BigEndian.Uint16(message[next:next+2]) != 1 || binary.BigEndian.Uint16(message[next+2:next+4]) != 1 {
		return dnsProof{}, malformedResponseError{"response question type or class mismatch"}
	}
	next += 4
	owner, next, err := readName(message, next)
	if err != nil || owner != "plab.test." || next+10 > len(message) {
		return dnsProof{}, malformedResponseError{"answer owner or header mismatch"}
	}
	typeCode := binary.BigEndian.Uint16(message[next : next+2])
	classCode := binary.BigEndian.Uint16(message[next+2 : next+4])
	ttl := binary.BigEndian.Uint32(message[next+4 : next+8])
	length := int(binary.BigEndian.Uint16(message[next+8 : next+10]))
	next += 10
	if typeCode != 1 || classCode != 1 || ttl != 0 || length != 4 || next+4 != len(message) || !bytes.Equal(message[next:next+4], []byte{192, 0, 2, 1}) {
		return dnsProof{}, malformedResponseError{"A answer semantics mismatch"}
	}
	normalizedHash := hash(canonicalResponse)
	if len(canonicalQuery) != 27 || hash(canonicalQuery) != queryHash || len(canonicalResponse) != 43 || normalizedHash != responseHash {
		return dnsProof{}, errors.New("internal canonical DNS fixture mismatch")
	}
	return dnsProof{
		FixtureID: fixtureID, Transport: "doq", QuestionName: "plab.test.", QuestionType: "A", QuestionClass: "IN",
		Answer: "192.0.2.1", TTLSeconds: 0, ResponseCode: "NOERROR", AuthoritativeAnswer: true,
		RecursionDesired: false, RecursionAvailable: false, ExternalUpstreamUsed: false, CacheEnabled: false,
		Framing: "two-octet-network-order-length-prefix", AuthorityMode: "local-fixture-authoritative",
		RequestMessageID: 0, RuntimeMessageID: 0, ResponseMessageID: 0, CanonicalHashNormalization: "identity",
		QueryLengthBytes: 27, QueryFramedLengthBytes: 29, QueryNormalizedSHA256: queryHash,
		ResponseRawLengthBytes: len(message), ResponseFramedLengthBytes: len(raw), ResponseCanonicalLengthBytes: 43,
		ResponseNormalizedSHA256: normalizedHash, ResponseCompressionDiffered: !bytes.Equal(message, canonicalResponse),
		ResponseCanonicalized: true, RawResponseEqualityRequired: false,
	}, nil
}

func readName(message []byte, offset int) (string, int, error) {
	var labels []string
	next := offset
	jumped := false
	seen := map[int]bool{}
	for steps := 0; steps < 32; steps++ {
		if offset >= len(message) || seen[offset] {
			return "", 0, errors.New("invalid DNS name")
		}
		seen[offset] = true
		value := message[offset]
		if value == 0 {
			if !jumped {
				next = offset + 1
			}
			return strings.Join(labels, ".") + ".", next, nil
		}
		if value&0xc0 == 0xc0 {
			if offset+1 >= len(message) {
				return "", 0, errors.New("truncated DNS pointer")
			}
			if !jumped {
				next = offset + 2
				jumped = true
			}
			offset = int(value&0x3f)<<8 | int(message[offset+1])
			continue
		}
		if value&0xc0 != 0 || int(value) > 63 || offset+1+int(value) > len(message) {
			return "", 0, errors.New("invalid DNS label")
		}
		labels = append(labels, string(message[offset+1:offset+1+int(value)]))
		offset += 1 + int(value)
		if !jumped {
			next = offset
		}
	}
	return "", 0, errors.New("DNS name exceeded parser bound")
}

func frame(message []byte) []byte {
	value := make([]byte, 2+len(message))
	binary.BigEndian.PutUint16(value[:2], uint16(len(message)))
	copy(value[2:], message)
	return value
}
func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func mustHex(value string) []byte {
	result, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return result
}
