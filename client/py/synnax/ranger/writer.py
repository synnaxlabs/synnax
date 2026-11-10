#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Mapping, Sequence
from typing import Any

from pydantic import BaseModel

from alamos import NOOP, Instrumentation, trace
from freighter import Empty, UnaryClient
from synnax import actions
from synnax.ranger.types_gen import Action, Key, Payload, UpdateAction
from synnax.telem import TimeStamp


class _CreateRequest(BaseModel):
    ranges: list[Payload] = []


class _CreateResponse(BaseModel):
    ranges: list[Payload] = []


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Writer:
    _client: UnaryClient
    _sender: actions.Sender[Key, Action]
    instrumentation: Instrumentation

    def __init__(
        self,
        *,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self._sender = actions.Sender[Key, Action](
            client=client, endpoint="/range/dispatch", update=UpdateAction
        )
        self.instrumentation = instrumentation

    @trace("debug", key="range.create")
    def create(self, ranges: list[Payload]) -> list[Payload]:
        req = _CreateRequest(ranges=ranges)
        return self._client.send("/range/create", req, _CreateResponse).ranges

    @trace("debug", key="range.delete")
    def delete(self, keys: list[Key]) -> None:
        req = _DeleteRequest(keys=keys)
        self._client.send("/range/delete", req, Empty)

    @trace("debug", key="range.update")
    def update(self, updates: Sequence[Mapping[str, Any]]) -> None:
        self._sender.update(updates)

    @trace("debug", key="range.rename")
    def rename(self, key: Key, name: str) -> None:
        self.update([{"key": key, "name": name}])

    @trace("debug", key="range.set_end")
    def set_end(self, key: Key, end: TimeStamp) -> None:
        self.update([{"key": key, "time_range": {"end": end}}])
