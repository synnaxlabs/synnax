#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import hashlib
import hmac
import http.client
import json
from collections.abc import Iterator

import pytest

from webhook import MAX_BODY, Webhook

SECRET = "secret"


@pytest.fixture
def hook() -> Iterator[Webhook]:
    w = Webhook(secret=SECRET, port=0)
    yield w
    w.close()


def post(hook: Webhook, *, body: bytes, headers: dict[str, str]) -> int:
    conn = http.client.HTTPConnection("127.0.0.1", hook.server.server_address[1])
    conn.putrequest("POST", "/")
    for k, v in headers.items():
        conn.putheader(k, v)
    conn.endheaders()
    conn.send(body)
    status = conn.getresponse().status
    conn.close()
    return status


def signed(body: bytes) -> dict[str, str]:
    digest = hmac.new(SECRET.encode(), body, hashlib.sha256).hexdigest()
    return {
        "Content-Length": str(len(body)),
        "X-Hub-Signature-256": f"sha256={digest}",
        "X-GitHub-Event": "workflow_job",
    }


class TestWebhook:
    """Tests for the GitHub webhook receiver."""

    def test_should_queue_signed_deliveries(self, hook: Webhook) -> None:
        body = json.dumps({"action": "queued"}).encode()
        assert post(hook, body=body, headers=signed(body)) == 204
        assert hook.drain() == [{"action": "queued"}]

    def test_should_reject_unsigned_deliveries(self, hook: Webhook) -> None:
        body = b"{}"
        headers = signed(body) | {"X-Hub-Signature-256": "sha256=00"}
        assert post(hook, body=body, headers=headers) == 401
        assert hook.drain() == []

    def test_should_reject_a_body_past_the_limit_unread(self, hook: Webhook) -> None:
        headers = {"Content-Length": str(MAX_BODY + 1)}
        assert post(hook, body=b"", headers=headers) == 413
        assert hook.rejected == 1
