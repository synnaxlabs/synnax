#  Copyright 2026 Synnax Labs, Inc.
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
from freighter import UnaryClient, send_required
from pydantic import BaseModel

from synnax.channel.payload import (
    Key,
    NormalizedChannelNameResult,
    Params,
    Payload,
    normalize_params,
)
from synnax.exceptions import NotFoundError


class _Request(BaseModel):
    names: list[str] | None = None
    keys: list[int] | None = None
    leaseholder: int | None = None


class _Response(BaseModel):
    channels: list[Payload] = list()
    not_found: list[str] | None = list()


class Retriever(Protocol):
    """Protocol for retrieving channel payloads from the cluster."""

    def retrieve(self, channels: Params) -> list[Payload]: ...

    def retrieve_one(self, param: Key | str) -> Payload: ...


class ClusterRetriever:
    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    def _(self) -> Retriever:
        return self

    @trace("debug")
    def retrieve(self, channels: Params) -> list[Payload]:
        normal = normalize_params(channels)
        if len(normal.channels) == 0:
            return list()
        if isinstance(normal, NormalizedChannelNameResult):
            req = _Request(names=normal.channels)
        else:
            req = _Request(keys=normal.channels)
        return self.__exec_retrieve(req)

    @trace("debug")
    def retrieve_one(self, param: Key | str) -> Payload:
        req = _Request()
        if isinstance(param, Key):
            req.keys = [param]
        else:
            req.names = [param]
        channels = self.__exec_retrieve(req)
        if len(channels) == 0:
            raise NotFoundError(f"Could not find channel matching {param}")
        return channels[0]

    def __exec_retrieve(self, req: _Request) -> list[Payload]:
        return send_required(self._client, "/channel/retrieve", req, _Response).channels


class CacheRetriever:
    _retriever: Retriever
    _channels: dict[Key, Payload]
    _names_to_keys: dict[str, set[Key]]
    instrumentation: Instrumentation

    def __init__(
        self,
        retriever: Retriever,
        instrumentation: Instrumentation,
    ) -> None:
        self._channels = dict()
        self._names_to_keys = dict()
        self.instrumentation = instrumentation
        self._retriever = retriever

    def delete(self, keys: Params) -> None:
        normal = normalize_params(keys)
        if isinstance(normal, NormalizedChannelNameResult):
            matches = {
                ch for ch in self._channels.values() if ch.name in normal.channels
            }
            for ch in matches:
                self._channels.pop(ch.key)
                self._names_to_keys.pop(ch.name)
        else:
            for key in normal.channels:
                channel = self._channels.get(key)
                if channel is not None:
                    self._channels.pop(key)
                    self._names_to_keys.pop(channel.name)

    def rename(self, keys: list[Key], names: list[str]) -> None:
        for key, name in zip(keys, names):
            channel = self._channels.get(key)
            if channel is None:
                continue
            self._channels.pop(key)
            existing_keys = self._names_to_keys.get(channel.name)
            if existing_keys is not None:
                existing_keys.remove(key)
            channel.name = name
            self._channels[channel.key] = channel
            existing_keys = self._names_to_keys.get(name)
            if existing_keys is None:
                self._names_to_keys[name] = {channel.key}
            else:
                existing_keys.add(channel.key)

    def _(self) -> Retriever:
        return cast(Retriever, self)

    def _get(self, param: Key | str) -> list[Payload] | None:
        if isinstance(param, Key):
            ch = self._channels.get(param)
            return [ch] if ch is not None else None
        keys = self._names_to_keys.get(param, set())
        channels = list()
        for key in keys:
            ch = self._channels.get(key)
            if ch is not None:
                channels.append(ch)
        return None if len(channels) == 0 else channels

    def _get_one(self, param: Key | str) -> Payload | None:
        if isinstance(param, Key):
            return self._channels.get(param)
        keys = self._names_to_keys.get(param, None)
        if keys is None:
            return None
        return self._channels.get(next(iter(keys)))

    def set(self, channels: list[Payload]) -> None:
        for channel in channels:
            self._set_one(channel)

    def _set_one(self, channel: Payload) -> None:
        self._channels[channel.key] = channel
        keys = self._names_to_keys.get(channel.name)
        if keys is None:
            self._names_to_keys[channel.name] = {channel.key}
        else:
            keys.add(channel.key)

    @trace("debug")
    def retrieve(self, channels: Params) -> list[Payload]:
        normal = normalize_params(channels)
        results = list()
        to_retrieve: list[Key] | tuple[Key] | list[str] | tuple[str] = list()  # type: ignore
        for p in normal.channels:
            ch = self._get(p)
            if ch is None:
                to_retrieve.append(p)  # type: ignore
            else:
                results.extend(ch)

        if len(to_retrieve) == 0:
            return results

        retrieved = self._retriever.retrieve(to_retrieve)
        self.set(retrieved)
        results.extend(retrieved)
        return results

    def retrieve_one(self, param: Key | str) -> Payload:
        ch = self._get_one(param)
        if ch is not None:
            return ch
        retrieved = self._retriever.retrieve_one(param)
        self._set_one(retrieved)
        return retrieved


def retrieve_required(r: Retriever, channels: Params) -> list[Payload]:
    normal = normalize_params(channels)
    results = r.retrieve(channels)
    not_found: list[Key | str] = list()
    if isinstance(normal, NormalizedChannelNameResult):
        for p in normal.channels:
            ch = next((c for c in results if c.name == p), None)
            if ch is None:
                not_found.append(p)
    else:
        for k in normal.channels:
            ch = next((c for c in results if c.key == k), None)
            if ch is None:
                not_found.append(k)
    if len(not_found) > 0:
        raise NotFoundError(f"Could not find channels: {not_found}")
    return results
