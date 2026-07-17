import importlib.util
import json
import secrets
import struct
import sys
import tempfile
import unittest
from importlib.machinery import SourceFileLoader
from pathlib import Path


MODULE_PATH = Path("/usr/local/bin/aioquic-http3-server")
if not MODULE_PATH.exists():
    MODULE_PATH = Path(__file__).parents[1] / "docker" / "aioquic_http3_server.py"
SPEC = importlib.util.spec_from_loader("rfc9220_server", SourceFileLoader("rfc9220_server", str(MODULE_PATH)))
SERVER = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = SERVER
SPEC.loader.exec_module(SERVER)


def masked_frame(opcode, payload, final):
    first = (0x80 if final else 0) | opcode
    if len(payload) <= 125:
        header = bytearray([first, 0x80 | len(payload)])
    else:
        header = bytearray([first, 0x80 | 126])
        header.extend(struct.pack("!H", len(payload)))
    key = secrets.token_bytes(4)
    return bytes(header) + key + bytes(value ^ key[index % 4] for index, value in enumerate(payload))


class ServerFrameContractTests(unittest.TestCase):
    def test_fragmented_client_frames_are_masked_and_exact(self):
        reader = SERVER.WebSocketFrameReader()
        offset = 0
        frames = []
        for index, size in enumerate(SERVER.FRAGMENT_BYTES):
            chunk = SERVER.BINARY_PAYLOAD[offset : offset + size]
            offset += size
            frames.extend(reader.feed(masked_frame(SERVER.OPCODE_BINARY if index == 0 else SERVER.OPCODE_CONTINUATION, chunk, index == 2)))
        self.assertEqual([len(frame["payload"]) for frame in frames], [1024, 2048, 2928])
        self.assertEqual([frame["opcode"] for frame in frames], [SERVER.OPCODE_BINARY, SERVER.OPCODE_CONTINUATION, SERVER.OPCODE_CONTINUATION])
        self.assertEqual([frame["fin"] for frame in frames], [False, False, True])
        self.assertTrue(all(frame["masked"] for frame in frames))
        self.assertEqual(b"".join(frame["payload"] for frame in frames), SERVER.BINARY_PAYLOAD)


class Http3OriginRegressionTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        root = Path(self.temporary.name)
        (root / "status").write_bytes(b"aioquic HTTP/3 status\n")
        self.server = object.__new__(SERVER.StaticHttp3ServerProtocol)
        self.server._www_root = root.resolve()

    def tearDown(self):
        self.temporary.cleanup()

    def test_status_identity_is_canonical_json(self):
        status, body, headers = self.server._resolve_response("/status", {})
        self.assertEqual(status, 200)
        document = json.loads(body)
        self.assertEqual(document["protocol"], "h3")
        self.assertEqual(document["server"], "aioquic")
        self.assertEqual(document["implementation"], "aioquic-http3")
        self.assertIsInstance(document["processId"], int)
        self.assertIn("utc", document)
        self.assertIn((b"content-type", b"application/json"), headers)

    def test_plaintext_and_json_core_contracts_are_exact(self):
        status, body, headers = self.server._resolve_response("/plaintext", {})
        self.assertEqual(status, 200)
        self.assertEqual(body, b"Hello, World!")
        self.assertIn((b"content-type", b"text/plain"), headers)

        status, body, headers = self.server._resolve_response("/json", {})
        self.assertEqual(status, 200)
        self.assertEqual(json.loads(body), {"message": "Hello, World!"})
        self.assertIn((b"content-type", b"application/json"), headers)

    def test_one_kibibyte_payload_remains_deterministic(self):
        status, body, headers = self.server._resolve_response("/bytes/1024", {})
        self.assertEqual(status, 200)
        self.assertEqual(len(body), 1024)
        self.assertEqual(body, bytes(index % 251 for index in range(1024)))
        self.assertIn((b"content-type", b"application/octet-stream"), headers)

    def test_sixty_four_kibibyte_payload_is_deterministic(self):
        status, body, headers = self.server._resolve_response("/bytes/65536", {})
        self.assertEqual(status, 200)
        self.assertEqual(len(body), 65536)
        self.assertEqual(body, bytes(index % 251 for index in range(65536)))
        self.assertIn((b"content-type", b"application/octet-stream"), headers)

    def test_header_workload_remains_exact(self):
        status, body, headers = self.server._resolve_response("/headers/response", {"count": ["50"], "size": ["32"]})
        self.assertEqual(status, 200)
        self.assertEqual(body, b"headers")
        workload_headers = [(name, value) for name, value in headers if name.startswith(b"x-protocol-bench-header-")]
        self.assertEqual(len(workload_headers), 50)
        self.assertTrue(all(len(value) == 32 for _, value in workload_headers))

    def test_unknown_path_remains_fail_closed_404(self):
        status, body, _ = self.server._resolve_response("/missing", {})
        self.assertEqual(status, 404)
        self.assertEqual(body, b"not found\n")


if __name__ == "__main__":
    unittest.main()
