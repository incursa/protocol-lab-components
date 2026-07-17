import argparse
import asyncio
import hashlib
import json
import math
import os
import secrets
import ssl
import struct
import sys
import time
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import urlparse

from aioquic.asyncio import connect
from aioquic.asyncio.protocol import QuicConnectionProtocol
from aioquic.h3.connection import H3Connection, H3_ALPN, Setting
from aioquic.h3.events import DataReceived, HeadersReceived
from aioquic.quic.configuration import QuicConfiguration
from aioquic.quic.events import ConnectionTerminated
from aioquic.quic.logger import QuicFileLogger
from cryptography.hazmat.primitives import hashes, serialization


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
EXECUTOR_ID = "aioquic-rfc9220-websocket"
EXECUTOR_VERSION = "0.3.1"
LOAD_GENERATOR_ID = "aioquic-rfc9220-websocket-load"
LOAD_GENERATOR_VERSION = "0.3.1"
PARSER_ID = "protocol-lab-rfc9220-json"
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


def require_sha256(value, label):
    if len(value) != 64 or any(ch not in "0123456789abcdef" for ch in value.lower()):
        raise ValueError(f"{label} must be a lowercase SHA-256 digest")
    return value.lower()


def percentile(values, fraction):
    if not values:
        return 0.0
    ordered = sorted(values)
    index = max(0, min(len(ordered) - 1, math.ceil(fraction * len(ordered)) - 1))
    return ordered[index]


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


@dataclass
class StreamState:
    response_received: asyncio.Future
    frame_queue: asyncio.Queue = field(default_factory=asyncio.Queue)
    reader: WebSocketFrameReader = field(default_factory=WebSocketFrameReader)
    response_headers: list = field(default_factory=list)
    status: int | None = None
    sent_frames: list = field(default_factory=list)
    received_frames: list = field(default_factory=list)
    sent_frame_count: int = 0
    received_frame_count: int = 0
    client_mask_all: bool = True


class Http3WebSocketClientProtocol(QuicConnectionProtocol):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._http = H3Connection(self._quic)
        self._settings_received = asyncio.get_running_loop().create_future()
        self._streams = {}
        self.max_active_streams = 0

    @property
    def settings(self):
        return self._http.received_settings or {}

    @property
    def alpn(self):
        return self._quic.tls.alpn_negotiated

    @property
    def quic_version(self):
        return f"0x{self._quic._version:08x}"

    @property
    def peer_certificate(self):
        certificate = getattr(self._quic.tls, "_peer_certificate", None)
        if certificate is None:
            raise RuntimeError("authenticated peer certificate is unavailable")
        return certificate

    async def open_websocket(self, timeout):
        await asyncio.wait_for(asyncio.shield(self._settings_received), timeout=timeout)
        if self.settings.get(Setting.ENABLE_CONNECT_PROTOCOL) != 1:
            raise RuntimeError("peer did not advertise SETTINGS_ENABLE_CONNECT_PROTOCOL=1")
        stream_id = self._quic.get_next_available_stream_id()
        state = StreamState(asyncio.get_running_loop().create_future())
        self._streams[stream_id] = state
        self.max_active_streams = max(self.max_active_streams, len(self._streams))
        self._http.send_headers(
            stream_id=stream_id,
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
        await asyncio.wait_for(asyncio.shield(state.response_received), timeout=timeout)
        if state.status != 200:
            raise RuntimeError(f"unexpected WebSocket CONNECT status {state.status}")
        names = {name.lower() for name, _ in state.response_headers}
        prohibited = {b"connection", b"upgrade", b"sec-websocket-accept", b"sec-websocket-protocol", b"sec-websocket-extensions"}
        if names & prohibited:
            raise RuntimeError(f"prohibited response headers present: {sorted(names & prohibited)!r}")
        return stream_id

    def write_message(self, stream_id, opcode, payload=b"", *, final=True):
        state = self._streams[stream_id]
        encoded = write_frame(opcode, payload, masked=True, final=final)
        summary = {"opcode": opcode, "fin": final, "masked": True, "payloadBytes": len(payload), "payloadSha256": sha256_hex(payload)}
        state.sent_frame_count += 1
        state.client_mask_all = state.client_mask_all and summary["masked"]
        if len(state.sent_frames) < 8:
            state.sent_frames.append(summary)
        self._http.send_data(stream_id, encoded, end_stream=False)
        self.transmit()

    async def read_frame(self, stream_id, timeout):
        return await asyncio.wait_for(self._streams[stream_id].frame_queue.get(), timeout=timeout)

    def finish_stream(self, stream_id):
        return self._streams.pop(stream_id)

    def quic_event_received(self, event):
        if isinstance(event, ConnectionTerminated):
            for state in self._streams.values():
                if not state.response_received.done():
                    state.response_received.set_exception(RuntimeError(f"QUIC terminated: {event.reason_phrase}"))
        for http_event in self._http.handle_event(event):
            state = self._streams.get(http_event.stream_id)
            if state is None:
                continue
            if isinstance(http_event, HeadersReceived):
                state.response_headers.extend(http_event.headers)
                for name, value in http_event.headers:
                    if name == b":status":
                        state.status = int(value.decode("ascii"))
                if not state.response_received.done():
                    state.response_received.set_result(None)
            elif isinstance(http_event, DataReceived):
                for frame in state.reader.feed(http_event.data):
                    summary = {key: value for key, value in frame.items() if key != "payload"}
                    summary.update({"payloadBytes": len(frame["payload"]), "payloadSha256": sha256_hex(frame["payload"])})
                    state.received_frame_count += 1
                    if len(state.received_frames) < 8:
                        state.received_frames.append(summary)
                    state.frame_queue.put_nowait(frame)
        if self._http.received_settings is not None and not self._settings_received.done():
            self._settings_received.set_result(None)


async def expect_frame(protocol, stream_id, timeout, opcode, payload):
    frame = await protocol.read_frame(stream_id, timeout)
    if frame["masked"] or not frame["fin"] or frame["opcode"] != opcode or frame["payload"] != payload:
        raise RuntimeError(
            f"unexpected server frame opcode={frame['opcode']} fin={frame['fin']} masked={frame['masked']} bytes={len(frame['payload'])}"
        )


def scenario_payload(scenario_id):
    if scenario_id.endswith("control-frames"):
        return CONTROL_PAYLOAD, "control"
    if scenario_id.endswith("text-echo"):
        return TEXT_PAYLOAD, "text"
    if scenario_id.endswith("binary-echo"):
        return BINARY_PAYLOAD, "binary"
    return b"", "none"


async def message_operation(protocol, stream_id, scenario_id, timeout):
    payload, _ = scenario_payload(scenario_id)
    started = time.perf_counter()
    if scenario_id.endswith("control-frames"):
        protocol.write_message(stream_id, OPCODE_PING, payload)
        await expect_frame(protocol, stream_id, timeout, OPCODE_PONG, payload)
    elif scenario_id.endswith("text-echo"):
        protocol.write_message(stream_id, OPCODE_TEXT, payload)
        await expect_frame(protocol, stream_id, timeout, OPCODE_TEXT, payload)
    elif scenario_id.endswith("fragmented-binary-echo"):
        offset = 0
        for index, size in enumerate(FRAGMENT_BYTES):
            chunk = payload[offset : offset + size]
            offset += size
            protocol.write_message(stream_id, OPCODE_BINARY if index == 0 else OPCODE_CONTINUATION, chunk, final=index == len(FRAGMENT_BYTES) - 1)
        await expect_frame(protocol, stream_id, timeout, OPCODE_BINARY, payload)
    elif scenario_id.endswith("binary-echo"):
        protocol.write_message(stream_id, OPCODE_BINARY, payload)
        await expect_frame(protocol, stream_id, timeout, OPCODE_BINARY, payload)
    return (time.perf_counter() - started) * 1000.0


async def close_stream(protocol, stream_id, timeout):
    started = time.perf_counter()
    close_payload = struct.pack("!H", 1000)
    protocol.write_message(stream_id, OPCODE_CLOSE, close_payload)
    await expect_frame(protocol, stream_id, timeout, OPCODE_CLOSE, close_payload)
    state = protocol.finish_stream(stream_id)
    return (time.perf_counter() - started) * 1000.0, state


async def lifecycle_operation(protocol, scenario_id, timeout):
    started = time.perf_counter()
    stream_id = await protocol.open_websocket(timeout)
    connection_latency = (time.perf_counter() - started) * 1000.0
    close_latency, state = await close_stream(protocol, stream_id, timeout)
    operation_latency = close_latency if scenario_id.endswith("close") else connection_latency
    return operation_latency, connection_latency, close_latency, state


async def run_load(protocol, scenario_id, timeout, warmup, duration, concurrency):
    lifecycle = scenario_id.endswith("extended-connect") or scenario_id.endswith("close")
    captured_states = []
    operation_latencies = []
    connection_latencies = []
    close_latencies = []
    completed = 0
    failed = 0
    timed_out = 0
    sent_frame_count = 0
    received_frame_count = 0
    client_mask_all = True

    def record_state(state):
        nonlocal sent_frame_count, received_frame_count, client_mask_all
        sent_frame_count += state.sent_frame_count
        received_frame_count += state.received_frame_count
        client_mask_all = client_mask_all and state.client_mask_all
        if len(captured_states) < 16:
            captured_states.append(state)

    if lifecycle:
        await asyncio.gather(*(lifecycle_operation(protocol, scenario_id, timeout) for _ in range(concurrency)))
        await asyncio.sleep(max(0.0, warmup))
        deadline = time.perf_counter() + duration

        async def lifecycle_worker():
            nonlocal completed, failed, timed_out
            while time.perf_counter() < deadline:
                try:
                    op, connection, close, state = await lifecycle_operation(protocol, scenario_id, timeout)
                    operation_latencies.append(op)
                    connection_latencies.append(connection)
                    close_latencies.append(close)
                    record_state(state)
                    completed += 1
                except asyncio.TimeoutError:
                    timed_out += 1
                    raise
                except Exception:
                    failed += 1
                    raise

        measured_started = time.perf_counter()
        await asyncio.gather(*(lifecycle_worker() for _ in range(concurrency)))
    else:
        async def open_measured_stream():
            started = time.perf_counter()
            stream_id = await protocol.open_websocket(timeout)
            connection_latencies.append((time.perf_counter() - started) * 1000.0)
            return stream_id

        streams = await asyncio.gather(*(open_measured_stream() for _ in range(concurrency)))
        await asyncio.gather(*(message_operation(protocol, stream_id, scenario_id, timeout) for stream_id in streams))
        await asyncio.sleep(max(0.0, warmup))
        deadline = time.perf_counter() + duration

        async def message_worker(stream_id):
            nonlocal completed, failed, timed_out
            while time.perf_counter() < deadline:
                try:
                    operation_latencies.append(await message_operation(protocol, stream_id, scenario_id, timeout))
                    completed += 1
                except asyncio.TimeoutError:
                    timed_out += 1
                    raise
                except Exception:
                    failed += 1
                    raise

        measured_started = time.perf_counter()
        await asyncio.gather(*(message_worker(stream_id) for stream_id in streams))
        for stream_id in streams:
            close, state = await close_stream(protocol, stream_id, timeout)
            close_latencies.append(close)
            record_state(state)

    measured_seconds = time.perf_counter() - measured_started
    if completed <= 0 or failed or timed_out:
        raise RuntimeError(f"load completion mismatch completed={completed} failed={failed} timedOut={timed_out}")
    return {
        "states": captured_states,
        "operationLatencies": operation_latencies,
        "connectionLatencies": connection_latencies,
        "closeLatencies": close_latencies,
        "completed": completed,
        "failed": failed,
        "timedOut": timed_out,
        "measuredSeconds": measured_seconds,
        "sentFrameCount": sent_frame_count,
        "receivedFrameCount": received_frame_count,
        "clientMaskAll": client_mask_all,
    }


def certificate_proof(protocol):
    certificate = protocol.peer_certificate
    der = certificate.public_bytes(serialization.Encoding.DER)
    spki = certificate.public_key().public_bytes(serialization.Encoding.DER, serialization.PublicFormat.SubjectPublicKeyInfo)
    return der, {
        "authenticated": True,
        "serverName": AUTHORITY,
        "leafCertificateSha256": sha256_hex(der),
        "leafSpkiSha256": sha256_hex(spki),
        "signatureHashAlgorithm": certificate.signature_hash_algorithm.name,
    }


def build_result(protocol, args, load, peer_der):
    fragmented = args.scenario_id.endswith("fragmented-binary-echo")
    payload, message_type = scenario_payload(args.scenario_id)
    payload_hash = sha256_hex(payload)
    states = load["states"]
    sent_frames = [frame for state in states for frame in state.sent_frames]
    received_frames = [frame for state in states for frame in state.received_frames]
    if not sent_frames or not load["clientMaskAll"] or not all(frame["masked"] for frame in sent_frames):
        raise RuntimeError("client masking proof mismatch")
    data_frames = [frame for frame in sent_frames if frame["opcode"] in (OPCODE_CONTINUATION, OPCODE_TEXT, OPCODE_BINARY)]
    if fragmented:
        first_plan = data_frames[:3]
        if [frame["payloadBytes"] for frame in first_plan] != FRAGMENT_BYTES or [frame["opcode"] for frame in first_plan] != [OPCODE_BINARY, OPCODE_CONTINUATION, OPCODE_CONTINUATION] or [frame["fin"] for frame in first_plan] != [False, False, True]:
            raise RuntimeError("fragment plan proof mismatch")
        if payload_hash != BINARY_SHA256:
            raise RuntimeError("fragmented payload hash mismatch")
    der, certificate = certificate_proof(protocol)
    if der != peer_der:
        raise RuntimeError("peer certificate changed during result construction")
    if protocol.alpn != "h3" or protocol.quic_version != "0x00000001":
        raise RuntimeError(f"exact QUICv1/h3 negotiation mismatch quic={protocol.quic_version} alpn={protocol.alpn}")
    active_streams = protocol.max_active_streams
    if active_streams != args.concurrency:
        raise RuntimeError(f"active stream count mismatch expected={args.concurrency} observed={active_streams}")
    total_bytes = len(payload) * 2 * load["completed"]
    seconds = max(load["measuredSeconds"], 0.000001)
    operation_latencies = load["operationLatencies"]
    connection_latencies = load["connectionLatencies"] or operation_latencies
    close_latencies = load["closeLatencies"]
    metrics = {
        "connectionsPerSecond": load["completed"] / seconds if args.scenario_id.endswith(("extended-connect", "close")) else 1.0 / seconds,
        "messagesPerSecond": load["completed"] / seconds,
        "bytesPerSecond": total_bytes / seconds,
        "connectionLatencyMean": sum(connection_latencies) / len(connection_latencies),
        "connectionLatencyP50": percentile(connection_latencies, 0.50),
        "connectionLatencyP75": percentile(connection_latencies, 0.75),
        "connectionLatencyP90": percentile(connection_latencies, 0.90),
        "connectionLatencyP95": percentile(connection_latencies, 0.95),
        "connectionLatencyP99": percentile(connection_latencies, 0.99),
        "messageLatencyMean": sum(operation_latencies) / len(operation_latencies),
        "messageLatencyP50": percentile(operation_latencies, 0.50),
        "messageLatencyP75": percentile(operation_latencies, 0.75),
        "messageLatencyP90": percentile(operation_latencies, 0.90),
        "messageLatencyP95": percentile(operation_latencies, 0.95),
        "messageLatencyP99": percentile(operation_latencies, 0.99),
        "closeLatencyP50": percentile(close_latencies, 0.50),
        "closeLatencyP95": percentile(close_latencies, 0.95),
        "closeLatencyP99": percentile(close_latencies, 0.99),
        "completedOperations": load["completed"],
        "failedOperations": load["failed"],
        "timedOutOperations": load["timedOut"],
        "totalTransferredBytes": total_bytes,
        "effectiveConcurrency": args.concurrency,
        "effectiveStreams": active_streams,
    }
    requested = {
        "connections": 1,
        "concurrency": args.concurrency,
        "streamsPerConnection": args.concurrency,
        "warmupSeconds": args.warmup,
        "durationSeconds": args.duration,
        "cooldownSeconds": args.cooldown,
        "timeoutSeconds": args.timeout,
        "repetition": 1,
    }
    protocol_proof = {
        "requestedProtocol": "websocket-over-h3",
        "observedProtocol": "websocket-over-h3",
        "protocol": "h3",
        "protocolVersion": "HTTP/3",
        "protocolVariant": "websocket-h3-extended-connect",
        "fallbackDetected": False,
        "noFallback": True,
        "quicVersion": protocol.quic_version,
        "tlsVersion": "TLS 1.3",
        "alpn": protocol.alpn,
        "certificate": certificate,
        "settingsEnableConnectProtocol": protocol.settings.get(Setting.ENABLE_CONNECT_PROTOCOL),
        "requestPseudoHeaders": {":method": "CONNECT", ":protocol": "websocket", ":scheme": "https", ":authority": AUTHORITY, ":path": PATH},
        "secWebSocketVersion": "13",
        "prohibitedRequestHeadersPresent": False,
        "responseStatus": 200,
        "secWebSocketAcceptPresent": False,
        "secWebSocketProtocolPresent": False,
        "secWebSocketExtensionsPresent": False,
        "clientMaskObserved": True,
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
    materialization = {
        "scenarioPackageSha256": require_sha256(args.scenario_package_sha256, "scenario package digest"),
        "executorPackageSha256": require_sha256(args.executor_package_sha256, "executor package digest"),
        "targetPackageSha256": require_sha256(args.target_package_sha256, "target package digest"),
        "executorImageId": args.executor_image_id,
        "targetImageId": args.target_image_id,
        "immutable": args.executor_image_id.startswith("sha256:") and args.target_image_id.startswith("sha256:"),
    }
    if not materialization["immutable"]:
        raise RuntimeError("executor and target image identities must be immutable sha256 IDs")
    return {
        "schemaVersion": "protocol-lab.rfc9220-executor-result.v1",
        "executorId": EXECUTOR_ID,
        "executorVersion": EXECUTOR_VERSION,
        "loadGeneratorId": LOAD_GENERATOR_ID,
        "loadGeneratorVersion": LOAD_GENERATOR_VERSION,
        "parserId": PARSER_ID,
        "scenarioId": args.scenario_id,
        "loadProfileId": args.load_profile_id,
        "status": "passed",
        "passed": True,
        "implementationRole": args.implementation_role,
        "authorityCommit": "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574",
        "requestedLoad": requested,
        "effectiveLoad": {**requested, "activeConnections": 1, "activeStreams": active_streams, "measuredDurationSeconds": load["measuredSeconds"]},
        "protocolProof": protocol_proof,
        "validation": {"status": "passed", "passed": True, "zeroUnexpectedFailures": True, "zeroTimeouts": True},
        "metrics": metrics,
        "messageType": message_type,
        "payloadBytes": len(payload),
        "payloadSha256": payload_hash,
        "frameSummary": {"sentSamples": sent_frames, "receivedSamples": received_frames, "sentFrameCount": load["sentFrameCount"], "receivedFrameCount": load["receivedFrameCount"], "clientMaskObservedForAllSentFrames": load["clientMaskAll"]},
        "materialization": materialization,
        "artifacts": [
            "validation.json", "protocol-proof.json", "websocket-summary.json", "payload-hash.json", "frame-summary.json",
            "tls-negotiation.json", "quic-summary.json", "materialization-proof.json", "executor-identity.json",
            "load-generator-identity.json", "parser-identity.json", "tls-peer-certificate.der", "client-result.json",
            "result.json", "load.stdout.log", "load.stderr.log",
        ],
        "warnings": [],
    }


def write_artifacts(output_path, result, peer_der):
    output = Path(output_path)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    proof = result["protocolProof"]
    metrics = result["metrics"]
    artifacts = {
        "validation.json": {"schemaVersion": "protocol-lab.validation.v1", "scenarioId": result["scenarioId"], "status": "passed", "checks": ["quic:v1", "tls:1.3", "alpn:h3", "authenticated-certificate", "no-fallback", "settings-enable-connect-protocol", "extended-connect", "pseudo-headers", "client-masking", "payload-match", "close-code:1000", "zero-unexpected-failures", "zero-timeouts", "materialization-digests"]},
        "protocol-proof.json": proof,
        "websocket-summary.json": {"scenarioId": result["scenarioId"], "messageType": result["messageType"], "payloadBytes": result["payloadBytes"], "payloadSha256": result["payloadSha256"], **metrics},
        "payload-hash.json": {"algorithm": "sha256", "payloadBytes": result["payloadBytes"], "expected": result["payloadSha256"], "observed": result["payloadSha256"]},
        "frame-summary.json": result["frameSummary"],
        "tls-negotiation.json": {"tlsVersion": proof["tlsVersion"], "alpn": proof["alpn"], "certificate": proof["certificate"]},
        "quic-summary.json": {"quicVersion": proof["quicVersion"], "activeConnections": 1, "activeStreams": result["effectiveLoad"]["activeStreams"]},
        "materialization-proof.json": result["materialization"],
        "executor-identity.json": {"executorId": EXECUTOR_ID, "executorVersion": EXECUTOR_VERSION, "role": "test-executor"},
        "load-generator-identity.json": {"loadGeneratorId": LOAD_GENERATOR_ID, "loadGeneratorVersion": LOAD_GENERATOR_VERSION},
        "parser-identity.json": {"parserId": PARSER_ID, "parserVersion": "0.3.1"},
    }
    for name, payload in artifacts.items():
        (output.parent / name).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
    (output.parent / "tls-peer-certificate.der").write_bytes(peer_der)


async def main_async(args):
    parsed = urlparse(args.url)
    if parsed.scheme != "https":
        raise ValueError("URL must use https")
    if args.scenario_id.endswith("fragmented-binary-echo"):
        expected = ("diagnostic", 8, 1.0, 10.0, 1.0, 10.0)
    else:
        expected = ("websocket-smoke", 1, 1.0, 5.0, 0.0, 5.0)
    observed = (args.load_profile_id, args.concurrency, args.warmup, args.duration, args.cooldown, args.timeout)
    if observed != expected:
        raise ValueError(f"exact load profile mismatch expected={expected!r} observed={observed!r}")
    configuration = QuicConfiguration(is_client=True, alpn_protocols=H3_ALPN, verify_mode=ssl.CERT_REQUIRED)
    configuration.server_name = AUTHORITY
    configuration.load_verify_locations(cafile=args.ca_certificate)
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
            load = await run_load(protocol, args.scenario_id, args.timeout, args.warmup, args.duration, args.concurrency)
            peer_der, _ = certificate_proof(protocol)
            result = build_result(protocol, args, load, peer_der)
    finally:
        if secrets_log_file is not None:
            secrets_log_file.close()
    if args.cooldown:
        await asyncio.sleep(args.cooldown)
    write_artifacts(args.output, result, peer_der)
    print(json.dumps(result, separators=(",", ":"), sort_keys=True))


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("url")
    parser.add_argument("output")
    parser.add_argument("--scenario-id", required=True, choices=sorted(SUPPORTED_SCENARIOS))
    parser.add_argument("--load-profile-id", required=True)
    parser.add_argument("--concurrency", type=int, required=True)
    parser.add_argument("--warmup", type=float, required=True)
    parser.add_argument("--duration", type=float, required=True)
    parser.add_argument("--cooldown", type=float, required=True)
    parser.add_argument("--timeout", type=float, required=True)
    parser.add_argument("--ca-certificate", default="/certs/root.pem")
    parser.add_argument("--scenario-package-sha256", required=True)
    parser.add_argument("--executor-package-sha256", required=True)
    parser.add_argument("--target-package-sha256", required=True)
    parser.add_argument("--executor-image-id", required=True)
    parser.add_argument("--target-image-id", required=True)
    parser.add_argument("--implementation-role", required=True, choices=["origin-server", "proxy"])
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
