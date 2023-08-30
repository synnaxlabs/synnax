#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
from synnax.channel.payload import (
    ChannelKey,
    ChannelName,
    ChannelParams,
    ChannelPayload,
)
from synnax.channel.retrieve import ChannelRetriever
from synnax.channel.payload import normalize_channel_params
from synnax.framer.frame import Frame


class BackwardFrameAdapter:
    __adapter: dict[ChannelKey, ChannelName] | None
    retriever: ChannelRetriever
    keys: list[ChannelKey]

    def __init__(self, retriever: ChannelRetriever):
        self.retriever = retriever
        self.__adapter = None
        self.keys = list()

    def update(self, channels: ChannelParams):
        normal = normalize_channel_params(channels)
        if normal.variant == "keys":
            self.__adapter = None
            self.keys = normal.params
            return

        fetched = self.retriever.retrieve(normal.params)
        self.__adapter = dict[int, str]()
        for name in normal.params:
            ch = next((c for c in fetched if c.name == name), None)
            if ch is None:
                raise KeyError(f"Channel {name} not found.")
            self.__adapter[ch.key] = ch.name
        self.keys = list(self.__adapter.keys())

    def adapt(self, fr: Frame):
        if self.__adapter is None:
            return fr
        keys = [
            self.__adapter[k] if isinstance(k, ChannelKey) else k for k in fr.columns
        ]
        return Frame(columns_or_data=keys, series=fr.series)


class ForwardFrameAdapter:
    __adapter: dict[ChannelName, ChannelKey] | None
    retriever: ChannelRetriever
    __keys: list[ChannelKey] | None

    def __init__(self, retriever: ChannelRetriever):
        self.retriever = retriever
        self.__adapter = None
        self.__keys = None

    def update(self, channels: ChannelParams):
        normal = normalize_channel_params(channels)
        if normal.variant == "keys":
            self.__adapter = None
            self.__keys = normal.params
            return

        fetched = self.retriever.retrieve(normal.params)
        self.__adapter = dict[ChannelName, ChannelKey]()
        for name in normal.params:
            ch = next((c for c in fetched if c.name == name), None)
            if ch is None:
                raise KeyError(f"Channel {name} not found.")
            self.__adapter[ch.name] = ch.key

    @property
    def keys(self):
        return self.__keys or list(self.__adapter.values())

    def adapt(self, fr: Frame):
        if self.__adapter is None:
            return fr
        keys = [
            self.__adapter[k] if isinstance(k, ChannelName) else k for k in fr.columns
        ]

        return Frame(columns_or_data=keys, series=fr.series)
