#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import UnaryClient
from synnax import group
from synnax.channel import types_gen as channel
from synnax.impact.types_gen import (
    ApplyRequest,
    Change,
    DeleteChange,
    DeleteGroupsChange,
    DeleteGroupsResponse,
    DeleteResponse,
    Effect,
    PlanRequest,
    Response,
)


def delete_channels(keys: list[channel.Key]) -> DeleteChange:
    """Returns the change that deletes the channels with keys."""
    return DeleteChange(records=[channel.resource_id(key) for key in keys])


def delete_groups(keys: list[group.Key]) -> DeleteGroupsChange:
    """Returns the change that deletes the channel folders with keys and their
    tombstones.
    """
    return DeleteGroupsChange(groups=keys)


def _response_type(
    change: Change,
) -> type[DeleteResponse] | type[DeleteGroupsResponse]:
    """Returns the type of the response to change."""
    if isinstance(change, DeleteGroupsChange):
        return DeleteGroupsResponse
    return DeleteResponse


class Client:
    """Lists every record that a change to channels touches, and carries it out."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    def plan(
        self,
        change: Change,
        *,
        picks: dict[str, Effect] | None = None,
        hidden: Effect | None = None,
    ) -> Response:
        """Returns every record that change touches with the picks, under the type of
        change: an entry for each record that the caller can see, a count of the
        others, and the token that apply checks. A delete pick on a channel adds the
        records that hold it. Writes nothing.
        """
        req = PlanRequest(change=change, picks=picks or {}, hidden=hidden)
        return self._client.send("/impact/plan", req, _response_type(change))

    def apply(
        self,
        change: Change,
        *,
        token: str = "",
        picks: dict[str, Effect] | None = None,
        hidden: Effect | None = None,
    ) -> Response:
        """Carries out change with the picks and returns the entries that it applied,
        under the type of change. With a token, it raises PlanChangedError before any
        write when the plan is no longer the one that the token names.
        """
        req = ApplyRequest(change=change, token=token, picks=picks or {}, hidden=hidden)
        return self._client.send("/impact/apply", req, _response_type(change))
