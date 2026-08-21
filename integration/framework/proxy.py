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


class SeverableProxy:
    """TCP proxy in front of the live test Core so cases can simulate Core
    downtime without touching the Core itself. Point the console at ``port``,
    then ``sever()`` and ``restore()`` the link."""

    def __init__(self, target_host: str = "localhost", target_port: int = 9090):
        self.target_host = target_host
        self.target_port = target_port
        self._lock = threading.Lock()
        self._sockets: set[socket.socket] = set()
        self._listener: socket.socket | None = None
        self._severed = False
        self.port = self._listen(0)

    def _listen(self, port: int) -> int:
        listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        listener.bind(("127.0.0.1", port))
        listener.listen()
        with self._lock:
            self._severed = False
        self._listener = listener
        threading.Thread(
            target=self._accept_loop, args=(listener,), daemon=True
        ).start()
        return int(listener.getsockname()[1])

    def _accept_loop(self, listener: socket.socket) -> None:
        while True:
            try:
                downstream, _ = listener.accept()
            except OSError:
                return
            try:
                upstream = socket.create_connection(
                    (self.target_host, self.target_port), timeout=5
                )
            except OSError:
                downstream.close()
                continue
            # create_connection leaves its connect timeout on the socket, where
            # it would expire on a quiet stream and drop a healthy link.
            upstream.settimeout(None)
            with self._lock:
                live = not self._severed
                if live:
                    self._sockets.add(downstream)
                    self._sockets.add(upstream)
            # A sever took its snapshot before this pair was registered, so
            # nothing else will ever close it.
            if not live:
                self._drop(downstream, upstream)
                continue
            threading.Thread(
                target=self._pump, args=(downstream, upstream), daemon=True
            ).start()
            threading.Thread(
                target=self._pump, args=(upstream, downstream), daemon=True
            ).start()

    def _pump(self, src: socket.socket, dst: socket.socket) -> None:
        try:
            while True:
                data = src.recv(65536)
                if not data:
                    break
                dst.sendall(data)
        except OSError:
            pass
        finally:
            self._drop(src, dst)

    def _drop(self, *socks: socket.socket) -> None:
        with self._lock:
            for s in socks:
                self._sockets.discard(s)
        for s in socks:
            try:
                s.close()
            except OSError:
                pass

    def sever(self) -> None:
        """Drop every live connection and refuse new ones, as if the Core died."""
        listener, self._listener = self._listener, None
        if listener is not None:
            listener.close()
        with self._lock:
            self._severed = True
            socks, self._sockets = list(self._sockets), set()
        for s in socks:
            try:
                s.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            s.close()

    def restore(self) -> None:
        """Accept connections again on the same port."""
        if self._listener is not None:
            return
        self._listen(self.port)

    def close(self) -> None:
        """Shut the proxy down for good."""
        self.sever()
