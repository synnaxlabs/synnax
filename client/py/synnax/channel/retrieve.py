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

from typing import Protocol

from typing_extensions import Literal

from alamos import Instrumentation, trace, NOOP
from freighter import Payload, UnaryClient
from synnax.channel.payload import (
    ChannelPayload,
    ChannelKeys,
    ChannelNames,
    ChannelParams, ChannelKey, ChannelName
)
from synnax.util.normalize import normalize


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
    ) -> list[ChannelPayload]:
        normal = normalize_channel_params(params)
        req = _Request(**{normal.variant: normal.params})
        res, exc = self.client.send(self._ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
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

    def _get(self, param: ChannelKey | ChannelName) -> ChannelPayload | None:
        if isinstance(param, int):
            return self.channels.get(param)
        return self.channels.get(self.names_to_keys.get(param))

    @trace("debug")
    def retrieve(self, params: ChannelParams) -> list[ChannelPayload]:
        normal = normalize_channel_params(params)
        results = list()
        to_retrieve = list()
        for p in normal.params:
            ch = self._get(p)
            if ch is None:
                to_retrieve.append(p)
            else:
                results.append(ch)

        if len(to_retrieve) == 0:
            return results

        retrieved = self._retriever.retrieve(to_retrieve)
        for ch in retrieved:
            self.channels[ch.key] = ch
            self.names_to_keys[ch.name] = ch.key
            results.append(ch)

        return results


def normalize_channel_params(
    params: ChannelParams,
) -> NormalizedChannelParams:
    """Determine if a list of keys or names is a single key or name."""
    normalized = normalize(params)
    if len(normalized) == 0:
        raise ValueError("no keys or names provided")
    return NormalizedChannelParams(
        single=isinstance(params, (str, int)),
        variant="keys" if isinstance(normalized[0], int) else "names",
        params=normalized,
    )


@dataclass
class NormalizedChannelParams:
    single: bool
    variant: Literal["keys", "names"]
    params: ChannelNames | ChannelKeys
