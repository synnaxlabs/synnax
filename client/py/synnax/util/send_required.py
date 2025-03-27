#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from freighter import URL, UnaryClient
from freighter import send_required as fsend_required
from freighter.exceptions import Unreachable
from freighter.transport import RQ, RS


def send_required(client: UnaryClient, target: str, req: RQ, res_t: type[RS]) -> RS:
    """Sends a request to a target and returns the response. Raises an exception if the
    request returns an error.
    """
    try:
        return fsend_required(client, target, req, res_t)
    except Unreachable as exc:
        url = URL.parse(exc.target)
        raise Unreachable(
            target=url.stringify(),
            message=f"Failed to reach cluster at {url.host}:{url.port}",
        ) from exc
