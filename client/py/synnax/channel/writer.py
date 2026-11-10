#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Mapping, Sequence
from typing import Any, Literal

from pydantic import BaseModel

from alamos import Instrumentation, trace
from freighter import Empty, UnaryClient
from synnax import actions, group as group_
from synnax.channel.payload import (
    Key,
    NormalizedNameResult,
    Params,
    Payload,
    normalize_params,
)
from synnax.channel.types_gen import Update
from synnax.transport import send_batch
from x import patch


class _CreateRequest(BaseModel):
    channels: list[Payload]


class _Response(BaseModel):
    channels: list[Payload]


class _DeleteRequest(BaseModel):
    keys: list[Key] | tuple[Key, ...] | None = None
    names: list[str] | tuple[str, ...] | None = None


class _RenamePayload(BaseModel):
    name: str


class _RenameAction(BaseModel):
    type: Literal["rename"] = "rename"
    rename: _RenamePayload


class _UpdateAction(BaseModel):
    type: Literal["update"] = "update"
    update: patch.Update[Update]


_Action = _RenameAction | _UpdateAction


class Writer:
    _client: UnaryClient
    _sender: actions.Sender[Key, _Action]
    instrumentation: Instrumentation

    def __init__(
        self,
        *,
        client: UnaryClient,
        instrumentation: Instrumentation,
    ):
        self._client = client
        self._sender = actions.Sender[Key, _Action](
            client=client, endpoint="/channel/dispatch", update=_UpdateAction
        )
        self.instrumentation = instrumentation

    @trace("debug")
    def create(
        self,
        channels: list[Payload],
    ) -> list[Payload]:
        req = _CreateRequest(channels=channels)
        return self._client.send("/channel/create", req, _Response).channels

    @trace("debug")
    def delete(self, channels: Params) -> None:
        normal = normalize_params(channels)
        if isinstance(normal, NormalizedNameResult):
            req = _DeleteRequest(names=normal.channels)
        else:
            req = _DeleteRequest(keys=normal.channels)
        self._client.send("/channel/delete", req, Empty)

    @trace("debug")
    def rename(
        self, keys: list[Key] | tuple[Key, ...], names: list[str] | tuple[str, ...]
    ) -> None:
        reqs = [
            actions.DispatchRequest(
                key=k, actions=[_RenameAction(rename=_RenamePayload(name=n))]
            )
            for k, n in zip(keys, names, strict=True)
        ]
        send_batch(self._client, "/channel/dispatch", reqs)

    @trace("debug")
    def update(self, updates: Sequence[Mapping[str, Any]]) -> None:
        self._sender.update(updates)

    @trace("debug")
    def set_group(self, keys: list[Key], group: group_.Key | None) -> None:
        self.update([{"key": k, "group": group} for k in keys])

    @trace("debug")
    def set_expression(self, key: Key, expression: str) -> None:
        self.update([{"key": key, "expression": expression}])
