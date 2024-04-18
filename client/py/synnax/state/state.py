#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.channel import ChannelKey, ChannelName
from synnax.channel.retrieve import ChannelRetriever, retrieve_required
from synnax.framer import Frame
from synnax.telem import Series


class State:
    value: dict[ChannelKey, Series]
    __retriever: ChannelRetriever

    def __init__(self, retriever: ChannelRetriever):
        self.__retriever = retriever
        self.value = dict()

    def update(self, frame: Frame):
        for key in frame.channels:
            self.value[key] = frame[key]

    def __getitem__(self, ch: ChannelKey):
        ch = retrieve_required(self.__retriever, ch)[0]
        return self.value[ch.key]

    def __getattr__(self, ch: ChannelKey):
        return self.__getitem__(ch)


class LatestState:
    __state: State

    def __init__(self, state: State) -> None:
        self.__state = state

    def __getitem__(self, ch: ChannelKey | ChannelName):
        return self.__state.value[ch][-1]

    def __getattr__(self, ch: ChannelKey | ChannelName):
        return self.__getitem__(ch)
