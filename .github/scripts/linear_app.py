#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Opens tokens for the Synnax Linear app, an OAuth app with client credentials on.

Needs LINEAR_CLIENT_ID and LINEAR_CLIENT_SECRET. Linear shows its changes as the app.
"""

import json
import os
import urllib.parse
import urllib.request
from collections.abc import Iterator
from contextlib import contextmanager

LINEAR_OAUTH = "https://api.linear.app/oauth"
# Linear revokes every app token when a request asks for other scopes.
LINEAR_SCOPE = "read,write"


@contextmanager
def linear_token() -> Iterator[str]:
    """Yields a short-lived Linear app token and revokes it on exit, so runs do not
    pile up toward Linear's limit of active tokens per app."""
    form = {
        "grant_type": "client_credentials",
        "client_id": os.environ["LINEAR_CLIENT_ID"],
        "client_secret": os.environ["LINEAR_CLIENT_SECRET"],
        "scope": LINEAR_SCOPE,
    }
    with urllib.request.urlopen(
        f"{LINEAR_OAUTH}/token", urllib.parse.urlencode(form).encode(), timeout=30
    ) as resp:
        token: str = json.load(resp)["access_token"]
    if os.environ.get("GITHUB_ACTIONS") == "true":
        print(f"::add-mask::{token}", flush=True)
    try:
        yield token
    finally:
        revoke = {"token": token, "token_type_hint": "access_token"}
        urllib.request.urlopen(
            f"{LINEAR_OAUTH}/revoke",
            urllib.parse.urlencode(revoke).encode(),
            timeout=30,
        ).close()
