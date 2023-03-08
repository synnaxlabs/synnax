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
    keys_or_names: list[str] | None = None
    node_id: int | None = None


class _Response(Payload):
    channels: list[ChannelPayload] = []
    not_found: list[str] = []


class ChannelRetriever(Protocol):
    """Protocol for retrieving channel paylods from the cluster."""

    @overload
    def retrieve(
        self,
        key_or_name: str,
    ) -> ChannelPayload | None:
        ...

    @overload
    def retrieve(
        self,
        *keys_or_names: str | list[str],
        node_id: int | None = None,
        include_not_found: Literal[False] = False,
    ) -> list[ChannelPayload]:
        ...

    @overload
    def retrieve(
        self,
        *keys_or_names: str | list[str],
        node_id: int | None = None,
        include_not_found: Literal[True] = True,
    ) -> tuple[list[ChannelPayload], list[str]]:
        ...

    @overload
    def retrieve(
        self,
        key_or_name: str | list[str],
        *keys_or_names: str | list[str],
        node_id: int | None = None,
        include_not_found: bool = False,
    ) -> (
        tuple[list[ChannelPayload], list[str]]
        | list[ChannelPayload]
        | ChannelPayload
        | None
    ):
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
        key_or_name: str | list[str],
        *keys_or_names: str | list[str],
        node_id: int | None = None,
        include_not_found: bool = False,
    ) -> (
        tuple[list[ChannelPayload], list[str]]
        | list[ChannelPayload]
        | ChannelPayload
        | None
    ):
        single = is_single(key_or_name, keys_or_names)
        flat = flatten(key_or_name, keys_or_names)

        req = _Request(keys_or_names=flat, node_id=node_id)
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        if include_not_found is True:
            return res.channels, res.not_found
        if single:
            if len(res.channels) == 1:
                return res.channels[0]
            if len(res.channels) == 0:
                return None
            raise QueryError("multiple channels found")
        return res.channels


class CacheChannelRetriever:
    _retriever: ChannelRetriever
    channels: dict[str, ChannelPayload]
    names_to_keys: dict[str, str]

    def __init__(self, retriever: ChannelRetriever) -> None:
        self.channels = dict()
        self.names_to_keys = dict()
        self._retriever = retriever

    def _(self) -> ChannelRetriever:
        return self

    def retrieve(
        self,
        key_or_name: str | list[str],
        *keys_or_names: str | list[str],
        node_id: int | None = None,
        include_not_found: bool = False,
    ) -> (
        tuple[list[ChannelPayload], list[str]]
        | list[ChannelPayload]
        | ChannelPayload
        | None
    ):
        if node_id is not None:
            return self._retriever.retrieve(
                key_or_name,
                *keys_or_names,
                node_id=node_id,
                include_not_found=include_not_found,
            )

        single = len(keys_or_names) == 1 and isinstance(keys_or_names[0], str)
        flat = flatten(key_or_name, keys_or_names)

        results = list()
        to_retrieve = list()

        for entry in flat:
            key = self.names_to_keys.get(entry, entry)
            channel = self.channels.get(key, None)
            if channel is None:
                to_retrieve.append(key)
            else:
                results.append(channel)

        if len(to_retrieve) == 0:
            return results

        retrieved, not_found = self._retriever.retrieve(
            to_retrieve,
            include_not_found=True,
        )

        for channel in retrieved:
            self.channels[channel.key] = channel
            self.names_to_keys[channel.name] = channel.key
            results.append(channel)

        if include_not_found:
            return results, not_found
        if single:
            return None if len(results) == 0 else results[0]
        return results


def flatten(
    key_or_name: str | list[str], keys_or_names: tuple[str | list[str]]
) -> list[str]:
    """Flatten a list of keys or names."""
    _keys_or_names = [key_or_name] if isinstance(key_or_name, str) else key_or_name
    for keys_or_name in keys_or_names:
        if isinstance(keys_or_name, str):
            _keys_or_names.append(keys_or_name)
        else:
            _keys_or_names.extend(keys_or_name)
    return _keys_or_names


def is_single(
    key_or_name: str | list[str], keys_or_names: tuple[str | list[str]]
) -> bool:
    """Determine if a list of keys or names is a single key or name."""
    return isinstance(key_or_name, str) & len(keys_or_names) == 0
