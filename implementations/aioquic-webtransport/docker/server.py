import argparse
import asyncio
import hashlib
import json
from urllib.parse import urlsplit

from aioquic.asyncio import serve
from aioquic.asyncio.protocol import QuicConnectionProtocol
from aioquic.h3.connection import H3Connection, H3_ALPN
from aioquic.h3.events import HeadersReceived, WebTransportStreamDataReceived
from aioquic.quic.configuration import QuicConfiguration

IMPLEMENTATION_ID = "aioquic-webtransport"
IMPLEMENTATION_VERSION = "0.1.2"
UPSTREAM_VERSION = "1.3.0"
AUTHORITY = b"webtransport.plab.test"
PATH = "/webtransport/echo"
WEBTRANSPORT_PROTOCOLS = {b"webtransport", b"webtransport-h3"}
SETTINGS_WT_ENABLED = 0x2C7CF000
PAYLOAD_BYTES = 65536
PAYLOAD_SHA256 = "4b640d85ab3ba30fd02c9fc9db4a8928f416322ad27022ea58a65aaee68a4df2"


def emit(event_name, **values):
    print(json.dumps({"eventName": event_name, **values}, sort_keys=True), flush=True)


class StandardsTrackH3Connection(H3Connection):
    """Advertise current SETTINGS_WT_ENABLED alongside aioquic's legacy setting."""

    def _get_local_settings(self):
        settings = super()._get_local_settings()
        if self._enable_webtransport:
            settings[SETTINGS_WT_ENABLED] = 1
        return settings


class WebTransportEchoProtocol(QuicConnectionProtocol):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._http = StandardsTrackH3Connection(self._quic, enable_webtransport=True)
        self._sessions = set()
        self._streams = {}

    def quic_event_received(self, event):
        for http_event in self._http.handle_event(event):
            if isinstance(http_event, HeadersReceived):
                self._handle_headers(http_event)
            elif isinstance(http_event, WebTransportStreamDataReceived):
                self._handle_stream(http_event)

    def _handle_headers(self, event):
        headers = {name.lower(): value for name, value in event.headers}
        method = headers.get(b":method", b"")
        protocol = headers.get(b":protocol", b"")
        authority = headers.get(b":authority", b"")
        path = urlsplit(headers.get(b":path", b"/").decode("utf-8", errors="replace")).path
        authority_host = authority.split(b":", 1)[0]
        if method != b"CONNECT" or protocol not in WEBTRANSPORT_PROTOCOLS or authority_host != AUTHORITY or path != PATH:
            self._http.send_headers(stream_id=event.stream_id, headers=[(b":status", b"404")], end_stream=True)
            self.transmit()
            emit("webtransport-session-rejected", streamId=event.stream_id, method=method.decode(errors="replace"), protocol=protocol.decode(errors="replace"), authority=authority.decode(errors="replace"), path=path)
            return
        self._sessions.add(event.stream_id)
        self._http.send_headers(
            stream_id=event.stream_id,
            headers=[(b":status", b"200"), (b"sec-webtransport-http3-draft", b"draft02")],
            end_stream=False,
        )
        self.transmit()
        emit("webtransport-session-accepted", implementationId=IMPLEMENTATION_ID, sessionId=event.stream_id, protocol="webtransport-over-h3", alpn="h3", authority=authority.decode(), path=path)

    def _handle_stream(self, event):
        if event.session_id not in self._sessions:
            self._quic.close(error_code=0x102, reason_phrase="unknown WebTransport session")
            self.transmit()
            return
        data = self._streams.setdefault(event.stream_id, bytearray())
        data.extend(event.data)
        if len(data) > PAYLOAD_BYTES:
            self._quic.close(error_code=0x102, reason_phrase="payload too large")
            self.transmit()
            emit("webtransport-stream-invalid", streamId=event.stream_id, bytes=len(data), reason="payload-too-large")
            return
        if event.stream_ended:
            payload = bytes(data)
            digest = hashlib.sha256(payload).hexdigest()
            if len(payload) != PAYLOAD_BYTES or digest != PAYLOAD_SHA256:
                self._quic.close(error_code=0x102, reason_phrase="payload mismatch")
                self.transmit()
                emit("webtransport-stream-invalid", streamId=event.stream_id, bytes=len(payload), sha256=digest, reason="payload-mismatch")
                return
            self._quic.send_stream_data(event.stream_id, payload, end_stream=True)
            self.transmit()
            self._streams.pop(event.stream_id, None)
            emit("webtransport-stream-echoed", implementationId=IMPLEMENTATION_ID, sessionId=event.session_id, streamId=event.stream_id, streamDirection="client-initiated-bidirectional", streamCount=1, bytes=len(payload), sha256=digest)


async def main_async(args):
    configuration = QuicConfiguration(is_client=False, alpn_protocols=H3_ALPN, max_datagram_frame_size=65536)
    configuration.load_cert_chain(args.cert, args.key)
    await serve(args.host, args.port, configuration=configuration, create_protocol=WebTransportEchoProtocol)
    emit("ready", implementationId=IMPLEMENTATION_ID, implementationVersion=IMPLEMENTATION_VERSION, upstreamVersion=UPSTREAM_VERSION, listenAddress=f"{args.host}:{args.port}", protocol="webtransport-over-h3", alpn="h3", tlsVersion="TLS 1.3", path=PATH, payloadBytes=PAYLOAD_BYTES, payloadSha256=PAYLOAD_SHA256)
    await asyncio.Event().wait()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="0.0.0.0")
    parser.add_argument("--port", type=int, default=4433)
    parser.add_argument("--cert", default="/certs/leaf.pem")
    parser.add_argument("--key", default="/certs/leaf-key.pem")
    parser.add_argument("--version", action="store_true")
    parser.add_argument("runner_entrypoint", nargs="?", choices=["run.sh"], help=argparse.SUPPRESS)
    args = parser.parse_args()
    if args.version:
        print(f"{IMPLEMENTATION_ID} {IMPLEMENTATION_VERSION} aioquic {UPSTREAM_VERSION}")
        return 0
    asyncio.run(main_async(args))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
