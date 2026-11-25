#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import threading
import typing
from collections.abc import Iterator
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import override

import pytest

import synnax as sy
from freighter import HTTPClient
from freighter.url import URL
from synnax.exceptions import create_registry
from x.codec import JSONCodec

# What core/pkg/service/impact.Apply returns with no token, picks, or hidden pick,
# encoded by the Core's JSON codec: a plan of the change with no entries.
EMPTY_APPLIED_PLAN = (
    '{"entries":[],"kept":0,"hidden":{"restrict":0,"cascade":0,"detach":0},'
    '"token":"",'
    '"change":{"type":"delete_channels","keys":[1048577]}}'
)


# The change that the specs below send.
CHANGE = sy.impact.DeleteChannelsChange(keys=[1048577])


class FakeCore:
    """Answers each POST with the next status and body, as a Core over HTTP."""

    def __init__(self) -> None:
        self.replies: list[tuple[int, str]] = []
        self.sent: list[tuple[str, dict[str, object]]] = []
        core = self

        class Handler(BaseHTTPRequestHandler):
            def do_POST(self) -> None:
                n = int(self.headers.get("Content-Length", 0))
                core.sent.append((self.path, json.loads(self.rfile.read(n))))
                status, body = core.replies.pop(0)
                data = body.encode()
                self.send_response(status)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

            @override
            def log_message(self, *_: object) -> None:
                pass

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.client = sy.impact.Client(
            HTTPClient(
                url=URL(host="127.0.0.1", port=self.server.server_address[1]),
                codec=JSONCodec(),
                registry=create_registry(),
            )
        )

    def close(self) -> None:
        self.server.shutdown()
        self.server.server_close()


@pytest.fixture
def core() -> Iterator[FakeCore]:
    c = FakeCore()
    yield c
    c.close()


@pytest.mark.impact
def test_synnax_impact_return_annotation_resolves() -> None:
    """Should resolve the return type of Synnax.impact to impact.Client"""
    build = sy.Synnax.impact._build
    assert typing.get_type_hints(build)["return"] is sy.impact.Client


@pytest.mark.impact
def test_apply_with_no_options_returns_after_the_core_applies(core: FakeCore) -> None:
    """Should return from an apply with no options once the Core has applied it"""
    core.replies.append((200, EMPTY_APPLIED_PLAN))
    applied = core.client.apply(CHANGE)
    assert applied.entries == []
