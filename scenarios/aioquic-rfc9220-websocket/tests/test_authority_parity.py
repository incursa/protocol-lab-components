import argparse
import hashlib
import json
from pathlib import Path


EXPECTED = {
    "rfc9220-extended-connect.yaml": ("59640ee041d85d36ef484c915658e96f7772207637069e36e190cb6a74c4950a", "operation: establish", "messageType: none"),
    "rfc9220-control-frames.yaml": ("f08929cde090744524fb880805adbfd892d96f663ad080b3cf4c2b5f83c2d645", "operation: control-frames", "controlPayloadText: protocol-lab-ping"),
    "rfc9220-text-echo.yaml": ("9b1b3a9cb0639b37bfaa5319d9e4a4368cd074ed60673944e574317b3be6038c", "operation: text-echo", "payloadText: protocol-lab"),
    "rfc9220-binary-echo.yaml": ("167efd8e61b5b9d1a3d7d5bfee3fa8f38c7fabc6dfe94635d01f7ff640b1ef03", "operation: binary-echo", "payloadOctet: 165"),
    "rfc9220-close.yaml": ("15d2f7b1a0e00ab5bfee3aa890d6a351e6d06cb38db8bda74ace315b42225303", "operation: close", "closeCode: 1000"),
    "rfc9220-fragmented-binary-echo.yaml": ("76bb1c269d42b5ba53742bf5c69e8f2728427406946a7cf2802023f482959725", "operation: binary-echo", "framePayloadBytes: [1024, 2048, 2928]"),
}


def require(text, tokens, label):
    missing = [token for token in tokens if token not in text]
    if missing:
        raise SystemExit(f"{label} is missing semantic tokens: {missing}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--scenario-root", required=True)
    parser.add_argument("--executor-root")
    parser.add_argument("--target-root")
    args = parser.parse_args()
    scenario_root = Path(args.scenario_root).resolve()
    lock = json.loads((scenario_root / "authority-lock.json").read_text(encoding="utf-8"))
    if lock["authorityCommit"] != "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574" or len(lock["files"]) != 6:
        raise SystemExit("authority identity or six-file cardinality mismatch")
    observed = {}
    common = ["schemaVersion: \"2.0\"", "version: 2.0.0", "protocol: h3", "family: websocket", "binding: http3-extended-connect", "standard: rfc9220", "path: /websocket-proof", "authority: websocket.plab.test", "settingsEnableConnectProtocol: 1", "method: CONNECT", "protocol: websocket", "scheme: https", "secWebSocketVersion: \"13\"", "responseStatus: 200", "requested: websocket-over-h3", "expectedObserved: websocket-over-h3", "fallbackAllowed: false"]
    for name, (expected_hash, operation, detail) in EXPECTED.items():
        relative = f"scenarios/http3/websocket/{name}"
        data = (scenario_root / relative).read_bytes()
        digest = hashlib.sha256(data).hexdigest()
        if lock["files"].get(relative) != expected_hash or digest != expected_hash:
            raise SystemExit(f"authority hash mismatch: {relative}")
        text = data.decode("utf-8")
        require(text, common + [operation, detail], name)
        observed[relative] = digest
    if args.executor_root and args.target_root:
        executor = (Path(args.executor_root) / "docker/aioquic_http3_websocket_client.py").read_text(encoding="utf-8")
        target = (Path(args.target_root) / "docker/aioquic_http3_server.py").read_text(encoding="utf-8")
        ids = ["http3.websocket.rfc9220." + name.removeprefix("rfc9220-").removesuffix(".yaml") for name in EXPECTED]
        require(executor, ids + ['AUTHORITY = "websocket.plab.test"', 'PATH = "/websocket-proof"', 'TEXT_PAYLOAD = b"protocol-lab"', 'CONTROL_PAYLOAD = b"protocol-lab-ping"', 'bytes([0xA5]) * 6000', 'FRAGMENT_BYTES = [1024, 2048, 2928]'], "executor")
        require(target, ['AUTHORITY = b"websocket.plab.test"', 'PATH = "/websocket-proof"', 'TEXT_PAYLOAD = b"protocol-lab"', 'CONTROL_PAYLOAD = b"protocol-lab-ping"', 'bytes([0xA5]) * 6000', 'FRAGMENT_BYTES = [1024, 2048, 2928]', 'struct.pack("!H", 1000)'], "target")
    print(json.dumps({"authorityCommit": lock["authorityCommit"], "files": observed}, sort_keys=True))


if __name__ == "__main__":
    main()
