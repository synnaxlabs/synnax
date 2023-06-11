#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from dataclasses import dataclass

from typing import Protocol, overload

from typing_extensions import Literal

from alamos import Instrumentation, trace, NOOP
from freighter import HTTPClient, Payload, UnaryClient
from synnax.channel.payload import (
    ChannelPayload,
    ChannelKeys,
    ChannelNames,
    ChannelParams, ChannelKey, ChannelName
)
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
    """Protocol for retrieving channel payloads from the cluster."""

    def retrieve(self, params: ChannelParams) -> list[ChannelPayload]:
        ...


class ClusterChannelRetriever:
    _ENDPOINT = "/channel/retrieve"
    client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self.client = client
        self.instrumentation = instrumentation

    def _(self) -> ChannelRetriever:
        return self

    @trace("debug")
    def retrieve(
        self,
        params: ChannelKey | ChannelName,
        include_not_found: bool = False,
    ) -> (
        tuple[list[ChannelPayload], list[str] | list[int]]
        | list[ChannelPayload]
        | ChannelPayload
        | None
    ):
        normal = normalize_channel_params(params)
        req = _Request(**{normal.variant: normal.params})
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        assert res is not None
        if include_not_found:
            return res.channels, res.not_found or list()
        if normal:
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
        key_or_name: ChannelParams,
        *keys_or_names: ChannelParams,
        include_not_found: bool = False,
    ) -> (
        tuple[list[ChannelPayload], list[str]]
        | list[ChannelPayload]
        | ChannelPayload
        | None
    ):

        single = normalize_channel_params(key_or_name, keys_or_names)
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


def normalize_channel_params(
    params: ChannelParams,
) -> NormalizedChannelParams:
    """Determine if a list of keys or names is a single key or name."""
    normalized = flatten(params),
    if len(normalized) == 0:
        raise ValueError("no keys or names provided")
    return NormalizedChannelParams(
        single=isinstance(params, (str, int)),
        variant="keys" if isinstance(params, int) else "names",
        params=normalized,
    )


@dataclass
class NormalizedChannelParams:
    single: bool
    variant: Literal["keys", "names"]
    params: ChannelNames | ChannelKeys
