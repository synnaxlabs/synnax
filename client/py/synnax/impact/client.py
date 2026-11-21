#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import UnaryClient
from synnax.impact.types_gen import ApplyRequest, Change, Choice, Plan, PlanRequest


class Client:
    """Lists every record that a change to channels touches, and carries it out."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    def plan(
        self,
        change: Change,
        *,
        picks: dict[str, Choice] | None = None,
        hidden: Choice | None = None,
    ) -> Plan:
        """Returns every record that change touches with the picks: an entry for each
        record that the caller can see, a count of the others, and the token that
        apply checks. A delete pick on a channel adds the records that hold it. Writes
        nothing.
        """
        req = PlanRequest(change=change, picks=picks or {}, hidden=hidden)
        return self._client.send("/impact/plan", req, Plan)

    def apply(
        self,
        change: Change,
        *,
        token: str = "",
        picks: dict[str, Choice] | None = None,
        hidden: Choice | None = None,
    ) -> Plan:
        """Carries out change with the picks and returns the entries that it applied.
        With a token, it raises PlanChangedError before any write when the plan is no
        longer the one that the token names.
        """
        req = ApplyRequest(change=change, token=token, picks=picks or {}, hidden=hidden)
        return self._client.send("/impact/apply", req, Plan)
