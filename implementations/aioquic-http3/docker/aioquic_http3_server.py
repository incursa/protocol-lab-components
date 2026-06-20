import argparse
import asyncio
import mimetypes
import os
import struct
import sys
from pathlib import Path
from urllib.parse import parse_qs, urlsplit

from aioquic.asyncio import serve
from aioquic.asyncio.protocol import QuicConnectionProtocol
from aioquic.h3.connection import H3Connection, H3_ALPN
from aioquic.h3.events import DataReceived, HeadersReceived
from aioquic.quic.configuration import QuicConfiguration


OPCODE_TEXT = 0x1
OPCODE_BINARY = 0x2
OPCODE_CLOSE = 0x8
OPCODE_PING = 0x9
OPCODE_PONG = 0xA


def write_frame(opcode, payload=b""):
    first = 0x80 | opcode
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

            first = self._buffer[0]
            second = self._buffer[1]
            final = (first & 0x80) != 0
            opcode = first & 0x0F
            masked = (second & 0x80) != 0
            length = second & 0x7F
            offset = 2
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
            if not final:
                raise RuntimeError("fragmented client frames are not expected in this proof")
            frames.append((opcode, payload))


class StaticHttp3ServerProtocol(QuicConnectionProtocol):
    def __init__(self, *args, www_root, **kwargs):
        super().__init__(*args, **kwargs)
        self._http = H3Connection(self._quic)
        self._www_root = Path(www_root).resolve()
        self._websocket_readers = {}

    def quic_event_received(self, event):
        for http_event in self._http.handle_event(event):
            if isinstance(http_event, HeadersReceived):
                self._handle_headers(http_event.stream_id, http_event.headers)
            elif isinstance(http_event, DataReceived) and http_event.stream_id in self._websocket_readers:
                self._handle_websocket_data(http_event.stream_id, http_event.data)

    def _handle_headers(self, stream_id, headers):
        request_headers = {name: value for name, value in headers}
        method = request_headers.get(b":method", b"").decode("ascii", errors="replace")
        protocol = request_headers.get(b":protocol", b"").decode("ascii", errors="replace")
        raw_path = request_headers.get(b":path", b"/").decode("utf-8", errors="replace")
        uri = urlsplit(raw_path)
        path = uri.path
        query = parse_qs(uri.query, keep_blank_values=True)

        if method == "CONNECT" and protocol == "websocket" and path == "/websocket-proof":
            self._start_websocket_proof(stream_id)
            return

        if method != "GET":
            self._send_response(stream_id, 405, b"method not allowed", [(b"content-type", b"text/plain")])
            return

        status, body, response_headers = self._resolve_response(path, query)
        self._send_response(stream_id, status, body, response_headers, split_payload_size=17 if path == "/split-data.bin" else None)

    def _resolve_response(self, path, query):
        if path == "/headers/response":
            count = self._parse_positive_int(query.get("count", ["50"])[0], default=50, maximum=128)
            size = self._parse_positive_int(query.get("size", ["32"])[0], default=32, maximum=256)
            headers = [(b"content-type", b"text/plain")]
            value = ("x" * size).encode("ascii")
            for index in range(count):
                headers.append((f"x-protocol-bench-header-{index:03d}".encode("ascii"), value))
            return 200, b"headers", headers

        if path == "/many-headers.txt":
            headers = [(b"content-type", b"text/plain")]
            for index in range(64):
                headers.append((f"x-incursa-header-{index:02d}".encode("ascii"), b"present"))
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

    def _parse_positive_int(self, value, *, default, maximum):
        try:
            parsed = int(value)
        except ValueError:
            return default
        if parsed < 0:
            return default
        return min(parsed, maximum)

    def _start_websocket_proof(self, stream_id):
        self._websocket_readers[stream_id] = WebSocketFrameReader()
        self._http.send_headers(
            stream_id=stream_id,
            headers=[
                (b":status", b"200"),
                (b"sec-websocket-protocol", b"proof.v1"),
            ],
            end_stream=False,
        )
        self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_PING, b"server-proof"), end_stream=False)
        self.transmit()

    def _handle_websocket_data(self, stream_id, data):
        reader = self._websocket_readers[stream_id]
        for opcode, payload in reader.feed(data):
            if opcode == OPCODE_PING:
                self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_PONG, payload), end_stream=False)
            elif opcode == OPCODE_PONG:
                continue
            elif opcode == OPCODE_TEXT:
                self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_TEXT, b"echo:" + payload), end_stream=False)
            elif opcode == OPCODE_BINARY:
                self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_BINARY, payload), end_stream=False)
            elif opcode == OPCODE_CLOSE:
                self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_CLOSE, payload), end_stream=True)
                self._websocket_readers.pop(stream_id, None)
            else:
                self._http.send_data(stream_id=stream_id, data=write_frame(OPCODE_CLOSE, struct.pack("!H", 1003)), end_stream=True)
                self._websocket_readers.pop(stream_id, None)
        self.transmit()

    def _send_response(self, stream_id, status, body, headers, split_payload_size=None):
        response_headers = [(b":status", str(status).encode("ascii"))]
        response_headers.extend(headers)
        response_headers.append((b"content-length", str(len(body)).encode("ascii")))
        self._http.send_headers(stream_id=stream_id, headers=response_headers)

        if split_payload_size is None or split_payload_size <= 0:
            self._http.send_data(stream_id=stream_id, data=body, end_stream=True)
        else:
            offset = 0
            while offset < len(body):
                chunk = body[offset : offset + split_payload_size]
                offset += len(chunk)
                self._http.send_data(stream_id=stream_id, data=chunk, end_stream=offset >= len(body))
            if not body:
                self._http.send_data(stream_id=stream_id, data=b"", end_stream=True)

        self.transmit()


async def main_async(args):
    configuration = QuicConfiguration(is_client=False, alpn_protocols=H3_ALPN)
    configuration.load_cert_chain(args.cert, args.key)

    await serve(
        args.host,
        args.port,
        configuration=configuration,
        create_protocol=lambda *protocol_args, **protocol_kwargs: StaticHttp3ServerProtocol(
            *protocol_args,
            www_root=args.www_root,
            **protocol_kwargs,
        ),
    )
    print(f"aioquic HTTP/3 server serving {args.www_root} on {args.host}:{args.port}", flush=True)
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
        print(repr(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
