#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations
from typing import Protocol, overload
from typing_extensions import Literal

from freighter import HTTPClientPool, Payload, UnaryClient

from synnax.exceptions import QueryError
from synnax.channel.payload import ChannelPayload


class _Request(Payload):
    keys: list[str] | None = None
    node_id: int | None = None
    names: list[str] | None = None


class _Response(Payload):
    channels: list[ChannelPayload] = []
    not_found: list[str] = []


class ChannelRetriever(Protocol):
    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        name: str | None = None,
    ) -> ChannelPayload | None:
        ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
        include_not_found: Literal[False] = False,
    ) -> list[ChannelPayload]:
        ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
        include_not_found: Literal[True] = True,
    ) -> tuple[list[ChannelPayload], list[str]]:
        ...

    @overload
    def retrieve(
        self,
        *,
        key: str | None = None,
        name: str | None = None,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
        include_not_found: bool = False,
    ) -> list[ChannelPayload] | tuple[
        list[ChannelPayload], list[str]
    ] | ChannelPayload | None:
        ...


class ClusterChannelRetriever:
    _ENDPOINT = "/channel/retrieve"
    client: UnaryClient

    def __init__(self, client: HTTPClientPool):
        self.client = client.get_client()

    def _(self) -> ChannelRetriever:
        return self

    def retrieve(
        self,
        key: str | None = None,
        name: str | None = None,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
        include_not_found: bool = False,
    ) -> tuple[list[ChannelPayload], list[str]] | list[
        ChannelPayload
    ] | ChannelPayload | None:
        single_key = key is not None
        single_name = name is not None
        req = _Request(
            keys=[key] if single_key else keys,
            names=[name] if single_name else names,
            node_id=node_id,
        )
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        if include_not_found is True:
            return res.channels, res.not_found
        if single_key or single_name:
            if len(res.channels) == 1:
                return res.channels[0]
            if len(res.channels) == 0:
                return None
            raise QueryError("multiple channels found")
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
        self,
        key: str | None = None,
        name: str | None = None,
        keys: list[str] | None = None,
        names: list[str] | None = None,
        node_id: int | None = None,
        include_not_found: bool = False,
    ) -> tuple[list[ChannelPayload], list[str]] | list[
        ChannelPayload
    ] | ChannelPayload | None:
        if node_id is not None:
            return self.retriever.retrieve(
                node_id=node_id, include_not_found=include_not_found
            )

        keys, single_key = self._normalize(key, keys)
        names, single_name = self._normalize(name,names)
        keys_to_retrieve = list()
        names_to_retrieve = list()
        results = list()

        for name in names:
            key = self.names_to_keys.get(name, None)
            if key is not None:
                keys.append(key)
            else:
                names_to_retrieve.append(name)

        for key in keys:
            channel = self.channels.get(key, None)
            if channel is None:
                keys_to_retrieve.append(key)
            else:
                results.append(channel)

        if len(keys_to_retrieve) == 0 and len(names_to_retrieve) == 0:
            return results

        retrieved, not_found = self.retriever.retrieve(
            keys=keys_to_retrieve,
            names=names_to_retrieve,
            include_not_found=True,
        )

        for channel in retrieved:
            self.channels[channel.key] = channel
            self.names_to_keys[channel.name] = channel.key
            results.append(channel)

        if include_not_found:
            return results, not_found
        if single_key or single_name:
            return None if len(results) == 0 else results[0]
        return results

    def _normalize(self, key: str | None, keys: list[str] | None) -> tuple[list[str], bool]:
        if key is not None:
            return [key], True
        if keys is None:
            return [], False
        return keys, False
