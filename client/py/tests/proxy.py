#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import socket
import threading

_CHUNK_SIZE = 65536
_UPGRADED = b"HTTP/1.1 101"
_KEEP_ALIVE = (
    b'\xfe{"payload":{"frame":{"keys":[],"series":[]},"group":0,"keep_alive":true},'
    b'"error":{"type":"","data":""},"type":"data"}'
)
# An unmasked, unfragmented binary WebSocket frame.
_KEEP_ALIVE_FRAME = bytes([0x82, len(_KEEP_ALIVE)]) + _KEEP_ALIVE


def _frame_end(buf: bytes, start: int) -> int | None:
    if len(buf) < start + 2:
        return None
    size, offset = buf[start + 1] & 0x7F, start + 2
    if size >= 126:
        width = 2 if size == 126 else 8
        if len(buf) < offset + width:
            return None
        size = int.from_bytes(buf[offset : offset + width])
        offset += width
    end = offset + size
    return end if len(buf) >= end else None


class _Injector:
    """Writes a keep-alive into a Core WebSocket stream right after the transport's
    open message, ahead of any response from the endpoint.
    """

    def __init__(self) -> None:
        self._buf = b""
        self._done = False

    def feed(self, data: bytes) -> bytes:
        """Returns the bytes ready to forward, holding back a partial prefix."""
        if self._done:
            return data
        self._buf += data
        head_end = self._buf.find(b"\r\n\r\n")
        if head_end < 0:
            return b""
        if not self._buf.startswith(b"HTTP/1.1 101"):
            self._done = True
            return self._buf
        end = _frame_end(self._buf, head_end + 4)
        if end is None:
            return b""
        self._done = True
        return self._buf[:end] + _KEEP_ALIVE_FRAME + self._buf[end:]


class _Pair:
    def __init__(
        self,
        downstream: socket.socket,
        upstream: socket.socket,
        injector: _Injector | None,
    ) -> None:
        self.downstream = downstream
        self.upstream = upstream
        self.injector = injector
        self.blackholed = False
        self.head = b""
        self.upgraded: bool | None = None


class Proxy:
    """A TCP proxy in front of the live Core that can silently kill connections
    without touching the Core. Point a client at `port`.

    :param inject_keep_alive: Sends every WebSocket client a keep-alive before the
    endpoint's first response, as if the Core sent one ahead of the open ack.
    """

    port: int

    def __init__(self, host: str, port: int, inject_keep_alive: bool = False) -> None:
        self._target = (host, port)
        self._keep_alive_injected = inject_keep_alive
        self._server = socket.create_server(("localhost", 0))
        self._server.settimeout(0.05)
        self.port = self._server.getsockname()[1]
        self._pairs: set[_Pair] = set()
        self._lock = threading.Lock()
        self._closed = threading.Event()
        self._acceptor = threading.Thread(target=self._accept, daemon=True)
        self._acceptor.start()

    def _accept(self) -> None:
        while not self._closed.is_set():
            try:
                downstream, _ = self._server.accept()
            except TimeoutError:
                continue
            downstream.settimeout(None)
            pair = _Pair(
                downstream,
                socket.create_connection(self._target),
                _Injector() if self._keep_alive_injected else None,
            )
            with self._lock:
                self._pairs.add(pair)
            for src, dst in (
                (pair.downstream, pair.upstream),
                (pair.upstream, pair.downstream),
            ):
                threading.Thread(
                    target=self._forward, args=(pair, src, dst), daemon=True
                ).start()

    def _forward(self, pair: _Pair, src: socket.socket, dst: socket.socket) -> None:
        try:
            while data := src.recv(_CHUNK_SIZE):
                if src is pair.upstream and pair.upgraded is None:
                    pair.head += data
                    if len(pair.head) >= len(_UPGRADED):
                        pair.upgraded = pair.head.startswith(_UPGRADED)
                if src is pair.upstream and pair.injector is not None:
                    data = pair.injector.feed(data)
                # A blackholed client's writes are read and dropped so backpressure
                # never closes its socket.
                if not pair.blackholed:
                    dst.sendall(data)
        except OSError:
            pass
        if not pair.blackholed:
            self._drop(pair)

    def _drop(self, pair: _Pair) -> None:
        with self._lock:
            self._pairs.discard(pair)
        for s in (pair.downstream, pair.upstream):
            try:
                s.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            s.close()

    @property
    def websockets(self) -> int:
        """The number of live WebSocket connections."""
        with self._lock:
            return sum(1 for p in self._pairs if p.upgraded)

    def blackhole(self) -> int:
        """Kills the Core side of every live connection and holds the client side open
        and silent, modeling a half-open socket. New connections still forward.

        :returns: The number of connections blackholed.
        """
        with self._lock:
            pairs = [p for p in self._pairs if not p.blackholed]
        for pair in pairs:
            pair.blackholed = True
            pair.upstream.shutdown(socket.SHUT_RDWR)
        return len(pairs)

    def close(self) -> None:
        """Closes every connection, blackholed ones included, and stops the proxy."""
        self._closed.set()
        self._acceptor.join()
        with self._lock:
            pairs = list(self._pairs)
        for pair in pairs:
            self._drop(pair)
        self._server.close()
