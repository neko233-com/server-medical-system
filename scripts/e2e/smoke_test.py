# -*- coding: utf-8 -*-
"""端到端冒烟测试：登录 / RBAC / 运营分析 / 院内即时通讯(WebSocket)。

用法：先启动服务（PORT 默认 6970），再运行本脚本。
"""
import base64
import json
import os
import socket
import struct
import time
import urllib.error
import urllib.parse
import urllib.request

BASE = os.environ.get("BASE", "http://127.0.0.1:6970")
HOST = "127.0.0.1"
PORT = int(os.environ.get("PORT", "6970"))


# ---------------------------------------------------------------- HTTP utils
def req(method, path, token=None, data=None, raw=None, ctype="application/json"):
    url = BASE + path
    headers = {}
    body = None
    if token:
        headers["Authorization"] = "Bearer " + token
    if raw is not None:
        body = raw
        headers["Content-Type"] = ctype
    elif data is not None:
        body = json.dumps(data).encode()
        headers["Content-Type"] = "application/json"
    r = urllib.request.Request(url, data=body, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=15) as resp:
            return resp.status, resp.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()


def multipart_post(path, token, filename, content, fields=None):
    """Build a multipart/form-data request (mirrors the browser FormData upload)."""
    boundary = "----smokeboundary1234"
    body = b""
    for k, v in (fields or {}).items():
        body += '--%s\r\nContent-Disposition: form-data; name="%s"\r\n\r\n%s\r\n' % (boundary, k, v)
    body += (
        '--%s\r\nContent-Disposition: form-data; name="file"; filename="%s"\r\n'
        'Content-Type: image/png\r\n\r\n' % (boundary, filename)
    ).encode()
    body += content + ("\r\n--%s--\r\n" % boundary).encode()
    r = urllib.request.Request(
        BASE + path, data=body, method="POST",
        headers={"Authorization": "Bearer " + token,
                 "Content-Type": "multipart/form-data; boundary=" + boundary},
    )
    try:
        with urllib.request.urlopen(r, timeout=20) as resp:
            return resp.status, resp.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()


def login(user, pw):
    s, b = req("POST", "/api/auth/login", data={"username": user, "password": pw})
    if s != 200:
        raise SystemExit("login %s failed: %s %s" % (user, s, b))
    return json.loads(b)["token"]


fails = []


def check(name, cond, extra=""):
    print(("PASS " if cond else "FAIL ") + name, extra)
    if not cond:
        fails.append(name)


# ------------------------------------------------------- Minimal WS client
# 聊天走 WebSocket，标准库没有 WS 客户端，这里实现一个够用的最小版：
# 握手 + 掩码文本帧收发，足以验证实时投递语义。
class WS:
    def __init__(self, token):
        self.sock = socket.create_connection((HOST, PORT), timeout=10)
        key = base64.b64encode(os.urandom(16)).decode()
        path = "/api/chat/ws?token=" + urllib.parse.quote(token)
        handshake = (
            "GET %s HTTP/1.1\r\nHost: %s:%d\r\nUpgrade: websocket\r\n"
            "Connection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n"
            % (path, HOST, PORT, key)
        )
        self.sock.sendall(handshake.encode())
        self.buf = b""
        while b"\r\n\r\n" not in self.buf:
            chunk = self.sock.recv(4096)
            if not chunk:
                raise RuntimeError("ws handshake failed")
            self.buf += chunk
        head, self.buf = self.buf.split(b"\r\n\r\n", 1)
        if b"101" not in head.split(b"\r\n")[0]:
            raise RuntimeError("ws upgrade rejected: %s" % head[:120])

    def _recv(self, n):
        while len(self.buf) < n:
            chunk = self.sock.recv(65536)
            if not chunk:
                raise RuntimeError("ws closed")
            self.buf += chunk
        out, self.buf = self.buf[:n], self.buf[n:]
        return out

    def send_json(self, obj):
        payload = json.dumps(obj).encode()
        mask = os.urandom(4)
        masked = bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
        frame = b"\x81"  # FIN + text opcode
        ln = len(payload)
        if ln < 126:
            frame += struct.pack("!B", 0x80 | ln)
        else:
            frame += struct.pack("!BH", 0x80 | 126, ln)
        frame += mask + masked
        self.sock.sendall(frame)

    def recv_json(self, timeout=6):
        self.sock.settimeout(timeout)
        while True:
            b0, b1 = self._recv(2)
            opcode = b0 & 0x0F
            ln = b1 & 0x7F
            if ln == 126:
                ln = struct.unpack("!H", self._recv(2))[0]
            elif ln == 127:
                ln = struct.unpack("!Q", self._recv(8))[0]
            payload = self._recv(ln) if ln else b""
            if opcode == 0x1:
                try:
                    return json.loads(payload.decode())
                except Exception:
                    return {"raw": payload[:80].decode(errors="replace")}
            if opcode == 0x8:
                raise RuntimeError("ws closed by peer")

    def collect(self, want_type, seconds=5):
        """Collect frames until want_type arrives or time expires."""
        deadline = time.time() + seconds
        got = []
        while time.time() < deadline:
            try:
                fr = self.recv_json(timeout=max(0.5, deadline - time.time()))
            except Exception:
                break
            got.append(fr)
            if fr.get("type") == want_type:
                return got
        return got

    def close(self):
        try:
            self.sock.close()
        except Exception:
            pass


# ================================================================== TESTS
print("=== auth & RBAC ===")
root = login("root", "root")
doctor = login("doctor", "doctor123")
check("default account root/root works", bool(root))

s, b = req("POST", "/api/auth/login", data={"username": "root", "password": "root"})
perms = json.loads(b).get("permissions") or {}
check("login returns permission map", isinstance(perms, dict) and perms.get("user:manage") is True,
      "(%d perms)" % len(perms))

s, _ = req("GET", "/api/rbac/users", token=doctor)
check("doctor blocked from user list", s == 403, "status=%s" % s)
s, _ = req("GET", "/api/rbac/users", token=root)
check("root can list users", s == 200, "status=%s" % s)

print("\n=== operations analytics ===")
for ep in ("/api/operations/overview", "/api/operations/trend?days=7",
           "/api/operations/departments", "/api/operations/doctors", "/api/operations/status"):
    s, _ = req("GET", ep, token=root)
    check("root can read %s" % ep.split("?")[0], s == 200, "status=%s" % s)
s, _ = req("GET", "/api/operations/overview", token=doctor)
check("doctor blocked from analytics", s == 403, "status=%s" % s)

print("\n=== chat: REST ===")
s, b = req("GET", "/api/chat/online", token=root)
check("chat online endpoint", s == 200, "status=%s" % s)
s, b = req("GET", "/api/chat/history?group=1&limit=5", token=root)
check("group history", s == 200 and "items" in json.loads(b), "status=%s" % s)

s, b = multipart_post("/api/chat/images", root, "chat.png", b"PNGDATA-CHAT")
up = json.loads(b) if s in (200, 201) else {}
fname = up.get("file_name")
check("chat image upload", s in (200, 201) and bool(fname), "status=%s %s" % (s, fname))
if fname:
    s, b = req("GET", "/api/chat/files/%s" % fname, token=root)
    check("chat image serve (authed)", s == 200, "status=%s" % s)

print("\n=== chat: WebSocket ===")
wA = wB = None
try:
    wA = WS(doctor)   # A = doctor (id 2)
    wB = WS(root)     # B = root   (id 1)
    check("two WS clients connected", True)
except Exception as e:
    check("two WS clients connected", False, repr(e))

if wA and wB:
    pres = wB.collect("online", seconds=4)
    check("presence/online frame received", any(f.get("type") == "online" for f in pres),
          "%d frames" % len(pres))

    # --- group message: A sends, B must receive ---
    stamp = "grp-%d" % int(time.time())
    wA.send_json({"type": "message", "scope": "group", "content_type": "text", "content": stamp})
    frames = wB.collect("message", seconds=5)
    hit = [f for f in frames if f.get("type") == "message" and f.get("content") == stamp]
    check("group message delivered to peer", bool(hit), "%d frames" % len(frames))
    if hit:
        check("group message scope=group", hit[0].get("scope") == "group", str(hit[0].get("scope")))

    # --- image message over WS ---
    if fname:
        wA.send_json({"type": "message", "scope": "group", "content_type": "image",
                      "content": "", "file_name": fname, "orig_name": "chat.png", "file_size": 14})
        frames = wB.collect("message", seconds=5)
        imgs = [f for f in frames if f.get("content_type") == "image" and f.get("file_name") == fname]
        check("image message delivered", bool(imgs), "%d frames" % len(frames))

    # --- direct message A -> B (root id = 1) ---
    dstamp = "dm-%d" % int(time.time())
    wA.send_json({"type": "message", "scope": "direct", "receiver_id": 1,
                  "content_type": "text", "content": dstamp})
    frames = wB.collect("message", seconds=5)
    dm = [f for f in frames if f.get("content") == dstamp and f.get("scope") == "direct"]
    check("direct message delivered to recipient", bool(dm), "%d frames" % len(frames))
    if dm:
        check("direct message has peer_key", bool(dm[0].get("peer_key")), str(dm[0].get("peer_key")))
        check("direct message sets receiver_id", dm[0].get("receiver_id") == 1, str(dm[0].get("receiver_id")))

    # --- history reflects what was sent ---
    time.sleep(0.5)
    s, b = req("GET", "/api/chat/history?group=1&limit=10", token=root)
    hist = json.loads(b).get("items", []) if s == 200 else []
    check("group history contains sent message", any(h.get("content") == stamp for h in hist),
          "%d items" % len(hist))

    s, b = req("GET", "/api/chat/history?peer=2&limit=10", token=root)
    dhist = json.loads(b).get("items", []) if s == 200 else []
    check("dm history contains sent message", any(h.get("content") == dstamp for h in dhist),
          "%d items" % len(dhist))

    # unsupported content type must be rejected gracefully
    wA.send_json({"type": "message", "scope": "group", "content_type": "exe", "content": "x"})
    frames = wA.collect("error", seconds=3)
    check("unsupported content type rejected", any(f.get("type") == "error" for f in frames))

    wA.close()
    wB.close()

try:
    WS("garbage.token.value")
    check("WS rejects invalid token", False, "connected anyway")
except Exception:
    check("WS rejects invalid token", True)

print("\n=== health ===")
s, b = req("GET", "/health")
hp = json.loads(b) if s == 200 else {}
check("health reports db_driver", hp.get("db_driver") == "sqlite", str(hp))

print("\n=== SUMMARY ===")
if fails:
    print("FAILURES:", fails)
    raise SystemExit(1)
print("ALL CHECKS PASSED")
