import argparse
import asyncio
import json
import os
from dataclasses import dataclass

from aioquic.asyncio import serve
from aioquic.asyncio.protocol import QuicConnectionProtocol
from aioquic.quic.configuration import QuicConfiguration
from aioquic.quic.events import StreamDataReceived


IMPLEMENTATION_ID = "aioquic-raw"
PACKAGE_ID = "org.protocol-lab.components.implementation.aioquic-raw"
ALPN = "plab-raw-quic"
DEFAULT_PORT = 5451
DEFAULT_ECHO_MAX = 64 * 1024
MAX_READ_BYTES = 64 * 1024 * 1024
DOWNLOAD_MAGIC = b"PLAB-DL1"
SUPPORTED_SCENARIOS = [
    "quic.transport.handshake-cold",
    "quic.transport.latency.echo-1kb",
    "quic.transport.stream-throughput.1mb",
    "quic.transport.multiplex.100x64kb",
    "quic.transport.duplex-streams",
]


def echo_max_for_scenario(scenario):
    if scenario in {"quic.transport.handshake-cold", "quic.transport.stream-throughput.1mb"}:
        return 0
    if scenario == "quic.transport.latency.echo-1kb":
        return 1024
    return DEFAULT_ECHO_MAX


def parse_download_request(payload):
    if len(payload) != len(DOWNLOAD_MAGIC) + 8 or not payload.startswith(DOWNLOAD_MAGIC):
        return None
    length = int.from_bytes(payload[len(DOWNLOAD_MAGIC) :], "big")
    return length if 0 < length <= MAX_READ_BYTES else None


def deterministic_payload(length):
    return bytes(index % 251 for index in range(length))


@dataclass
class StreamState:
    payload: bytearray


class RawQuicProtocol(QuicConnectionProtocol):
    def __init__(self, *args, echo_max, **kwargs):
        super().__init__(*args, **kwargs)
        self._echo_max = echo_max
        self._streams = {}

    def quic_event_received(self, event):
        if not isinstance(event, StreamDataReceived):
            return

        state = self._streams.setdefault(event.stream_id, StreamState(payload=bytearray()))
        if len(state.payload) + len(event.data) > MAX_READ_BYTES:
            self._quic.reset_stream(event.stream_id, error_code=0x504C4142)
            self._streams.pop(event.stream_id, None)
            self.transmit()
            return

        state.payload.extend(event.data)
        if not event.end_stream:
            return

        payload = bytes(state.payload)
        self._streams.pop(event.stream_id, None)
        download_length = parse_download_request(payload)
        if download_length is not None:
            response = deterministic_payload(download_length)
        elif payload and len(payload) <= self._echo_max:
            response = payload
        else:
            response = b""
        self._quic.send_stream_data(event.stream_id, response, end_stream=True)
        self.transmit()


def self_test():
    request = DOWNLOAD_MAGIC + (1024 * 1024).to_bytes(8, "big")
    assert parse_download_request(request) == 1024 * 1024
    assert parse_download_request(request + b"x") is None
    assert deterministic_payload(502) == bytes(list(range(251)) * 2)
    assert echo_max_for_scenario("quic.transport.handshake-cold") == 0
    assert echo_max_for_scenario("quic.transport.latency.echo-1kb") == 1024
    assert echo_max_for_scenario("quic.transport.multiplex.100x64kb") == 64 * 1024
    print("aioquic raw adapter self-test passed", flush=True)


async def run_server(args):
    scenario = os.getenv("PLAB_SCENARIO_ID", "")
    echo_max = echo_max_for_scenario(scenario)
    configuration = QuicConfiguration(
        is_client=False,
        alpn_protocols=[ALPN],
        max_data=128 * 1024 * 1024,
        max_stream_data=64 * 1024 * 1024,
    )
    configuration.load_cert_chain(args.cert, args.key)
    await serve(
        args.host,
        args.port,
        configuration=configuration,
        create_protocol=lambda *protocol_args, **protocol_kwargs: RawQuicProtocol(
            *protocol_args, echo_max=echo_max, **protocol_kwargs
        ),
    )
    print(
        json.dumps(
            {
                "status": "ready",
                "implementationId": IMPLEMENTATION_ID,
                "packageId": PACKAGE_ID,
                "protocol": "quic",
                "alpn": ALPN,
                "listen": f"{args.host}:{args.port}",
                "advertiseHost": os.getenv("PROTOCOL_LAB_TARGET_ADVERTISE_HOST", "").strip(),
                "aioquicVersion": "1.3.0",
                "processId": os.getpid(),
                "supportedScenarios": SUPPORTED_SCENARIOS,
            },
            separators=(",", ":"),
        ),
        flush=True,
    )
    await asyncio.Event().wait()


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default=os.getenv("PROTOCOL_LAB_TARGET_BIND_ADDRESS", "0.0.0.0"))
    parser.add_argument("--port", type=int, default=int(os.getenv("PLAB_QUIC_PORT", str(DEFAULT_PORT))))
    parser.add_argument("--cert", default=os.getenv("PLAB_CERT_FILE", "certs/leaf.pem"))
    parser.add_argument("--key", default=os.getenv("PLAB_KEY_FILE", "certs/leaf-key.pem"))
    parser.add_argument("--self-test", action="store_true")
    return parser.parse_args()


def main():
    args = parse_args()
    if args.self_test:
        self_test()
        return 0
    asyncio.run(run_server(args))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
