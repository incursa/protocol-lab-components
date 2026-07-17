#!/usr/bin/env python3
import asyncio
import base64
import hashlib
import json
import struct


OPCODE_CONTINUATION = 0x0
OPCODE_TEXT = 0x1
OPCODE_BINARY = 0x2
OPCODE_CLOSE = 0x8
OPCODE_PING = 0x9
OPCODE_PONG = 0xA
PATH = "/websocket-proof"
TEXT_PAYLOAD = b"protocol-lab"
CONTROL_PAYLOAD = b"protocol-lab-ping"
BINARY_PAYLOAD = bytes([0xA5]) * 6000
BINARY_SHA256 = "8f8d8f75d55c80475ffb0c12b1ede7083d6df689e8ef04f05176c5050873bfb7"
FRAGMENT_BYTES = [1024, 2048, 2928]
GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
event_counts = {}
connection_sequence = 0


def emit(event_name, **values):
    count = event_counts.get(event_name, 0)
    event_counts[event_name] = count + 1
    if count < 64:
        print(json.dumps({"eventName": event_name, **values}, separators=(",", ":")), flush=True)


def frame(opcode, payload=b"", *, final=True):
    first = (0x80 if final else 0) | opcode
    if len(payload) <= 125:
        return bytes([first, len(payload)]) + payload
    if len(payload) <= 0xFFFF:
        return bytes([first, 126]) + struct.pack("!H", len(payload)) + payload
    return bytes([first, 127]) + struct.pack("!Q", len(payload)) + payload


async def read_exact_frame(reader):
    first, second = await reader.readexactly(2)
    final = bool(first & 0x80)
    if first & 0x70:
        raise RuntimeError("RSV-bearing client frame is not supported")
    opcode = first & 0x0F
    masked = bool(second & 0x80)
    length = second & 0x7F
    if length == 126:
        length = struct.unpack("!H", await reader.readexactly(2))[0]
    elif length == 127:
        length = struct.unpack("!Q", await reader.readexactly(8))[0]
    if not masked:
        raise RuntimeError("RFC 6455 client frame was not masked")
    masking_key = await reader.readexactly(4)
    payload = await reader.readexactly(length)
    payload = bytes(value ^ masking_key[index % 4] for index, value in enumerate(payload))
    return opcode, payload, final


async def read_request(reader):
    raw = await reader.readuntil(b"\r\n\r\n")
    lines = raw.decode("iso-8859-1").split("\r\n")
    method, path, version = lines[0].split(" ", 2)
    headers = {}
    for line in lines[1:]:
        if not line:
            continue
        name, value = line.split(":", 1)
        headers[name.strip().lower()] = value.strip()
    return method, path, version, headers


async def handle(reader, writer):
    global connection_sequence
    connection_sequence += 1
    stream_id = connection_sequence
    fragments = []
    fragment_sizes = []
    try:
        method, path, version, headers = await read_request(reader)
        if method != "GET" or path != PATH or version != "HTTP/1.1":
            raise RuntimeError("nghttpx backend request was not an HTTP/1.1 WebSocket upgrade")
        if headers.get("upgrade", "").lower() != "websocket" or "upgrade" not in headers.get("connection", "").lower():
            raise RuntimeError("nghttpx did not translate RFC 9220 to an HTTP/1.1 WebSocket upgrade")
        key = headers.get("sec-websocket-key")
        if not key or headers.get("sec-websocket-version") != "13":
            raise RuntimeError("WebSocket upgrade headers were incomplete")
        accept = base64.b64encode(hashlib.sha1((key + GUID).encode("ascii")).digest()).decode("ascii")
        writer.write((
            "HTTP/1.1 101 Switching Protocols\r\n"
            "Upgrade: websocket\r\n"
            "Connection: Upgrade\r\n"
            f"Sec-WebSocket-Accept: {accept}\r\n\r\n"
        ).encode("ascii"))
        await writer.drain()
        emit("rfc9220-extended-connect-accepted", streamId=stream_id, protocol="h3", alpn="h3", settingsEnableConnectProtocol=1, authority="websocket.plab.test", path=PATH, responseStatus=200)

        while True:
            opcode, payload, final = await read_exact_frame(reader)
            if fragments and opcode != OPCODE_CONTINUATION:
                raise RuntimeError("control or data frame interleaved with fragmented message")
            if opcode == OPCODE_CONTINUATION:
                if not fragments:
                    raise RuntimeError("continuation frame without fragmented message")
                fragments.append(payload)
                fragment_sizes.append(len(payload))
                if final:
                    reassembled = b"".join(fragments)
                    if fragment_sizes != FRAGMENT_BYTES or reassembled != BINARY_PAYLOAD:
                        raise RuntimeError("fragmented binary reassembly mismatch")
                    writer.write(frame(OPCODE_BINARY, reassembled))
                    await writer.drain()
                    emit("rfc9220-fragmented-binary-reassembled", streamId=stream_id, fragmentPayloadBytes=fragment_sizes, opcodes=["binary", "continuation", "continuation"], fin=[False, False, True], interleavedControlFrames=False, clientMaskObserved=True, messageBytes=len(reassembled), payloadSha256=hashlib.sha256(reassembled).hexdigest())
                    fragments = []
                    fragment_sizes = []
            elif opcode == OPCODE_BINARY:
                if final:
                    if payload != BINARY_PAYLOAD:
                        raise RuntimeError("binary payload mismatch")
                    writer.write(frame(OPCODE_BINARY, payload))
                    await writer.drain()
                else:
                    if len(payload) != FRAGMENT_BYTES[0]:
                        raise RuntimeError("first fragmented binary payload size mismatch")
                    fragments = [payload]
                    fragment_sizes = [len(payload)]
            elif opcode == OPCODE_TEXT:
                if not final or payload != TEXT_PAYLOAD:
                    raise RuntimeError("text payload mismatch")
                writer.write(frame(OPCODE_TEXT, payload))
                await writer.drain()
            elif opcode == OPCODE_PING:
                if not final or payload != CONTROL_PAYLOAD:
                    raise RuntimeError("ping payload mismatch")
                writer.write(frame(OPCODE_PONG, payload))
                await writer.drain()
            elif opcode == OPCODE_CLOSE:
                if not final or payload != struct.pack("!H", 1000):
                    raise RuntimeError("close payload mismatch")
                writer.write(frame(OPCODE_CLOSE, payload))
                await writer.drain()
                emit("rfc9220-websocket-clean-close", streamId=stream_id, closeCode=1000, clientMaskObserved=True)
                return
            else:
                raise RuntimeError(f"unsupported WebSocket opcode {opcode}")
    except asyncio.IncompleteReadError:
        pass
    except Exception as exc:
        emit("websocket-backend-error", streamId=stream_id, error=repr(exc))
        raise
    finally:
        writer.close()
        await writer.wait_closed()


async def main():
    server = await asyncio.start_server(handle, "127.0.0.1", 8080)
    emit("ready", implementationId="nghttpx-rfc9220-gateway", implementationVersion="0.1.1", implementationRole="proxy", runtimeComponent="nghttpx", listenAddress="0.0.0.0:4433", protocol="h3", quicVersion="QUICv1", tlsVersion="TLS 1.3", alpn="h3", settingsEnableConnectProtocol=1, path=PATH, binaryPayloadSha256=BINARY_SHA256)
    async with server:
        await server.serve_forever()


if __name__ == "__main__":
    asyncio.run(main())
