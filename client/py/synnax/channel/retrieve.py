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
from synnax.exceptions import NotFoundError


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
    __names_to_keys: dict[ChannelName, set[ChannelKey]]
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

    def delete(self, keys: ChannelParams) -> None:
        normal = normalize_channel_params(keys)
        if normal.variant == "names":
            matches = {
                ch for ch in self.__channels.values() if ch.name in normal.params
            }
            for ch in matches:
                self.__channels.pop(ch.key)
                self.__names_to_keys.pop(ch.name)
        else:
            for key in normal.params:
                channel = self.__channels.get(key)
                if channel is not None:
                    self.__channels.pop(key)
                    self.__names_to_keys.pop(channel.name)

    def rename(self, keys: list[ChannelKey], names: list[ChannelName]) -> None:
        for key, name in zip(keys, names):
            channel = self.__channels.get(key)
            if channel is None:
                continue
            self.__channels.pop(key)
            existing_keys = self.__names_to_keys.get(channel.name)
            if existing_keys is not None:
                existing_keys.remove(key)
            channel.name = name
            self.__channels[channel.key] = channel
            existing_keys = self.__names_to_keys.get(name)
            if existing_keys is None:
                self.__names_to_keys[name] = {channel.key}
            else:
                existing_keys.add(channel.key)

    def _(self) -> ChannelRetriever:
        return self

    def __get(self, param: ChannelKey | ChannelName) -> list[ChannelPayload] | None:
        if isinstance(param, ChannelKey):
            ch = self.__channels.get(param)
            return [ch] if ch is not None else None
        keys = self.__names_to_keys.get(param, set())
        channels = list()
        for key in keys:
            ch = self.__channels.get(key)
            if ch is not None:
                channels.append(ch)
        return None if len(channels) == 0 else channels

    def set(self, channels: list[ChannelPayload]) -> None:
        for channel in channels:
            self.__channels[channel.key] = channel
            keys = self.__names_to_keys.get(channel.name)
            if keys is None:
                self.__names_to_keys[channel.name] = {channel.key}
            else:
                keys.add(channel.key)

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
                results.extend(ch)

        if len(to_retrieve) == 0:
            return results

        retrieved = self.__retriever.retrieve(to_retrieve)
        self.set(retrieved)
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
        raise NotFoundError(f"Could not find channels: {not_found}")
    return results
