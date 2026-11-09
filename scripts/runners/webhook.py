#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Receives GitHub workflow_job deliveries over HTTP and queues them for the
controller to drain on its next cycle."""

import hashlib
import hmac
import json
import threading
from collections import deque
from collections.abc import Callable
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from common import warn

EVENT = "workflow_job"
# A backlog this deep already means the controller is not cycling; the oldest
# deliveries are the least useful, and the resync rebuilds what they carried.
MAX_QUEUED = 1000
# GitHub caps a delivery at 25 MB; workflow_job payloads are a few KB.
MAX_BODY = 1024 * 1024
# Bounds how long a client can hold a request thread.
REQUEST_TIMEOUT = 10


class Webhook:
    """Serves POSTs on port from a daemon thread, drops any delivery whose
    X-Hub-Signature-256 does not match secret, and keeps the workflow_job payloads
    until drain() takes them."""

    def __init__(
        self,
        *,
        secret: str,
        port: int,
        report: Callable[[str, Exception], None] = warn,
    ) -> None:
        self.secret = secret.encode()
        self.report = report
        self.deliveries: deque[dict] = deque(maxlen=MAX_QUEUED)
        self.lock = threading.Lock()
        self.received = 0
        self.rejected = 0
        self.server = ThreadingHTTPServer(("0.0.0.0", port), self.handler())
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()

    def valid(self, body: bytes, *, signature: str) -> bool:
        digest = hmac.new(self.secret, body, hashlib.sha256).hexdigest()
        return hmac.compare_digest(f"sha256={digest}", signature)

    def enqueue(self, body: bytes) -> None:
        try:
            payload = json.loads(body)
        except Exception as e:
            self.report("webhook decode", e)
            return
        with self.lock:
            self.deliveries.append(payload)
            self.received += 1

    def drain(self) -> list[dict]:
        with self.lock:
            taken = list(self.deliveries)
            self.deliveries.clear()
        return taken

    def handler(self) -> type[BaseHTTPRequestHandler]:
        hook = self

        class Handler(BaseHTTPRequestHandler):
            protocol_version = "HTTP/1.1"
            timeout = REQUEST_TIMEOUT

            def do_POST(self) -> None:
                try:
                    length = int(self.headers.get("Content-Length", 0))
                except ValueError:
                    length = -1
                if not 0 <= length <= MAX_BODY:
                    hook.rejected += 1
                    self.close_connection = True
                    self.reply(413)
                    return
                body = self.rfile.read(length)
                if not hook.valid(
                    body, signature=self.headers.get("X-Hub-Signature-256", "")
                ):
                    hook.rejected += 1
                    self.reply(401)
                    return
                if self.headers.get("X-GitHub-Event") == EVENT:
                    hook.enqueue(body)
                # Anything else, ping included, is acknowledged and dropped.
                self.reply(204)

            def reply(self, code: int) -> None:
                self.send_response(code)
                self.send_header("Content-Length", "0")
                self.end_headers()

            def log_message(self, *args: object) -> None:
                pass  # Per-request access logs would flood the log channel.

        return Handler

    def close(self) -> None:
        self.server.shutdown()
        self.server.server_close()
