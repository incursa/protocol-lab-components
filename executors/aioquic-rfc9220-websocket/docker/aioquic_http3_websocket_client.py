import argparse
import asyncio
import hashlib
import json
import os
import secrets
import ssl
import struct
import sys
import time
from pathlib import Path
from urllib.parse import urlparse

from aioquic.asyncio import connect
from aioquic.asyncio.protocol import QuicConnectionProtocol
from aioquic.h3.connection import H3Connection, H3_ALPN, Setting
from aioquic.h3.events import DataReceived, HeadersReceived
from aioquic.quic.configuration import QuicConfiguration
from aioquic.quic.events import ConnectionTerminated
from aioquic.quic.logger import QuicFileLogger


OPCODE_CONTINUATION = 0x0
OPCODE_TEXT = 0x1
OPCODE_BINARY = 0x2
OPCODE_CLOSE = 0x8
OPCODE_PING = 0x9
OPCODE_PONG = 0xA
AUTHORITY = "websocket.plab.test"
PATH = "/websocket-proof"
TEXT_PAYLOAD = b"protocol-lab"
CONTROL_PAYLOAD = b"protocol-lab-ping"
BINARY_PAYLOAD = bytes([0xA5]) * 6000
BINARY_SHA256 = "8f8d8f75d55c80475ffb0c12b1ede7083d6df689e8ef04f05176c5050873bfb7"
FRAGMENT_BYTES = [1024, 2048, 2928]
SUPPORTED_SCENARIOS = {
    "http3.websocket.rfc9220.extended-connect",
    "http3.websocket.rfc9220.control-frames",
    "http3.websocket.rfc9220.text-echo",
    "http3.websocket.rfc9220.binary-echo",
    "http3.websocket.rfc9220.close",
    "http3.websocket.rfc9220.fragmented-binary-echo",
}


def sha256_hex(payload):
    return hashlib.sha256(payload).hexdigest()


def write_frame(opcode, payload=b"", *, masked=False, final=True):
    first = (0x80 if final else 0) | opcode
    payload_length = len(payload)
    header = bytearray([first])
    mask_bit = 0x80 if masked else 0
    if payload_length <= 125:
        header.append(mask_bit | payload_length)
    elif payload_length <= 0xFFFF:
        header.append(mask_bit | 126)
        header.extend(struct.pack("!H", payload_length))
    else:
        header.append(mask_bit | 127)
        header.extend(struct.pack("!Q", payload_length))
    if not masked:
        return bytes(header) + payload
    masking_key = secrets.token_bytes(4)
    masked_payload = bytes(value ^ masking_key[index % 4] for index, value in enumerate(payload))
    return bytes(header) + masking_key + masked_payload


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
                raise RuntimeError("RSV-bearing server frame is not allowed")
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


class Http3WebSocketClientProtocol(QuicConnectionProtocol):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._http = H3Connection(self._quic)
        self._settings_received = asyncio.get_running_loop().create_future()
        self._response_received = asyncio.get_running_loop().create_future()
        self._frame_queue = asyncio.Queue()
        self._reader = WebSocketFrameReader()
        self._stream_id = None
        self._response_headers = []
        self._status = None
        self.sent_frames = []
        self.received_frames = []

    @property
    def status(self):
        return self._status

    @property
    def response_headers(self):
        return self._response_headers

    @property
    def settings(self):
        return self._http.received_settings or {}

    @property
    def alpn(self):
        return self._quic.tls.alpn_negotiated

    @property
    def quic_version(self):
        return f"0x{self._quic._version:08x}"

    async def open_websocket(self, timeout):
        await asyncio.wait_for(self._settings_received, timeout=timeout)
        if self.settings.get(Setting.ENABLE_CONNECT_PROTOCOL) != 1:
            raise RuntimeError("peer did not advertise SETTINGS_ENABLE_CONNECT_PROTOCOL=1")
        self._stream_id = self._quic.get_next_available_stream_id()
        self._http.send_headers(
            stream_id=self._stream_id,
            headers=[
                (b":method", b"CONNECT"),
                (b":protocol", b"websocket"),
                (b":scheme", b"https"),
                (b":authority", AUTHORITY.encode("ascii")),
                (b":path", PATH.encode("ascii")),
                (b"sec-websocket-version", b"13"),
            ],
            end_stream=False,
        )
        self.transmit()
        await asyncio.wait_for(self._response_received, timeout=timeout)
        if self._status != 200:
            raise RuntimeError(f"unexpected WebSocket CONNECT status {self._status}")
        names = {name.lower() for name, _ in self._response_headers}
        prohibited = {b"connection", b"upgrade", b"sec-websocket-accept", b"sec-websocket-protocol", b"sec-websocket-extensions"}
        if names & prohibited:
            raise RuntimeError(f"prohibited response headers present: {sorted(names & prohibited)!r}")

    def write_message(self, opcode, payload=b"", *, final=True):
        if self._stream_id is None:
            raise RuntimeError("WebSocket stream is not open")
        encoded = write_frame(opcode, payload, masked=True, final=final)
        self.sent_frames.append({"opcode": opcode, "fin": final, "masked": True, "payloadBytes": len(payload), "payloadSha256": sha256_hex(payload)})
        self._http.send_data(self._stream_id, encoded, end_stream=False)
        self.transmit()

    async def read_frame(self, timeout):
        return await asyncio.wait_for(self._frame_queue.get(), timeout=timeout)

    def quic_event_received(self, event):
        if isinstance(event, ConnectionTerminated) and not self._response_received.done():
            self._response_received.set_exception(RuntimeError(f"QUIC terminated: {event.reason_phrase}"))
        for http_event in self._http.handle_event(event):
            if isinstance(http_event, HeadersReceived) and http_event.stream_id == self._stream_id:
                self._response_headers.extend(http_event.headers)
                for name, value in http_event.headers:
                    if name == b":status":
                        self._status = int(value.decode("ascii"))
                if not self._response_received.done():
                    self._response_received.set_result(None)
            elif isinstance(http_event, DataReceived) and http_event.stream_id == self._stream_id:
                for frame in self._reader.feed(http_event.data):
                    summary = {key: value for key, value in frame.items() if key != "payload"}
                    summary.update({"payloadBytes": len(frame["payload"]), "payloadSha256": sha256_hex(frame["payload"])})
                    self.received_frames.append(summary)
                    self._frame_queue.put_nowait(frame)
        if self._http.received_settings is not None and not self._settings_received.done():
            self._settings_received.set_result(None)


async def expect_frame(protocol, timeout, opcode, payload):
    frame = await protocol.read_frame(timeout)
    if frame["masked"] or not frame["fin"] or frame["opcode"] != opcode or frame["payload"] != payload:
        raise RuntimeError(
            f"unexpected server frame opcode={frame['opcode']} fin={frame['fin']} masked={frame['masked']} bytes={len(frame['payload'])}"
        )
    return frame


async def run_proof(protocol, scenario_id, timeout, url):
    connection_started = time.perf_counter()
    await protocol.open_websocket(timeout)
    connection_latency_ms = (time.perf_counter() - connection_started) * 1000
    operation_started = time.perf_counter()
    payload = b""
    message_type = "none"
    fragmented = scenario_id.endswith("fragmented-binary-echo")
    if scenario_id.endswith("control-frames"):
        payload = CONTROL_PAYLOAD
        protocol.write_message(OPCODE_PING, payload)
        await expect_frame(protocol, timeout, OPCODE_PONG, payload)
        message_type = "control"
    elif scenario_id.endswith("text-echo"):
        payload = TEXT_PAYLOAD
        protocol.write_message(OPCODE_TEXT, payload)
        await expect_frame(protocol, timeout, OPCODE_TEXT, payload)
        message_type = "text"
    elif scenario_id.endswith("binary-echo") and not fragmented:
        payload = BINARY_PAYLOAD
        protocol.write_message(OPCODE_BINARY, payload)
        await expect_frame(protocol, timeout, OPCODE_BINARY, payload)
        message_type = "binary"
    elif fragmented:
        payload = BINARY_PAYLOAD
        offset = 0
        for index, size in enumerate(FRAGMENT_BYTES):
            chunk = payload[offset : offset + size]
            offset += size
            protocol.write_message(OPCODE_BINARY if index == 0 else OPCODE_CONTINUATION, chunk, final=index == len(FRAGMENT_BYTES) - 1)
        await expect_frame(protocol, timeout, OPCODE_BINARY, payload)
        message_type = "binary"
    operation_latency_ms = (time.perf_counter() - operation_started) * 1000

    close_started = time.perf_counter()
    close_payload = struct.pack("!H", 1000)
    protocol.write_message(OPCODE_CLOSE, close_payload)
    await expect_frame(protocol, timeout, OPCODE_CLOSE, close_payload)
    close_latency_ms = (time.perf_counter() - close_started) * 1000

    payload_hash = sha256_hex(payload)
    data_frames = [frame for frame in protocol.sent_frames if frame["opcode"] in (OPCODE_CONTINUATION, OPCODE_TEXT, OPCODE_BINARY)]
    if fragmented:
        observed_plan = [frame["payloadBytes"] for frame in data_frames]
        observed_opcodes = [frame["opcode"] for frame in data_frames]
        observed_fins = [frame["fin"] for frame in data_frames]
        if observed_plan != FRAGMENT_BYTES or observed_opcodes != [OPCODE_BINARY, OPCODE_CONTINUATION, OPCODE_CONTINUATION] or observed_fins != [False, False, True]:
            raise RuntimeError("fragment plan proof mismatch")
        if payload_hash != BINARY_SHA256:
            raise RuntimeError("fragmented payload hash mismatch")

    protocol_proof = {
        "requestedProtocol": "websocket-over-h3",
        "observedProtocol": "websocket-over-h3",
        "protocol": "h3",
        "protocolVersion": "HTTP/3",
        "protocolVariant": "websocket-h3-extended-connect",
        "noFallback": True,
        "quicVersion": protocol.quic_version,
        "tlsVersion": "TLS 1.3",
        "alpn": protocol.alpn,
        "settingsEnableConnectProtocol": protocol.settings.get(Setting.ENABLE_CONNECT_PROTOCOL),
        "requestPseudoHeaders": {":method": "CONNECT", ":protocol": "websocket", ":scheme": "https", ":authority": AUTHORITY, ":path": PATH},
        "secWebSocketVersion": "13",
        "prohibitedRequestHeadersPresent": False,
        "responseStatus": protocol.status,
        "secWebSocketAcceptPresent": False,
        "secWebSocketProtocolPresent": False,
        "secWebSocketExtensionsPresent": False,
        "clientMaskObserved": all(frame["masked"] for frame in protocol.sent_frames),
        "fragmentedMessage": fragmented,
        "fragmentPayloadBytes": FRAGMENT_BYTES if fragmented else [],
        "fragmentOpcodes": ["binary", "continuation", "continuation"] if fragmented else [],
        "fragmentFin": [False, False, True] if fragmented else [],
        "interleavedControlFrames": False,
        "reassembledPayloadBytes": len(payload),
        "reassembledPayloadSha256": payload_hash,
        "closeSent": 1000,
        "closeReceived": 1000,
        "cleanCompletion": True,
    }
    metrics = {
        "connectionsPerSecond": 1000 / max(connection_latency_ms, 0.001),
        "connectionLatencyMean": connection_latency_ms,
        "connectionLatencyP50": connection_latency_ms,
        "connectionLatencyP75": connection_latency_ms,
        "connectionLatencyP90": connection_latency_ms,
        "connectionLatencyP95": connection_latency_ms,
        "connectionLatencyP99": connection_latency_ms,
        "messageLatencyMean": operation_latency_ms,
        "messageLatencyP50": operation_latency_ms,
        "messageLatencyP75": operation_latency_ms,
        "messageLatencyP90": operation_latency_ms,
        "messageLatencyP95": operation_latency_ms,
        "messageLatencyP99": operation_latency_ms,
        "controlFrameLatencyP50": operation_latency_ms,
        "controlFrameLatencyP95": operation_latency_ms,
        "controlFrameLatencyP99": operation_latency_ms,
        "closeLatencyP50": close_latency_ms,
        "closeLatencyP95": close_latency_ms,
        "closeLatencyP99": close_latency_ms,
        "completedOperations": 1,
        "failedOperations": 0,
        "timedOutOperations": 0,
        "totalTransferredBytes": len(payload) * 2,
        "effectiveStreams": 1,
    }
    return {
        "schemaVersion": "protocol-lab.aioquic-rfc9220-result.v2",
        "status": "passed",
        "evidenceClass": "local-package-diagnostic",
        "scenarioId": scenario_id,
        "url": url,
        "statusCode": protocol.status,
        "messageType": message_type,
        "payloadBytes": len(payload),
        "payloadSha256": payload_hash,
        "protocolProof": protocol_proof,
        "frameSummary": {"sent": protocol.sent_frames, "received": protocol.received_frames, "dataFrames": data_frames},
        "metrics": metrics,
        "responseHeaders": [{"name": name.decode("ascii", errors="replace"), "value": value.decode("utf-8", errors="replace")} for name, value in protocol.response_headers],
        "warnings": ["Local package-backed RFC 9220 evidence is diagnostic and non-publishable."],
    }


def write_artifacts(output_path, result):
    output = Path(output_path)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    artifacts = {
        "validation.json": {"schemaVersion": "protocol-lab.validation.v1", "scenarioId": result["scenarioId"], "status": "passed", "checks": ["protocol:h3", "no-fallback", "extended-connect", "client-masking", "payload-match", "close-code:1000", "zero-unexpected-failures", "zero-timeouts"]},
        "protocol-proof.json": result["protocolProof"],
        "websocket-summary.json": {"scenarioId": result["scenarioId"], "messageType": result["messageType"], "payloadBytes": result["payloadBytes"], "payloadSha256": result["payloadSha256"], "completedOperations": 1, "failedOperations": 0, "timedOutOperations": 0},
        "payload-hash.json": {"algorithm": "sha256", "payloadBytes": result["payloadBytes"], "expected": result["payloadSha256"], "observed": result["payloadSha256"]},
        "frame-summary.json": result["frameSummary"],
    }
    for name, payload in artifacts.items():
        (output.parent / name).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


async def main_async(args):
    parsed = urlparse(args.url)
    if parsed.scheme != "https":
        raise ValueError("URL must use https")
    configuration = QuicConfiguration(is_client=True, alpn_protocols=H3_ALPN, verify_mode=ssl.CERT_NONE)
    qlog_dir = os.environ.get("QLOGDIR")
    if qlog_dir:
        os.makedirs(qlog_dir, exist_ok=True)
        configuration.quic_logger = QuicFileLogger(qlog_dir)
    secrets_log_file = None
    sslkeylogfile = os.environ.get("SSLKEYLOGFILE")
    if sslkeylogfile:
        os.makedirs(os.path.dirname(sslkeylogfile), exist_ok=True)
        secrets_log_file = open(sslkeylogfile, "a", encoding="utf-8")
        configuration.secrets_log_file = secrets_log_file
    try:
        async with connect(parsed.hostname, parsed.port or 443, configuration=configuration, create_protocol=Http3WebSocketClientProtocol) as protocol:
            result = await run_proof(protocol, args.scenario_id, args.timeout, args.url)
    finally:
        if secrets_log_file is not None:
            secrets_log_file.close()
    if result["protocolProof"]["alpn"] != "h3":
        raise RuntimeError(f"exact h3 ALPN was not negotiated: {result['protocolProof']['alpn']!r}")
    write_artifacts(args.output, result)
    print(f"scenarioId={args.scenario_id} completed=1 failed=0 timedOut=0 output={args.output}")


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("url")
    parser.add_argument("output")
    parser.add_argument("--scenario-id", required=True, choices=sorted(SUPPORTED_SCENARIOS))
    parser.add_argument("--timeout", type=float, default=20.0)
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
