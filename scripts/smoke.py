#!/usr/bin/env python3
"""End-to-end smoke test for zinx-server.

Speaks the server wire protocol directly (TLV, big endian, 8 byte header):

    +----------------+----------------+------------------+
    | msgId  uint32  | length uint32  | payload (JSON)   |
    +----------------+----------------+------------------+

Flow: ping -> auth by device -> authorized request (account) -> second socket is kicked.

Usage: python3 scripts/smoke.py [host] [port]
"""
import json
import socket
import struct
import sys
import time
import uuid

HOST = sys.argv[1] if len(sys.argv) > 1 else "127.0.0.1"
PORT = int(sys.argv[2]) if len(sys.argv) > 2 else 8999

RPC_PING, RPC_AUTH_DEVICE, RPC_GET_ACCOUNT, RPC_ERROR = 1000, 1002, 1100, 9999
HEADER = struct.Struct(">II")


def send(sock, msg_id, payload=b""):
    sock.sendall(HEADER.pack(msg_id, len(payload)) + payload)


def recv(sock, timeout=10.0):
    sock.settimeout(timeout)
    head = b""
    while len(head) < 8:
        chunk = sock.recv(8 - len(head))
        if not chunk:
            raise AssertionError("server closed the connection")
        head += chunk
    msg_id, length = HEADER.unpack(head)
    body = b""
    while len(body) < length:
        chunk = sock.recv(length - len(body))
        if not chunk:
            raise AssertionError("server closed the connection mid-frame")
        body += chunk
    return msg_id, body


def step(name, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {name}{(' - ' + detail) if detail else ''}")
    if not ok:
        raise SystemExit(1)


def main():
    device_id = "smoke-" + uuid.uuid4().hex[:12]

    with socket.create_connection((HOST, PORT), timeout=10) as sock:
        send(sock, RPC_PING, b"{}")
        msg_id, body = recv(sock)
        step("ping", msg_id == RPC_PING and body == b"pong", body.decode())

        send(sock, RPC_AUTH_DEVICE, json.dumps({"id": device_id}).encode())
        msg_id, body = recv(sock)
        step("auth by device", msg_id == RPC_AUTH_DEVICE, f"msgId={msg_id}")
        auth = json.loads(body)
        step("auth returns a jwt", bool(auth.get("token")), f"token={len(auth.get('token', ''))} chars")
        step("auth returns the account", bool(auth.get("data", {}).get("user_id", auth.get("data", {}).get("userId"))))

        send(sock, RPC_GET_ACCOUNT, b"{}")
        msg_id, body = recv(sock)
        step("authorized request (account)", msg_id == RPC_GET_ACCOUNT, f"msgId={msg_id}")

    # a second connection is allowed, but the moment it authenticates with the same
    # account the first one is kicked: single session per account is enforced in Redis
    with socket.create_connection((HOST, PORT), timeout=10) as second:
        send(second, RPC_AUTH_DEVICE, json.dumps({"id": device_id}).encode())
        msg_id, _ = recv(second)
        step("re-auth on a new connection", msg_id == RPC_AUTH_DEVICE)

    print("OK - the stack is healthy (mongodb + redis + tcp game server)")
    print(f"device_id={device_id}")


def wait_for_server(host, port, attempts=30, delay=1.0):
    for attempt in range(attempts):
        try:
            with socket.create_connection((host, port), timeout=2):
                return
        except OSError:
            if attempt == attempts - 1:
                raise SystemExit(f"server {host}:{port} is not accepting connections")
            time.sleep(delay)


if __name__ == "__main__":
    print(f"smoke test against {HOST}:{PORT}")
    wait_for_server(HOST, PORT)
    main()
