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

from alamos import Instrumentation, trace, NOOP
from freighter import HTTPClientPool, Payload, UnaryClient
from synnax.channel.payload import ChannelPayload, Keys, Names, KeysOrNames
from synnax.exceptions import QueryError
from synnax.util.flatten import flatten


class _Request(Payload):
    names: list[str] | None = None
    keys: list[int] | None = None
    leaseholder: int | None = None


class _Response(Payload):
    channels: list[ChannelPayload] = []
    not_found: list[str] | None = []


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
        key_or_name: KeysOrNames,
        *keys_or_names: KeysOrNames,
        leaseholder: int | None = None,
        include_not_found: Literal[False] = False,
    ) -> list[ChannelPayload]:
        ...

    @overload
    def retrieve(
        self,
        key_or_name: Keys,
        *keys_or_names: Keys,
        leaseholder: int | None = None,
        include_not_found: Literal[True] = True,
    ) -> tuple[list[ChannelPayload], list[int]]:
        ...

    @overload
    def retrieve(
        self,
        key_or_name: Names,
        *keys_or_names: Names,
        leaseholder: int | None = None,
        include_not_found: Literal[True] = True,
    ) -> tuple[list[ChannelPayload], list[str]]:
        ...

    def retrieve(
        self,
        key_or_name: KeysOrNames,
        *keys_or_names: KeysOrNames,
        leaseholder: int | None = None,
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
    instrumentation: Instrumentation

    def __init__(
        self,
        client: HTTPClientPool,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self.client = client.get_client()
        self.instrumentation = instrumentation

    def _(self) -> ChannelRetriever:
        return self

    @trace("debug")
    def retrieve(
        self,
        key_or_name: KeysOrNames,
        *keys_or_names: KeysOrNames,
        leaseholder: int | None = None,
        include_not_found: bool = False,
    ) -> (
        tuple[list[ChannelPayload], list[str] | list[int]]
        | list[ChannelPayload]
        | ChannelPayload
        | None
    ):
        single = is_single(key_or_name, keys_or_names)
        keys, names = split_keys_and_names(key_or_name, keys_or_names)
        req = _Request(keys=keys, names=names, leaseholder=leaseholder)
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        if include_not_found:
            return res.channels, res.not_found or list()
        if single:
            if len(res.channels) == 1:
                return res.channels[0]
            if len(res.channels) == 0:
                return None
            raise QueryError("multiple channels found")
        return res.channels


class CacheChannelRetriever:
    _retriever: ChannelRetriever
    channels: dict[int, ChannelPayload]
    names_to_keys: dict[str, int]
    instrumentation: Instrumentation

    def __init__(
        self,
        retriever: ChannelRetriever,
        instrumentation: Instrumentation,
    ) -> None:
        self.channels = dict()
        self.names_to_keys = dict()
        self.instrumentation = instrumentation
        self._retriever = retriever

    def _(self) -> ChannelRetriever:
        return self

    @trace("debug")
    def retrieve(
        self,
        key_or_name: KeysOrNames,
        *keys_or_names: KeysOrNames,
        leaseholder: int | None = None,
        include_not_found: bool = False,
    ) -> (
        tuple[list[ChannelPayload], list[str]]
        | list[ChannelPayload]
        | ChannelPayload
        | None
    ):
        if leaseholder is not None:
            return self._retriever.retrieve(
                key_or_name,
                *keys_or_names,
                leaseholder=leaseholder,
                include_not_found=include_not_found,
            )

        single = is_single(key_or_name, keys_or_names)
        keys, names = split_keys_and_names(key_or_name, keys_or_names)

        results = list()
        to_retrieve = list()

        for name in names:
            key = self.names_to_keys.get(name)
            if key is None:
                to_retrieve.append(name)
            else:
                results.append(self.channels.get(key))

        for key in keys:
            channel = self.channels.get(key)
            if channel is None:
                to_retrieve.append(key)
            else:
                results.append(channel)

        not_found = list()
        if len(to_retrieve) != 0:
            retrieved, not_found = self._retriever.retrieve(
                *to_retrieve,
                include_not_found=True,
            )

            for channel in retrieved:
                self.channels[channel.key] = channel
                self.names_to_keys[channel.name] = channel.key
                results.append(channel)

        if include_not_found:
            return results, not_found
        if single:
            if len(results) == 0:
                return None
            if len(results) == 1:
                return results[0]
            raise QueryError("multiple channels found")
        return results


def is_single(
    key_or_name: KeysOrNames,
    keys_or_names: tuple[KeysOrNames],
) -> bool:
    """Determine if a list of keys or names is a single key or name."""
    return isinstance(key_or_name, (str, int)) and len(keys_or_names) == 0


def split_keys_and_names(
    key_or_name: KeysOrNames,
    keys_or_names: tuple[KeysOrNames],
) -> tuple[list[int], list[str]]:
    """Split a list of keys or names into a list of keys and a list of names."""
    keys = list()
    names = list()
    flat = flatten(key_or_name, *keys_or_names)
    for entry in flat:
        if isinstance(entry, int):
            keys.append(entry)
        else:
            names.append(entry)
    return keys, names
