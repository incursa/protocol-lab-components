import argparse
import asyncio
import hashlib
import json
import mimetypes
import os
import struct
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import parse_qs, urlsplit

from aioquic.asyncio import serve
from aioquic.asyncio.protocol import QuicConnectionProtocol
from aioquic.h3.connection import H3Connection, H3_ALPN
from aioquic.h3.events import DataReceived, HeadersReceived
from aioquic.quic.configuration import QuicConfiguration


OPCODE_CONTINUATION = 0x0
OPCODE_TEXT = 0x1
OPCODE_BINARY = 0x2
OPCODE_CLOSE = 0x8
OPCODE_PING = 0x9
OPCODE_PONG = 0xA
AUTHORITY = b"websocket.plab.test"
PATH = "/websocket-proof"
TEXT_PAYLOAD = b"protocol-lab"
CONTROL_PAYLOAD = b"protocol-lab-ping"
BINARY_PAYLOAD = bytes([0xA5]) * 6000
BINARY_SHA256 = "8f8d8f75d55c80475ffb0c12b1ede7083d6df689e8ef04f05176c5050873bfb7"
FRAGMENT_BYTES = [1024, 2048, 2928]


def write_frame(opcode, payload=b"", *, final=True):
    first = (0x80 if final else 0) | opcode
    payload_length = len(payload)
    header = bytearray([first])
    if payload_length <= 125:
        header.append(payload_length)
    elif payload_length <= 0xFFFF:
        header.append(126)
        header.extend(struct.pack("!H", payload_length))
    else:
        header.append(127)
        header.extend(struct.pack("!Q", payload_length))
    return bytes(header) + payload


class WebSocketFrameReader:
    def __init__(self):
        self._buffer = bytearray()

    def feed(self, data):
        self._buffer.extend(data)
        frames = []
        while True:
            if len(self._buffer) < 2:
                return frames
            first, second = self._buffer[0], self._buffer[1]
            final = (first & 0x80) != 0
            rsv = first & 0x70
            opcode = first & 0x0F
            masked = (second & 0x80) != 0
            length = second & 0x7F
            offset = 2
            if rsv:
                raise RuntimeError("RSV-bearing client frame is not supported")
            if length == 126:
                if len(self._buffer) < offset + 2:
                    return frames
                length = struct.unpack("!H", self._buffer[offset : offset + 2])[0]
                offset += 2
            elif length == 127:
                if len(self._buffer) < offset + 8:
                    return frames
                length = struct.unpack("!Q", self._buffer[offset : offset + 8])[0]
                offset += 8
            masking_key = None
            if masked:
                if len(self._buffer) < offset + 4:
                    return frames
                masking_key = self._buffer[offset : offset + 4]
                offset += 4
            if len(self._buffer) < offset + length:
                return frames
            payload = bytes(self._buffer[offset : offset + length])
            del self._buffer[: offset + length]
            if masked:
                payload = bytes(value ^ masking_key[index % 4] for index, value in enumerate(payload))
            frames.append({"opcode": opcode, "fin": final, "masked": masked, "payload": payload})


class StaticHttp3ServerProtocol(QuicConnectionProtocol):
    def __init__(self, *args, www_root, **kwargs):
        super().__init__(*args, **kwargs)
        self._http = H3Connection(self._quic)
        self._www_root = Path(www_root).resolve()
        self._websocket_states = {}
        self._proof_log_counts = {}

    def quic_event_received(self, event):
        for http_event in self._http.handle_event(event):
            if isinstance(http_event, HeadersReceived):
                self._handle_headers(http_event.stream_id, http_event.headers)
            elif isinstance(http_event, DataReceived) and http_event.stream_id in self._websocket_states:
                self._handle_websocket_data(http_event.stream_id, http_event.data)

    def _handle_headers(self, stream_id, headers):
        request_headers = {name.lower(): value for name, value in headers}
        method = request_headers.get(b":method", b"")
        protocol = request_headers.get(b":protocol", b"")
        raw_path = request_headers.get(b":path", b"/").decode("utf-8", errors="replace")
        uri = urlsplit(raw_path)
        if method == b"CONNECT" and protocol == b"websocket" and uri.path == PATH:
            self._validate_websocket_headers(request_headers)
            self._start_websocket(stream_id)
            return
        if method != b"GET":
            self._send_response(stream_id, 405, b"method not allowed", [(b"content-type", b"text/plain")])
            return
        query = parse_qs(uri.query, keep_blank_values=True)
        status, body, response_headers = self._resolve_response(uri.path, query)
        self._send_response(stream_id, status, body, response_headers, split_payload_size=17 if uri.path == "/split-data.bin" else None)

    def _validate_websocket_headers(self, headers):
        expected = {b":scheme": b"https", b":authority": AUTHORITY, b"sec-websocket-version": b"13"}
        for name, value in expected.items():
            if headers.get(name) != value:
                raise RuntimeError(f"RFC 9220 request header mismatch: {name!r}")
        prohibited = {b"connection", b"upgrade", b"sec-websocket-key", b"sec-websocket-protocol", b"sec-websocket-extensions", b"origin"}
        present = sorted(name.decode("ascii") for name in prohibited if name in headers)
        if present:
            raise RuntimeError(f"prohibited WebSocket request headers present: {present!r}")

    def _start_websocket(self, stream_id):
        self._websocket_states[stream_id] = {"reader": WebSocketFrameReader(), "fragments": [], "fragmentSizes": []}
        self._http.send_headers(stream_id=stream_id, headers=[(b":status", b"200")], end_stream=False)
        self.transmit()
        self._log_proof("rfc9220-extended-connect-accepted", {"streamId": stream_id, "protocol": "h3", "alpn": "h3", "settingsEnableConnectProtocol": 1, "authority": AUTHORITY.decode(), "path": PATH, "responseStatus": 200})

    def _log_proof(self, event_name, payload):
        count = self._proof_log_counts.get(event_name, 0)
        self._proof_log_counts[event_name] = count + 1
        if count < 64:
            print(json.dumps({"eventName": event_name, **payload}), flush=True)

    def _handle_websocket_data(self, stream_id, data):
        state = self._websocket_states[stream_id]
        for frame in state["reader"].feed(data):
            if not frame["masked"]:
                raise RuntimeError("RFC 6455 client frame was not masked")
            opcode, payload, final = frame["opcode"], frame["payload"], frame["fin"]
            if state["fragments"] and opcode != OPCODE_CONTINUATION:
                raise RuntimeError("control or data frame interleaved with fragmented message")
            if opcode == OPCODE_CONTINUATION:
                if not state["fragments"]:
                    raise RuntimeError("continuation frame without fragmented message")
                state["fragments"].append(payload)
                state["fragmentSizes"].append(len(payload))
                if final:
                    reassembled = b"".join(state["fragments"])
                    if state["fragmentSizes"] != FRAGMENT_BYTES or reassembled != BINARY_PAYLOAD:
                        raise RuntimeError("fragmented binary reassembly mismatch")
                    self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_BINARY, reassembled), end_stream=False)
                    self._log_proof("rfc9220-fragmented-binary-reassembled", {"streamId": stream_id, "fragmentPayloadBytes": state["fragmentSizes"], "opcodes": ["binary", "continuation", "continuation"], "fin": [False, False, True], "interleavedControlFrames": False, "clientMaskObserved": True, "messageBytes": len(reassembled), "payloadSha256": hashlib.sha256(reassembled).hexdigest()})
                    state["fragments"] = []
                    state["fragmentSizes"] = []
            elif opcode == OPCODE_BINARY:
                if final:
                    if payload != BINARY_PAYLOAD:
                        raise RuntimeError("binary payload mismatch")
                    self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_BINARY, payload), end_stream=False)
                else:
                    if len(payload) != FRAGMENT_BYTES[0]:
                        raise RuntimeError("first fragmented binary payload size mismatch")
                    state["fragments"] = [payload]
                    state["fragmentSizes"] = [len(payload)]
            elif opcode == OPCODE_TEXT:
                if not final or payload != TEXT_PAYLOAD:
                    raise RuntimeError("text payload mismatch")
                self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_TEXT, payload), end_stream=False)
            elif opcode == OPCODE_PING:
                if not final or payload != CONTROL_PAYLOAD:
                    raise RuntimeError("ping payload mismatch")
                self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_PONG, payload), end_stream=False)
            elif opcode == OPCODE_CLOSE:
                if not final or payload != struct.pack("!H", 1000):
                    raise RuntimeError("close payload mismatch")
                self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_CLOSE, payload), end_stream=True)
                self._websocket_states.pop(stream_id, None)
                self._log_proof("rfc9220-websocket-clean-close", {"streamId": stream_id, "closeCode": 1000, "clientMaskObserved": True})
            else:
                raise RuntimeError(f"unsupported WebSocket opcode {opcode}")
        self.transmit()

    def _resolve_response(self, path, query):
        if path == "/status":
            body = json.dumps({
                "protocol": "h3",
                "server": "aioquic",
                "implementation": "aioquic-http3",
                "utc": datetime.now(timezone.utc).isoformat(),
                "processId": os.getpid(),
            }).encode("utf-8")
            return 200, body, [(b"content-type", b"application/json")]
        if path == "/headers/response":
            count = self._parse_positive_int(query.get("count", ["50"])[0], default=50, maximum=128)
            size = self._parse_positive_int(query.get("size", ["32"])[0], default=32, maximum=256)
            headers = [(b"content-type", b"text/plain")]
            value = ("x" * size).encode("ascii")
            headers.extend((f"x-protocol-bench-header-{index:03d}".encode("ascii"), value) for index in range(count))
            return 200, b"headers", headers
        if path == "/many-headers.txt":
            headers = [(b"content-type", b"text/plain")]
            headers.extend((f"x-incursa-header-{index:02d}".encode("ascii"), b"present") for index in range(64))
            return 200, b"many headers\n", headers
        if path == "/split-data.bin":
            return 200, bytes(index % 251 for index in range(4096)), [(b"content-type", b"application/octet-stream")]
        if path == "/bytes/1024":
            return 200, bytes(index % 251 for index in range(1024)), [(b"content-type", b"application/octet-stream")]
        relative = path.lstrip("/") or "index.html"
        candidate = (self._www_root / relative).resolve()
        if not str(candidate).startswith(str(self._www_root)) or not candidate.is_file():
            return 404, b"not found\n", [(b"content-type", b"text/plain")]
        content_type = mimetypes.guess_type(candidate.name)[0] or "application/octet-stream"
        return 200, candidate.read_bytes(), [(b"content-type", content_type.encode("ascii"))]

    @staticmethod
    def _parse_positive_int(value, *, default, maximum):
        try:
            parsed = int(value)
        except ValueError:
            return default
        return default if parsed < 0 else min(parsed, maximum)

    def _send_response(self, stream_id, status, body, headers, split_payload_size=None):
        response_headers = [(b":status", str(status).encode("ascii")), *headers, (b"content-length", str(len(body)).encode("ascii"))]
        self._http.send_headers(stream_id=stream_id, headers=response_headers)
        if not split_payload_size:
            self._http.send_data(stream_id=stream_id, data=body, end_stream=True)
        else:
            offset = 0
            while offset < len(body):
                chunk = body[offset : offset + split_payload_size]
                offset += len(chunk)
                self._http.send_data(stream_id=stream_id, data=chunk, end_stream=offset >= len(body))
        self.transmit()


async def main_async(args):
    configuration = QuicConfiguration(is_client=False, alpn_protocols=H3_ALPN)
    configuration.load_cert_chain(args.cert, args.key)
    await serve(args.host, args.port, configuration=configuration, create_protocol=lambda *protocol_args, **protocol_kwargs: StaticHttp3ServerProtocol(*protocol_args, www_root=args.www_root, **protocol_kwargs))
    print(json.dumps({"eventName": "ready", "implementationId": "aioquic-http3", "implementationVersion": "0.3.2", "implementationRole": "origin-server", "listenAddress": f"{args.host}:{args.port}", "protocol": "h3", "quicVersion": "QUICv1", "tlsVersion": "TLS 1.3", "alpn": "h3", "settingsEnableConnectProtocol": 1, "path": PATH, "binaryPayloadSha256": BINARY_SHA256}), flush=True)
    await asyncio.Event().wait()


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("www_root")
    parser.add_argument("cert")
    parser.add_argument("key")
    parser.add_argument("port", type=int)
    parser.add_argument("--host", default="0.0.0.0")
    return parser.parse_args()


def main():
    try:
        asyncio.run(main_async(parse_args()))
        return 0
    except Exception as exc:
        print(repr(exc), flush=True)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
