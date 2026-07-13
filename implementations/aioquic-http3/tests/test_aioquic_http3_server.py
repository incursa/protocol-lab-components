import importlib.util
import secrets
import struct
import sys
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


if __name__ == "__main__":
    unittest.main()
