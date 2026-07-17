import importlib.util
from importlib.machinery import SourceFileLoader
import sys
import unittest
from pathlib import Path


MODULE_PATH = Path("/usr/local/bin/aioquic-http3-websocket-client")
if not MODULE_PATH.exists():
    MODULE_PATH = Path(__file__).parents[1] / "docker" / "aioquic_http3_websocket_client.py"
SPEC = importlib.util.spec_from_loader("rfc9220_client", SourceFileLoader("rfc9220_client", str(MODULE_PATH)))
CLIENT = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = CLIENT
SPEC.loader.exec_module(CLIENT)


class FrameContractTests(unittest.TestCase):
    def test_fragment_plan_reassembles_exact_payload(self):
        reader = CLIENT.WebSocketFrameReader()
        offset = 0
        frames = []
        for index, size in enumerate(CLIENT.FRAGMENT_BYTES):
            chunk = CLIENT.BINARY_PAYLOAD[offset : offset + size]
            offset += size
            encoded = CLIENT.write_frame(
                CLIENT.OPCODE_BINARY if index == 0 else CLIENT.OPCODE_CONTINUATION,
                chunk,
                masked=True,
                final=index == 2,
            )
            frames.extend(reader.feed(encoded))
        self.assertEqual([len(frame["payload"]) for frame in frames], [1024, 2048, 2928])
        self.assertEqual([frame["opcode"] for frame in frames], [CLIENT.OPCODE_BINARY, CLIENT.OPCODE_CONTINUATION, CLIENT.OPCODE_CONTINUATION])
        self.assertEqual([frame["fin"] for frame in frames], [False, False, True])
        self.assertTrue(all(frame["masked"] for frame in frames))
        self.assertEqual(b"".join(frame["payload"] for frame in frames), CLIENT.BINARY_PAYLOAD)
        self.assertEqual(CLIENT.sha256_hex(CLIENT.BINARY_PAYLOAD), CLIENT.BINARY_SHA256)

    def test_exact_six_scenarios(self):
        self.assertEqual(
            CLIENT.SUPPORTED_SCENARIOS,
            {
                "http3.websocket.rfc9220.extended-connect",
                "http3.websocket.rfc9220.control-frames",
                "http3.websocket.rfc9220.text-echo",
                "http3.websocket.rfc9220.binary-echo",
                "http3.websocket.rfc9220.close",
                "http3.websocket.rfc9220.fragmented-binary-echo",
            },
        )

    def test_runner_admission_identities_are_versioned_together(self):
        self.assertEqual(CLIENT.EXECUTOR_ID, "aioquic-rfc9220-websocket")
        self.assertEqual(CLIENT.EXECUTOR_VERSION, "0.3.1")
        self.assertEqual(CLIENT.LOAD_GENERATOR_ID, "aioquic-rfc9220-websocket-load")
        self.assertEqual(CLIENT.LOAD_GENERATOR_VERSION, "0.3.1")
        self.assertEqual(CLIENT.PARSER_ID, "protocol-lab-rfc9220-json")

    def test_sha256_admission_is_fail_closed(self):
        digest = "a" * 64
        self.assertEqual(CLIENT.require_sha256(digest, "test"), digest)
        with self.assertRaises(ValueError):
            CLIENT.require_sha256("not-a-digest", "test")

    def test_percentiles_use_nearest_rank(self):
        self.assertEqual(CLIENT.percentile([1, 2, 3, 4], 0.50), 2)
        self.assertEqual(CLIENT.percentile([1, 2, 3, 4], 0.99), 4)


if __name__ == "__main__":
    unittest.main()
