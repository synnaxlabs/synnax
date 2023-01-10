#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import typing

from freighter import HTTPClientFactory, Payload, UnaryClient

from ..exceptions import QueryError, ValidationError
from .payload import ChannelPayload


class _Request(Payload):
    keys: list[str] | None = None
    node_id: int | None = None
    names: list[str] | None = None


class _Response(Payload):
    channels: list[ChannelPayload] | None = []


class ChannelRetriever:
    _ENDPOINT = "/channel/retrieve"
    client: UnaryClient

    def __init__(self, client: HTTPClientFactory):
        self.client = client.get_client()

    def retrieve(self, key: str = None, name: str = None) -> ChannelPayload:
        req = _Request()
        if key is None and name is None:
            raise ValidationError("Must specify a key or name")
        if key is not None:
            req.keys = [key]
        if name is not None:
            req.names = [name]
        res = self._execute(req)
        if len(res) == 0:
            raise QueryError("channel not found")
        elif len(res) > 1:
            raise QueryError("multiple channels found")
        return res[0]

    def filter(
        self,
        keys: list[str] = None,
        names: list[str] = None,
        node_id: int = None,
    ) -> list[ChannelPayload]:
        return self._execute(_Request(keys=keys, names=names))

    def _execute(self, req: _Request) -> list[ChannelPayload]:
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        return res.channels
