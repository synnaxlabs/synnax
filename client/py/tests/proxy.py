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


class _Pair:
    def __init__(self, downstream: socket.socket, upstream: socket.socket) -> None:
        self.downstream = downstream
        self.upstream = upstream
        self.blackholed = False


class Proxy:
    """A TCP proxy in front of the live Core that can silently kill connections
    without touching the Core. Point a client at `port`.
    """

    port: int

    def __init__(self, host: str, port: int) -> None:
        self._target = (host, port)
        self._server = socket.create_server(("localhost", 0))
        self._server.settimeout(0.05)
        self.port = self._server.getsockname()[1]
        self._pairs: set[_Pair] = set()
        self._lock = threading.Lock()
        self._closed = threading.Event()
        threading.Thread(target=self._accept, daemon=True).start()

    def _accept(self) -> None:
        while not self._closed.is_set():
            try:
                downstream, _ = self._server.accept()
            except TimeoutError:
                continue
            downstream.settimeout(None)
            pair = _Pair(downstream, socket.create_connection(self._target))
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
        with self._lock:
            pairs = list(self._pairs)
        for pair in pairs:
            self._drop(pair)
        self._server.close()
