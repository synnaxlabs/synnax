#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from synnax.channel import Key
from synnax.channel.retrieve import Retriever, retrieve_required
from synnax.framer import Frame
from synnax.telem import MultiSeries


class State:
    value: dict[Key, MultiSeries]
    _retriever: Retriever

    def __init__(self, retriever: Retriever):
        self._retriever = retriever
        self.value = dict()

    def update(self, frame: Frame):
        for key in frame.channels:
            if isinstance(key, int):
                self.value[key] = frame[key]

    def __getitem__(self, ch: Key | str) -> MultiSeries:
        payload = retrieve_required(self._retriever, ch)[0]
        return self.value[payload.key]

    def __getattr__(self, name: str) -> Any:
        return self.__getitem__(name)


class LatestState:
    _state: State

    def __init__(self, state: State) -> None:
        self._state = state

    def __getitem__(self, ch: Key | str) -> Any:
        return self._state[ch][-1]

    def __getattr__(self, name: str) -> Any:
        return self.__getitem__(name)
