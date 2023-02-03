#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations
from typing import Protocol

from freighter import HTTPClientFactory, Payload, UnaryClient

from synnax.exceptions import QueryError
from synnax.channel.payload import ChannelPayload


class _Request(Payload):
    keys: list[str] | None = None
    node_id: int | None = None
    names: list[str] | None = None


class _Response(Payload):
    channels: list[ChannelPayload] | None = []


class ChannelRetriever(Protocol):
    def retrieve(
        self, key: str | None = None, name: str | None = None
    ) -> ChannelPayload | None:
        ...

    def filter(
        self,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
    ) -> list[ChannelPayload]:
        ...


class ClusterChannelRetriever:
    _ENDPOINT = "/channel/retrieve"
    client: UnaryClient

    def __init__(self, client: HTTPClientFactory):
        self.client = client.get_client()

    def _(self) -> ChannelRetriever:
        return self

    def retrieve(
        self, key: str | None = None, name: str | None = None
    ) -> ChannelPayload | None:
        req = _Request()
        if key is not None:
            req.keys = [key]
        if name is not None:
            req.names = [name]
        res = self._execute(req)
        if len(res) == 1:
            return res[0]
        if len(res) == 0:
            return None
        raise QueryError("multiple channels found")

    def filter(
        self,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
    ) -> list[ChannelPayload]:
        return self._execute(_Request(keys=keys, names=names, node_id=node_id))

    def _execute(self, req: _Request) -> list[ChannelPayload]:
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        assert res.channels is not None
        return res.channels


class CacheChannelRetriever:
    retriever: ChannelRetriever
    channels: dict[str, ChannelPayload]
    names_to_keys: dict[str, str]

    def __init__(self, retriever: ChannelRetriever) -> None:
        self.channels = dict()
        self.names_to_keys = dict()
        self.retriever = retriever

    def _(self) -> ChannelRetriever:
        return self

    def retrieve(
        self, key: str | None = None, name: str | None = None
    ) -> ChannelPayload | None:
        if key is None:
            if name is None:
                return None
            key = self.names_to_keys.get(name, None)
            if key is None:
                return None
        record = self.channels.get(key, None)
        if record is None:
            record = self.retriever.retrieve(key=key)
            if record is None:
                return None
            self.channels[key] = record
        return record

    def filter(
        self,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
    ) -> list[ChannelPayload]:
        # TODO: Bypassing the cache on these filters for now. We need to revisit
        # this when node_id becomes a more relevant filter.
        if node_id is not None:
            return self.retriever.filter(keys=keys, names=names, node_id=node_id)
        results = list()
        retrieve_keys = list()
        retrieve_names = list()
        keys = keys or list()
        if names is not None:
            for name in names:
                key = self.names_to_keys.get(name, None)
                if key is not None:
                    keys.append(key)
                else:
                    retrieve_names.append(name)
        for key in keys:
            channel = self.channels.get(key, None)
            if channel is None:
                retrieve_keys.append(key)
            else:
                results.append(channel)
        if len(retrieve_keys) > 0 or len(retrieve_names) > 0:
            results.extend(
                self.retriever.filter(names=retrieve_names, keys=retrieve_keys)
            )
        return results
