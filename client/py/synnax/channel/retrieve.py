#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Protocol, cast

from alamos import NOOP, Instrumentation, trace
from freighter import Payload, UnaryClient

from synnax.channel.payload import (
    ChannelKey,
    ChannelKeys,
    ChannelName,
    ChannelNames,
    ChannelParams,
    ChannelPayload,
    normalize_channel_params,
)


class _Request(Payload):
    names: list[str] | None = None
    keys: list[int] | None = None
    leaseholder: int | None = None


class _Response(Payload):
    channels: list[ChannelPayload] = list()
    not_found: list[str] | None = list()


class ChannelRetriever(Protocol):
    """Protocol for retrieving channel payloads from the cluster."""

    def retrieve(self, params: ChannelParams) -> list[ChannelPayload]:
        ...


class ClusterChannelRetriever:
    __ENDPOINT = "/channel/retrieve"
    __client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self.__client = client
        self.instrumentation = instrumentation

    def _(self) -> ChannelRetriever:
        return self

    @trace("debug")
    def retrieve(self, params: ChannelParams) -> list[ChannelPayload]:
        normal = normalize_channel_params(params)
        if len(normal.params) == 0:
            return list()
        return self.__execute(
            _Request(**{normal.variant: normal.params})
        )  # type: ignore

    def __execute(
        self,
        req: _Request,
    ) -> list[ChannelPayload]:
        res, exc = self.__client.send(self.__ENDPOINT, req, _Response)
        if exc is not None:
            raise exc
        return res.channels


class CacheChannelRetriever:
    __retriever: ChannelRetriever
    __channels: dict[ChannelKey, ChannelPayload]
    __names_to_keys: dict[ChannelName, ChannelKey]
    instrumentation: Instrumentation

    def __init__(
        self,
        retriever: ChannelRetriever,
        instrumentation: Instrumentation,
    ) -> None:
        self.__channels = dict()
        self.__names_to_keys = dict()
        self.instrumentation = instrumentation
        self.__retriever = retriever

    def _(self) -> ChannelRetriever:
        return self

    def __get(self, param: ChannelKey | ChannelName) -> ChannelPayload | None:
        if isinstance(param, ChannelKey):
            return self.__channels.get(param)
        key = self.__names_to_keys.get(param)
        return None if key is None else self.__channels.get(key)

    def __set(self, channels: list[ChannelPayload]) -> None:
        for channel in channels:
            self.__channels[channel.key] = channel
            self.__names_to_keys[channel.name] = channel.key

    @trace("debug")
    def retrieve(self, params: ChannelParams) -> list[ChannelPayload]:
        normal = normalize_channel_params(params)
        results = list()
        to_retrieve: ChannelKeys | ChannelNames = list()  # type: ignore
        for p in normal.params:
            ch = self.__get(p)
            if ch is None:
                to_retrieve.append(p)  # type: ignore
            else:
                results.append(ch)

        if len(to_retrieve) == 0:
            return results

        retrieved = self.__retriever.retrieve(to_retrieve)
        self.__set(retrieved)
        results.extend(retrieved)
        return results


def retrieve_required(
    r: ChannelRetriever, params: ChannelParams
) -> list[ChannelPayload]:
    normal = normalize_channel_params(params)
    results = r.retrieve(params)
    not_found = list()
    for p in normal.params:
        ch = next((c for c in results if c.key == p or c.name == p), None)
        if ch is None:
            not_found.append(p)
    if len(not_found) > 0:
        raise RuntimeError(f"Could not find channels: {not_found}")
    return results
